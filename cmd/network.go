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

// networkingCmd represents the networking command
var networkingCmd = &cobra.Command{
	Use:   "network",
	Short: "Provision networking",
	Long:  `Set up static ip for eth0 and wlan0`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Provisioning network")
		return networkingEntrypoint(cmd)
	},
}

func networkingEntrypoint(cmd *cobra.Command) error {
	host, err := cmd.Flags().GetString("host")
	if err != nil {
		return err
	}
	if len(host) == 0 {
		return errors.New("must specify --host")
	}

	user, err := cmd.Flags().GetString("user")
	if err != nil {
		return err
	}

	password, err := cmd.Flags().GetString("password")
	if err != nil {
		return err
	}

	usesshKey, err := cmd.Flags().GetBool("ssh-key")
	if err != nil {
		return err
	}
	if !usesshKey && len(password) == 0 {
		return errors.New("must pass --ssh-key or --password")
	}

	port, err := cmd.Flags().GetInt("port")
	if err != nil {
		return err
	}

	ip, err := cmd.Flags().GetIP("ip")
	if err != nil {
		return err
	}

	address := fmt.Sprintf("%s:%d", host, port)

	var auth []ssh.AuthMethod

	if usesshKey {
		auth = append(auth, publicKey(expandPath("~/.ssh/id_rsa")))
	} else {
		auth = append(auth, ssh.Password(password))
	}
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	conn, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return err
	}
	defer conn.Close()

	err = setupNetworking(conn, interfaceArgs{
		ip:       ip,
		password: password,
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
	err := setupStaticIPIface(conn, args, "eth0", 100)
	if err != nil {
		return err
	}

	err = setupStaticIPIface(conn, args, "wlan0", 200)
	if err != nil {
		return err
	}

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

func init() {
	rootCmd.AddCommand(networkingCmd)

	networkingCmd.Flags().Bool("ssh-key", false, "Use ssh key")
	networkingCmd.Flags().String("user", "", "Login user")
	networkingCmd.Flags().String("password", "", "Login password")
	networkingCmd.Flags().String("host", "", "Server host")
	networkingCmd.Flags().Int("port", 22, "Server SSH port")
	networkingCmd.Flags().IP("ip", nil, "New IP")

	networkingCmd.MarkFlagRequired("user")
	networkingCmd.MarkFlagRequired("host")
	networkingCmd.MarkFlagRequired("ip")
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// networkingCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// networkingCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
