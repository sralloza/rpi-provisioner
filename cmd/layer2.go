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
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

type Layer2Args struct {
	user string
	host string
	port int
}

func NewLayer2Cmd() *cobra.Command {
	args := Layer2Args{}
	var layer2Cmd = &cobra.Command{
		Use:   "layer2",
		Short: "Provision layer 2",
		Long: `Layer 2 uses the deployer user and bash. It consists of:
		- Update and upgrade packages
		- Install libraries: build-essential, cmake, cron, curl, git, libffi-dev, nano, python3-pip, python3, wget
		- Install fish
		- Install docker
		`,
		RunE: func(cmd *cobra.Command, posArgs []string) error {
			fmt.Println("Provisioning layer 2")
			if err := ProvisionLayer2(args); err != nil {
				return err
			}
			fmt.Println("Layer 2 provisioned")
			return nil
		},
	}

	layer2Cmd.Flags().StringVar(&args.user, "user", "", "Login user")
	layer2Cmd.Flags().StringVar(&args.host, "host", "", "Server host")
	layer2Cmd.Flags().IntVar(&args.port, "port", 22, "Server SSH port")

	layer2Cmd.MarkFlagRequired("user")
	layer2Cmd.MarkFlagRequired("host")

	return layer2Cmd

}

func ProvisionLayer2(args Layer2Args) error {
	address := fmt.Sprintf("%s:%d", args.host, args.port)
	config := &ssh.ClientConfig{
		User:            args.user,
		Auth:            []ssh.AuthMethod{publicKey("~/.ssh/id_rsa")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	conn, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := InstallLibraries(conn); err != nil {
		return err
	}
	if err := InstallFish(conn, args); err != nil {
		return err
	}
	if err := InstallDocker(conn, args); err != nil {
		return err
	}
	return nil
}

func InstallLibraries(conn *ssh.Client) error {
	_, _, err := runCommand(basicSudoStdin("apt-get update", ""), conn)
	if err != nil {
		return err
	}

	_, _, err = runCommand(basicSudoStdin("apt-get upgrade -y", ""), conn)
	if err != nil {
		return err
	}

	libraries := []string{
		"build-essential",
		"cmake",
		"cron",
		"curl",
		"git",
		"nano",
		"python3-pip",
		"python3",
		"wget",
	}
	installCmd := fmt.Sprintf("apt-get install %s -y", strings.Join(libraries, " "))
	_, _, err = runCommand(basicSudoStdin(installCmd, ""), conn)
	if err != nil {
		return err
	}

	return nil
}

func InstallFish(conn *ssh.Client, args Layer2Args) error {
	_, _, err := runCommand("echo 'deb http://download.opensuse.org/repositories/shells:/fish:/release:/3/Debian_10/ /' | sudo tee /etc/apt/sources.list.d/shells:fish:release:3.list", conn)
	if err != nil {
		return err
	}

	_, _, err = runCommand("curl -fsSL https://download.opensuse.org/repositories/shells:fish:release:3/Debian_10/Release.key | gpg --dearmor | sudo tee /etc/apt/trusted.gpg.d/shells_fish_release_3.gpg", conn)
	if err != nil {
		return err
	}

	_, _, err = runCommand("sudo wget -nv https://download.opensuse.org/repositories/shells:fish:release:3/Debian_10/Release.key -O '/etc/apt/trusted.gpg.d/shells_fish_release_3.asc'", conn)
	if err != nil {
		return err
	}

	_, _, err = runCommand(basicSudoStdin("apt update", ""), conn)
	if err != nil {
		return err
	}

	_, _, err = runCommand(basicSudoStdin("apt install fish -y", ""), conn)
	if err != nil {
		return err
	}

	chshCmd := fmt.Sprintf("chsh -s /usr/bin/fish %s", args.user)
	_, _, err = runCommand(basicSudoStdin(chshCmd, ""), conn)
	if err != nil {
		return err
	}

	// # Oh My Fish
	_, _, err = runCommand("curl -L https://get.oh-my.fish > /tmp/omf.sh", conn)
	if err != nil {
		return err
	}

	_, _, err = runCommand("fish /tmp/omf.sh --noninteractive", conn)
	if err != nil {
		return err
	}

	_, _, err = runCommand("rm /tmp/omf.sh", conn)
	if err != nil {
		return err
	}

	_, _, err = runCommand("ps", conn)
	if err != nil {
		return err
	}

	_, _, err = runCommand("echo omf install agnoster | fish", conn)
	if err != nil {
		return err
	}

	_, _, err = runCommand("echo omf theme agnoster | fish", conn)
	if err != nil {
		return err
	}

	_, _, err = runCommand("echo omf install bang-bang | fish", conn)
	if err != nil {
		return err
	}

	return nil
}

func InstallDocker(conn *ssh.Client, args Layer2Args) error {
	_, _, err := runCommand("curl -fsSL https://get.docker.com -o /tmp/get-docker.sh", conn)
	if err != nil {
		return err
	}

	_, _, err = runCommand("sudo sh /tmp/get-docker.sh", conn)
	if err != nil {
		return err
	}

	_, _, err = runCommand("rm /tmp/get-docker.sh", conn)
	if err != nil {
		return err
	}

	_, _, err = runCommand(fmt.Sprintf("sudo usermod -aG docker %s", args.user), conn)
	if err != nil {
		return err
	}

	_, _, err = runCommand("python3 -m pip install docker-compose", conn)
	if err != nil {
		return err
	}

	_, _, err = runCommand(fmt.Sprintf("echo fish_add_path /home/%s/.local/bin/ | fish", args.user), conn)
	if err != nil {
		return err
	}

	return nil
}
