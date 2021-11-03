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
	"golang.org/x/crypto/ssh"
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
		Long:  `Set up static ip for eth0 and wlan0`,
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

	var auth []ssh.AuthMethod

	if args.useSSHKey {
		auth = append(auth, publicKey(expandPath("~/.ssh/id_rsa")))
	} else {
		auth = append(auth, ssh.Password(args.password))
	}
	config := &ssh.ClientConfig{
		User:            args.user,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	conn, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return err
	}
	defer conn.Close()

	err = setupNetworking(conn, interfaceArgs{
		ip:       args.ip,
		password: args.password,
	})
	if err != nil {
		return err
	}
	return nil
}

type interfaceArgs struct {
	ip       net.IP
	password string
}

func setupNetworking(conn *ssh.Client, args interfaceArgs) error {
	// the lowest metric has priority -> eth0
	fmt.Println("Setting static ip for interface eth0")
	err := setupStaticIPIface(conn, args, "eth0", 100)
	if err != nil {
		return err
	}

	fmt.Println("Setting static ip for interface wlan0")
	err = setupStaticIPIface(conn, args, "wlan0", 200)
	if err != nil {
		return err
	}

	fmt.Println("Restarting DHCP")
	err = rebootdhcpd(conn, args.password)
	if err != nil {
		return err
	}

	return nil
}

func setupStaticIPIface(conn *ssh.Client, args interfaceArgs, iface string, metric int) error {
	routerIP, err := getRouterIP(conn)
	if err != nil {
		return err
	}
	dhcpConfiguration := generateStaticDHCPConfiguration(iface, args.ip, routerIP, metric)

	catCmd := fmt.Sprintf("echo \"%s\" >> /etc/dhcpcd.conf", dhcpConfiguration)
	_, _, err = runCommand(basicSudoStdin(catCmd, args.password), conn)
	if err != nil {
		return err
	}
	return nil
}

func getRouterIP(conn *ssh.Client) (net.IP, error) {
	iprCmd := "ip r | grep default"
	data, _, err := runCommand(iprCmd, conn)
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

func rebootdhcpd(conn *ssh.Client, password string) error {
	_, _, err := runCommand(basicSudoStdin("systemctl restart dhcpcd.service", password), conn)
	if err != nil {
		return err
	}
	return nil
}
