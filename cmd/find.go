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
	"fmt"
	"net"

	"github.com/spf13/cobra"
	"github.com/sralloza/rpi-provisioner/ssh"
)

type FindArgs struct {
	subnet   string
	user     string
	password string
	port     int
	timeout  int
}

func NewFindCommand() *cobra.Command {
	args := FindArgs{}
	var findCmd = &cobra.Command{
		Use:   "find",
		Short: "Find your raspberry pi in your local network",
		Long:  `Find your raspberry pi in your local network using SSH.`,
		RunE: func(cmd *cobra.Command, posArgs []string) error {
			if err := findHost(args); err != nil {
				return err
			}
			return nil
		},
	}
	findCmd.Flags().StringVar(&args.subnet, "subnet", "", "Subnet to find the raspberry")
	findCmd.Flags().StringVar(&args.user, "user", "pi", "User to login via ssh")
	findCmd.Flags().StringVar(&args.password, "password", "raspberry", "Password to login via ssh")
	findCmd.Flags().IntVar(&args.port, "port", 22, "Port to connect via ssh")
	findCmd.Flags().IntVar(&args.timeout, "timeout", 1, "Password to login via ssh")
	return findCmd
}

func findHost(args FindArgs) error {
	CIDR := args.subnet
	if CIDR == "" {
		defaultCDIR, err := getDefaultCDIR()
		if err != nil {
			return err
		}
		CIDR = defaultCDIR
	}

	fmt.Printf("Getting IP addresses from CIDR %v...\n", CIDR)
	ipv4List, err := getIpsFromCIDR(CIDR)
	if err != nil {
		return err
	}
	fmt.Printf("Found %d IP addresses\n", len(ipv4List))

	fmt.Println("Validating IP addresses...")
	validIPs := findValidSSHHosts(ipv4List, args)
	fmt.Println("Done")

	fmt.Printf("Valid ips: %v\n", validIPs)
	return nil
}

func findValidSSHHosts(ipv4AddrList []net.IP, args FindArgs) []net.IP {
	validIPs := []net.IP{}
	for _, ip := range ipv4AddrList {
		if checkSSHConnection(ip, args) {
			validIPs = append(validIPs, ip)
		}
	}
	return validIPs
}

func checkSSHConnection(ipv4Addr net.IP, args FindArgs) bool {
	connection := ssh.SSHConnection{
		Password:  args.password,
		UseSSHKey: false,
		Debug:     false,
	}
	addr := fmt.Sprintf("%v:%d", ipv4Addr, args.port)
	err := connection.Connect(args.user, addr)
	return err == nil
}
