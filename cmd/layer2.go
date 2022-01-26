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

	"github.com/sralloza/rpi-provisioner/ssh"
	funk "github.com/thoas/go-funk"

	"github.com/spf13/cobra"
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
		Long: `Layer 2 uses the deployer user and bash. It will perform the following tasks:
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
	conn := ssh.SSHConnection{
		UseSSHKey: true,
		Debug:     DebugFlag,
	}
	err := conn.Connect(args.user, address)
	if err != nil {
		return err
	}
	defer conn.Close()

	fmt.Println("Updating and installing system libraries...")
	if err := InstallLibraries(conn); err != nil {
		return err
	}
	fmt.Println("Libraries updated successfully")

	fmt.Println("Provisioning fish...")
	if installed, err := InstallFish(conn, args); err != nil {
		return err
	} else if installed {
		fmt.Println("fish provisioned successfully")
	} else {
		fmt.Println("fish already provisioned")
	}

	fmt.Println("Provisioning oh-my-fish...")
	if installed, err := InstallOhMyFish(conn, args); err != nil {
		return err
	} else if installed {
		fmt.Println("oh-my-fish provisioned successfully")
	} else {
		fmt.Println("oh-my-fish already provisioned")
	}

	fmt.Println("Provisioning docker...")
	if installed, err := InstallDocker(conn, args); err != nil {
		return err
	} else if installed {
		fmt.Println("docker provisioned successfully")
	} else {
		fmt.Println("docker already provisioned")
	}

	fmt.Println("Provisioning docker-compose...")
	if installed, err := InstallDockerCompose(conn, args); err != nil {
		return err
	} else if installed {
		fmt.Println("docker-compose provisioned successfully")
	} else {
		fmt.Println("docker-compose already provisioned")
	}

	return nil
}

func InstallLibraries(conn ssh.SSHConnection) error {
	_, _, err := conn.RunSudo("apt-get update")
	if err != nil {
		return fmt.Errorf("error updating apt registry: %w", err)
	}

	_, _, err = conn.RunSudo("apt-get upgrade -y")
	if err != nil {
		return fmt.Errorf("error upgrading libraries: %w", err)
	}

	libraries := []string{
		"build-essential",
		"cmake",
		"cron",
		"curl",
		"git",
		"libffi-dev",
		"nano",
		"python3-pip",
		"python3",
		"wget",
	}
	installCmd := fmt.Sprintf("apt-get install %s -y", strings.Join(libraries, " "))
	_, _, err = conn.RunSudo(installCmd)
	if err != nil {
		return fmt.Errorf("error installing needed libraries: %w", err)
	}

	return nil
}

func InstallFish(conn ssh.SSHConnection, args Layer2Args) (bool, error) {
	_, _, err := conn.Run("which fish")
	if err == nil {
		return false, nil
	}

	// This line is critical. It doesn't work with conn.RunSudo(xxxx | tee xxx)
	_, _, err = conn.Run("echo 'deb http://download.opensuse.org/repositories/shells:/fish:/release:/3/Debian_10/ /' | sudo tee /etc/apt/sources.list.d/shells:fish:release:3.list")
	if err != nil {
		return false, fmt.Errorf("error adding fish apt registry: %w", err)
	}

	_, _, err = conn.RunSudo("curl -fsSL https://download.opensuse.org/repositories/shells:fish:release:3/Debian_10/Release.key | gpg --dearmor | tee /etc/apt/trusted.gpg.d/shells_fish_release_3.gpg")
	if err != nil {
		return false, fmt.Errorf("error downloading fish apt keys (1): %w", err)
	}

	_, _, err = conn.RunSudo("wget -nv https://download.opensuse.org/repositories/shells:fish:release:3/Debian_10/Release.key -O '/etc/apt/trusted.gpg.d/shells_fish_release_3.asc'")
	if err != nil {
		return false, fmt.Errorf("error downloading fish apt keys (2): %w", err)
	}

	_, _, err = conn.RunSudo("apt update")
	if err != nil {
		return false, fmt.Errorf("error updating apt registry after adding fish registry: %w", err)
	}

	_, _, err = conn.RunSudo("apt install fish -y")
	if err != nil {
		return false, fmt.Errorf("error installing fish: %w", err)
	}

	chshCmd := fmt.Sprintf("chsh -s /usr/bin/fish %s", args.user)
	_, _, err = conn.RunSudo(chshCmd)
	if err != nil {
		return false, fmt.Errorf("error setting deployer's shell to fish: %w", err)
	}

	// Just for debug
	conn.Run("fish --version")
	return true, nil
}

func InstallOhMyFish(conn ssh.SSHConnection, args Layer2Args) (bool, error) {
	_, _, err := conn.Run("omf --version")
	if err == nil {
		return false, nil
	}

	_, _, err = conn.Run("curl -L https://raw.githubusercontent.com/oh-my-fish/oh-my-fish/master/bin/install > /tmp/omf.sh")
	if err != nil {
		return false, fmt.Errorf("error downloading oh-my-fish installer: %w", err)
	}

	rmOmfCmd := fmt.Sprintf("sudo rm -rf /home/%s/.local/share/omf", args.user)
	_, _, err = conn.RunSudo(rmOmfCmd)
	if err != nil {
		return false, fmt.Errorf("couldn't remove omf install dir: %w", err)
	}

	_, _, err = conn.Run("fish /tmp/omf.sh --noninteractive")
	if err != nil {
		return false, fmt.Errorf("error running oh-my-fish installer: %w", err)
	}

	_, _, err = conn.Run("rm /tmp/omf.sh")
	if err != nil {
		return false, fmt.Errorf("error removing oh-my-fish installer: %w", err)
	}

	_, _, err = conn.Run("echo omf install agnoster | fish")
	if err != nil {
		return false, fmt.Errorf("error installing agnorester theme: %w", err)
	}

	_, _, err = conn.Run("echo omf theme agnoster | fish")
	if err != nil {
		return false, fmt.Errorf("error setting angoster theme: %w", err)
	}

	_, _, err = conn.Run("echo omf install bang-bang | fish")
	if err != nil {
		return false, fmt.Errorf("error installing bang-bang plugin: %w", err)
	}

	return true, nil
}

func InstallDocker(conn ssh.SSHConnection, args Layer2Args) (bool, error) {
	_, _, err := conn.Run("which docker")
	if err == nil {
		return false, nil
	}
	_, _, err = conn.Run("curl -fsSL https://get.docker.com -o /tmp/get-docker.sh")
	if err != nil {
		return false, fmt.Errorf("error downloading docker installer: %w", err)
	}

	_, _, err = conn.Run("sudo sh /tmp/get-docker.sh")
	if err != nil {
		return false, fmt.Errorf("error executing docker installer: %w", err)
	}

	_, _, err = conn.Run("rm /tmp/get-docker.sh")
	if err != nil {
		return false, fmt.Errorf("error removing docker installer: %w", err)
	}

	_, _, err = conn.Run(fmt.Sprintf("sudo usermod -aG docker %s", args.user))
	if err != nil {
		return false, fmt.Errorf("error adding deployer to docker group: %w", err)
	}

	return true, nil
}

func InstallDockerCompose(conn ssh.SSHConnection, args Layer2Args) (bool, error) {
	_, _, err := conn.Run("which docker-compose")
	if err == nil {
		return false, nil
	}

	_, _, err = conn.Run("mkdir -p ~/.local/bin")
	if err != nil {
		return false, fmt.Errorf("error creating folder ~/.local/bin: %w", err)
	}

	localBinPath := fmt.Sprintf("/home/%s/.local/bin", args.user)

	paths, _, err := conn.Run("bash -c \"echo $PATH\"")
	if err != nil {
		return false, fmt.Errorf("error getting current path")
	}
	pathList := strings.Split(strings.Trim(paths, "\n"), ":")

	if !funk.Contains(pathList, localBinPath) {
		_, _, err = conn.Run(fmt.Sprintf("echo fish_add_path %s | fish", localBinPath))
		if err != nil {
			return false, fmt.Errorf("error adding folder %q to path: %w", localBinPath, err)
		}
	}

	_, _, err = conn.Run("python3 -m pip install docker-compose")
	if err != nil {
		return false, fmt.Errorf("error installing docker-compose: %w", err)
	}

	return true, nil
}
