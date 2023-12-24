package networking

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"

	"slices"

	"github.com/rs/zerolog"
	"github.com/sralloza/rpi-provisioner/pkg/info"
	"github.com/sralloza/rpi-provisioner/pkg/logging"
	"github.com/sralloza/rpi-provisioner/pkg/ssh"
)

type NetworkingArgs struct {
	UseSSHKey bool
	User      string
	Password  string
	Host      string
	Port      int
	IpAddress net.IP
}

func NewNetworkingManager() *networkingManager {
	return &networkingManager{
		log: logging.Get(),
	}
}

type networkingManager struct {
	conn ssh.SSHConnection
	log  *zerolog.Logger
}

type NetworkProvisionResult struct {
	Provisioned               bool
	NeedRestartForDHCPCleanup bool
}

func SetupNetworking(conn ssh.SSHConnection, primaryIP net.IP, password, host string) (NetworkProvisionResult, error) {
	manager := NewNetworkingManager()
	manager.conn = conn
	return manager.setupNetworking(primaryIP, password)
}

func (n *networkingManager) Setup(args NetworkingArgs) (NetworkProvisionResult, error) {
	result := NetworkProvisionResult{
		Provisioned:               false,
		NeedRestartForDHCPCleanup: false,
	}

	if !args.UseSSHKey && len(args.Password) == 0 {
		return result, errors.New("must pass --ssh-key or --password")
	}

	err := n.connect(args.User, args.Password, args.Host, args.Port, args.UseSSHKey)
	if err != nil {
		return result, err
	}
	defer n.conn.Close()

	info.Title("Provisioning static IP %s", args.IpAddress)

	result, err = n.setupNetworking(args.IpAddress, args.Password)
	if err != nil {
		info.Fail()
		return result, err
	} else if result.Provisioned {
		info.Ok()
	} else {
		info.Skipped()
	}

	return result, nil
}

func (n *networkingManager) connect(user, password, host string, port int, useSSHKey bool) error {
	n.conn = ssh.SSHConnection{
		Password:  password,
		UseSSHKey: useSSHKey,
	}

	err := n.conn.Connect(user, fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return fmt.Errorf("error connecting to %s:%d: %w", host, port, err)
	}

	return nil
}

func (n *networkingManager) setupNetworking(ipAddress net.IP, password string) (NetworkProvisionResult, error) {
	result := NetworkProvisionResult{
		Provisioned:               false,
		NeedRestartForDHCPCleanup: false,
	}

	// The lower the metric, the higher the priority
	eth0Provisioned, err := n.provisionStaticIPIface(ipAddress, password, "eth0", 100)
	if err != nil {
		return result, fmt.Errorf("error provisioning static IP for eth0: %w", err)
	}

	wlan0Provisioned, err := n.provisionStaticIPIface(ipAddress, password, "wlan0", 200)
	if err != nil {
		return result, fmt.Errorf("error provisioning static IP for wlan0: %w", err)
	}

	if eth0Provisioned || wlan0Provisioned {
		err = n.restartNetworkManager(password)
		if err != nil {
			return result, fmt.Errorf("error restarting NetworkManager service: %w", err)
		}
		result.Provisioned = true
	}

	// Delete old DHCP IPs
	eth0Cleaned, err := n.deleteDhcpIps(password, "eth0", ipAddress)
	if err != nil {
		result.NeedRestartForDHCPCleanup = true
		return result, nil
	}

	wlan0Cleaned, err := n.deleteDhcpIps(password, "wlan0", ipAddress)
	if err != nil {
		result.NeedRestartForDHCPCleanup = true
		return result, nil
	}

	if eth0Cleaned || wlan0Cleaned {
		err = n.restartNetworkManager(password)
		if err != nil {
			return result, fmt.Errorf("error restarting NetworkManager service: %w", err)
		}
		result.Provisioned = true
	}

	return result, nil
}

func (n *networkingManager) getIfaceToConMap() (map[string]string, error) {
	stdout, _, err := n.conn.Run("nmcli con show")
	if err != nil {
		return nil, fmt.Errorf("error getting network interfaces: %w", err)
	}

	r := regexp.MustCompile(`([\w ]+)\s+([a-z0-9-]+)\s+(\w+)\s+(\w+)`)

	lines := strings.Split(stdout, "\n")
	data := make(map[string]string)
	for i, line := range lines {
		if i == 0 || line == "" {
			continue
		}
		matches := r.FindStringSubmatch(line)
		if len(matches) < 5 {
			return nil, fmt.Errorf("error parsing network interfaces: %s", line)
		}

		data[matches[4]] = matches[2]
	}

	n.log.Debug().Interface("data", data).Msg("network interfaces")
	return data, nil
}

