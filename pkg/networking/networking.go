package networking

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/sralloza/rpi-provisioner/pkg/info"
	"github.com/sralloza/rpi-provisioner/pkg/logging"
	"github.com/sralloza/rpi-provisioner/ssh"
)

type NetworkingArgs struct {
	UseSSHKey bool
	User      string
	Password  string
	Host      string
	Port      int
	Ip        net.IP
}

func NewNetworkingManager() *networkingManager {
	return &networkingManager{}
}

type networkingManager struct {
	conn ssh.SSHConnection
}

func (n *networkingManager) Setup(args NetworkingArgs) error {
	if !args.UseSSHKey && len(args.Password) == 0 {
		return errors.New("must pass --ssh-key or --password")
	}

	err := n.connect(args.User, args.Password, args.Host, args.Port, args.UseSSHKey)
	if err != nil {
		return err
	}
	defer n.conn.Close()

	info.Title("Provisioning static ip %s", args.Ip)
	if provisioned, err := SetupNetworking(n.conn, args.Ip, args.Password); err != nil {
		info.Fail()
		return err
	} else if provisioned {
		info.Ok()
	} else {
		info.Skipped()
	}

	return nil
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

func SetupNetworking(conn ssh.SSHConnection, ip net.IP, password string) (bool, error) {
	// the lowest metric has priority -> eth0
	eth0Provisioned, err := provisionStaticIPIface(conn, ip, password, "eth0", 100)
	if err != nil {
		return false, fmt.Errorf("error provisioning static IP for eth0: %w", err)
	}

	wlan0Provisioned, err := provisionStaticIPIface(conn, ip, password, "wlan0", 200)
	if err != nil {
		return false, fmt.Errorf("error provisioning static IP for wlan0: %w", err)
	}

	if !eth0Provisioned && !wlan0Provisioned {
		return false, nil
	}

	err = rebootNetworkd(conn, password)
	if err != nil {
		return false, fmt.Errorf("error rebooting networkd service: %w", err)
	}

	return true, nil
}

func getIfaceToConMap(conn ssh.SSHConnection) (map[string]string, error) {
	stdout, _, err := conn.Run("nmcli con show")
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

	logger := logging.Get()
	logger.Debug().Interface("data", data).Msg("network interfaces")
	return data, nil
}

func provisionStaticIPIface(conn ssh.SSHConnection, ip net.IP, password, iface string, metric int) (bool, error) {
	routerIP, err := getRouterIP(conn)
	if err != nil {
		return false, fmt.Errorf("error getting router IP: %w", err)
	}

	ifaceToConMap, err := getIfaceToConMap(conn)
	if err != nil {
		return false, fmt.Errorf("error getting map of devices to interfaces: %w", err)
	}

	conId, ok := ifaceToConMap[iface]
	if !ok {
		logger := logging.Get()
		logger.Warn().Str("iface", iface).Msg("Skipping iface, not found in nmcli con show")
		return false, nil
	}

	nmcliUpdateCmd := fmt.Sprintf(
		"nmcli con mod %s ipv4.addresses %s/24 ipv4.gateway %s ipv4.dns 1.1.1.1 ipv4.method manual connection.autoconnect yes ipv4.route-metric %d",
		conId, ip, routerIP, metric,
	)
	_, _, err = conn.RunSudoPassword(nmcliUpdateCmd, password)
	if err != nil {
		return false, fmt.Errorf("error updating network configuration: %w", err)
	}
	return true, nil
}

func getRouterIP(conn ssh.SSHConnection) (net.IP, error) {
	iprCmd := "ip r | grep default"
	data, _, err := conn.Run(iprCmd)
	if err != nil {
		return nil, fmt.Errorf("error executing ip r command: %w", err)
	}

	splitted := strings.Split(data, " ")
	if len(splitted) < 2 {
		return nil, fmt.Errorf("can't find router ip ('%s'=%#v)", iprCmd, data)
	}
	return net.ParseIP(splitted[2]), nil
}

func generateStaticDHCPConfiguration(iface string, IP net.IP, routerIP net.IP, metric int) string {
	// 	template := `
	// [Match]
	// Name=%s

	// [Network]
	// Address=%s/24
	// Gateway=%s
	// DNS=1.1.1.1

	// [Route]
	// Gateway=%s
	// Metric=%d
	// DHCP=no
	// `
	template := `
auto %s
iface %s inet static
address %s
netmask 255.255.255.0

`
	return fmt.Sprintf(template, iface, iface, IP, routerIP, routerIP, metric)
}

func rebootNetworkd(conn ssh.SSHConnection, password string) error {
	_, _, err := conn.RunSudoPassword("systemctl restart NetworkManager", password)
	if err != nil {
		return err
	}
	return nil
}

// func ensureNetworkdEnabled(conn ssh.SSHConnection, password string) error {
// 	_, _, err := conn.RunSudoPassword("systemctl enable systemd-networkd", password)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }
