/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/spf13/cobra"
)

type NetworkingArgs struct {
	useSSHKey bool
	user      string
	password  string
	host      string
	port      int
	ip        net.IP
}

func NewNetworkingCmd() *cobra.Command {
	args := NetworkingArgs{}

	var networkingCmd = &cobra.Command{
		Use:   "network",
		Short: "Provision networking",
		Long:  `Set up static ip for eth0 and wlan0.`,
		RunE: func(cmd *cobra.Command, posArgs []string) error {
			fmt.Println("Provisioning network")
			if err := networkingEntrypoint(args); err != nil {
				return err
			}
			ip, _ := cmd.Flags().GetIP("ip")
			fmt.Println("Network provisioned successfully")
			fmt.Println("Static ip: " + ip.String())
			return nil
		},
	}

	networkingCmd.Flags().BoolVar(&args.useSSHKey, "ssh-key", false, "Use ssh key")
	networkingCmd.Flags().StringVar(&args.user, "user", "", "Login user")
	networkingCmd.Flags().StringVar(&args.password, "password", "", "Login password")
	networkingCmd.Flags().StringVar(&args.host, "host", "", "Server host")
	networkingCmd.Flags().IntVar(&args.port, "port", 22, "Server SSH port")
	networkingCmd.Flags().IPVar(&args.ip, "ip", nil, "New IP")

	networkingCmd.MarkFlagRequired("user")
	networkingCmd.MarkFlagRequired("host")
	networkingCmd.MarkFlagRequired("ip")
	return networkingCmd
}

func networkingEntrypoint(args NetworkingArgs) error {
	if !args.useSSHKey && len(args.password) == 0 {
		return errors.New("must pass --ssh-key or --password")
	}

	address := fmt.Sprintf("%s:%d", args.host, args.port)

	conn := SSHConnection{
		password:  args.password,
		useSSHKey: args.useSSHKey,
	}

	err := conn.Connect(args.user, address)
	if err != nil {
		return err
	}
	defer conn.close()

	fmt.Printf("Provisioning static ip %s...\n", args.ip)
	if provisioned, err := setupNetworking(conn, interfaceArgs{
		ip:       args.ip,
		password: args.password,
	}); err != nil {
		return err
	} else if provisioned {
		fmt.Println("Provisioned static IP")
	} else {
		fmt.Println("Static IP already provisioned")
	}

	return nil
}

type interfaceArgs struct {
	ip       net.IP
	password string
}

func setupNetworking(conn SSHConnection, args interfaceArgs) (bool, error) {
	// the lowest metric has priority -> eth0
	eth0Provisioned, err := provisionStaticIPIface(conn, args, "eth0", 100)
	if err != nil {
		return false, fmt.Errorf("error provisioning static IP for eth0: %w", err)
	}

	wlan0Provisioned, err := provisionStaticIPIface(conn, args, "wlan0", 200)
	if err != nil {
		return false, fmt.Errorf("error provisioning static IP for wlan0", err)
	}

	if !eth0Provisioned && !wlan0Provisioned {
		return false, nil
	}

	err = rebootdhcpd(conn, args.password)
	if err != nil {
		return false, fmt.Errorf("error rebooting DHCP service: %w", err)
	}

	return true, nil
}

func provisionStaticIPIface(conn SSHConnection, args interfaceArgs, iface string, metric int) (bool, error) {
	routerIP, err := getRouterIP(conn)
	if err != nil {
		return false, fmt.Errorf("error getting router IP: %w", err)
	}
	dhcpConfiguration := generateStaticDHCPConfiguration(iface, args.ip, routerIP, metric)
	if err != nil {
		return false, fmt.Errorf("error generating static DHCP conf for %q: %w", iface, err)
	}

	// TODO: override interface settings (detect start and end)
	if strings.Contains(fmt.Sprintf("interface %s", iface), dhcpConfiguration) {
		return false, nil
	}

	catCmd := fmt.Sprintf("echo \"%s\" >> /etc/dhcpcd.conf", dhcpConfiguration)
	_, _, err = conn.runSudoPassword(catCmd, args.password)
	if err != nil {
		return false, fmt.Errorf("error updating DHCP configuration: %w", err)
	}
	return true, nil
}

func getRouterIP(conn SSHConnection) (net.IP, error) {
	iprCmd := "ip r | grep default"
	data, _, err := conn.run(iprCmd)
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
	template := `
interface %s
static ip_address=%s/24
static routers=%s
static domain_name_servers=1.1.1.1
metric %d
`
	return fmt.Sprintf(template, iface, IP, routerIP, metric)
}

func rebootdhcpd(conn SSHConnection, password string) error {
	_, _, err := conn.runSudoPassword("systemctl restart dhcpcd.service", password)
	if err != nil {
		return err
	}
	return nil
}