func (n *networkingManager) provisionStaticIPIface(ip net.IP, password, iface string, metric int) (bool, error) {
	routerIP, err := n.getRouterIP()
	if err != nil {
		return false, fmt.Errorf("error getting router IP: %w", err)
	}

	ifaceToConMap, err := n.getIfaceToConMap()
	if err != nil {
		return false, fmt.Errorf("error getting map of devices to interfaces: %w", err)
	}

	conId, ok := ifaceToConMap[iface]
	if !ok {
		n.log.Warn().Str("iface", iface).Msg("Skipping iface, not found in nmcli con show")
		return false, nil
	}

	nmcliUpdateCmd := fmt.Sprintf(
		"nmcli con mod %s ipv4.addresses %s/24 ipv4.gateway %s ipv4.dns 1.1.1.1 ipv4.method manual connection.autoconnect yes ipv4.route-metric %d",
		conId, ip, routerIP, metric,
	)
	_, _, err = n.conn.RunSudoPassword(nmcliUpdateCmd, password)
	if err != nil {
		return false, fmt.Errorf("error updating network configuration: %w", err)
	}

	return true, nil
}

func (n *networkingManager) deleteDhcpIps(password, iface string, realIp net.IP) (bool, error) {
	routes, _, err := n.conn.Run("ip route")
	if err != nil {
		return false, fmt.Errorf("error getting IP routes: %w", err)
	}
	dhcpIps := []string{}
	ipRegexp := regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)
	// Line example:
	// "default via 192.168.0.1 dev eth0 proto dhcp src 192.168.0.70 metric 100"
	for _, route := range strings.Split(routes, "\n") {
		if strings.Contains(route, "dhcp") && strings.Contains(route, iface) {
			// Avoid getting the gateway IP
			parts := strings.Split(route, "dhcp")
			result := ipRegexp.FindString(parts[1])
			dhcpIps = append(dhcpIps, result)
		}
	}

	n.log.Debug().Str("iface", iface).Strs("dhcpIps", dhcpIps).Msgf("Found %d DHCP IPs", len(dhcpIps))

	for _, ip := range dhcpIps {
		cmd := fmt.Sprintf("ip addr del %s/32 dev %s", ip, iface)
		_, stderr, err := n.conn.RunSudoPassword(cmd, password)
		if err != nil {
			if stderr == "RTNETLINK answers: Cannot assign requested address\n" {
				return false, fmt.Errorf("error deleting old DHCP IPs, try rebooting the device")
			}

			return false, fmt.Errorf(
				"error deleting IP %s for interface %s [%w]: %s", ip, iface, err, stderr)
		}
	}

	// Detect if there is more than 1 IP in total. Example:
	// "192.168.0.0/24 dev eth0 proto kernel scope link src 192.168.0.158 metric 100 "

	r := regexp.MustCompile(`src (\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}) metric`)
	kernelScopeIps := []string{}
	matches := r.FindAllStringSubmatch(routes, -1)
	for _, match := range matches {
		ipAddress := match[1]
		if !slices.Contains(dhcpIps, ipAddress) {
			kernelScopeIps = append(kernelScopeIps, ipAddress)
		}
	}

	n.log.Debug().Str("iface", iface).Strs("kernelScopeIps", kernelScopeIps).Msgf("Found %d kernel scope IPs", len(kernelScopeIps))
	if len(kernelScopeIps) > 1 {
		return false, fmt.Errorf("error deleting old DHCP IPs, try rebooting the device")
	}

	return true, nil
}

func (n *networkingManager) getRouterIP() (net.IP, error) {
	iprCmd := "ip r | grep default"
	data, _, err := n.conn.Run(iprCmd)
	if err != nil {
		return nil, fmt.Errorf("error executing ip r command: %w", err)
	}

	splitted := strings.Split(data, " ")
	if len(splitted) < 2 {
		return nil, fmt.Errorf("can't find router ip ('%s'=%#v)", iprCmd, data)
	}
	return net.ParseIP(splitted[2]), nil
}

func (n *networkingManager) restartNetworkManager(password string) error {
	_, _, err := n.conn.RunSudoPassword("systemctl restart NetworkManager", password)
	if err != nil {
		return err
	}
	return nil
}
