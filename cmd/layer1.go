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
	"strings"

	"github.com/spf13/cobra"
)

type Layer1Args struct {
	loginUser        string
	loginPassword    string
	deployerUser     string
	deployerPassword string
	rootPassword     string
	host             string
	hostname         string
	port             int
	s3Path           string
	staticIP         net.IP
}

type Layer1Settings struct {
	loginUser        string
	loginPassword    string
	deployerGroup    string
	deployerUser     string
	deployerPassword string
	hostname         string
	s3Bucket         string
	s3File           string
	s3Region         string
	rootPassword     string
}

func NewLayer1Cmd() *cobra.Command {
	args := Layer1Args{}
	var layer1Cmd = &cobra.Command{
		Use:   "layer1",
		Short: "Provision layer 1",
		Long: `Layer 1 uses the default user and bash shell. It consists of:
 - Create deployer user
 - Set hostname
 - Setup ssh config and keys
 - Disable pi login
 - [optional] static ip configuration
 `,
		RunE: func(cmd *cobra.Command, posArgs []string) error {
			fmt.Println("Provisioning layer 1...")
			if err := layer1(args); err != nil {
				return err
			}

			fmt.Println("\nLayer 1 provisioned successfully")
			fmt.Println(
				"Note: you must restart the server to apply the hostname change " +
					"and suppress the security risk warning")
			fmt.Println("\nContinue with layer 2 or ssh into server:")
			fmt.Printf("  ssh %s@%s\n", args.deployerUser, args.host)
			return nil
		},
	}

	layer1Cmd.Flags().StringVar(&args.loginUser, "login-user", "", "Login user")
	layer1Cmd.Flags().StringVar(&args.loginPassword, "login-password", "", "Login password")
	layer1Cmd.Flags().StringVar(&args.deployerPassword, "deployer-user", "", "Deployer user")
	layer1Cmd.Flags().StringVar(&args.deployerUser, "deployer-password", "", "Deployer password")
	layer1Cmd.Flags().StringVar(&args.rootPassword, "root-password", "", "Root password")
	layer1Cmd.Flags().StringVar(&args.host, "host", "", "Server host")
	layer1Cmd.Flags().StringVar(&args.hostname, "hostname", "", "Server hostname")
	layer1Cmd.Flags().IntVar(&args.port, "port", 22, "Server SSH port")
	layer1Cmd.Flags().StringVar(&args.s3Path, "s3-path", "", "Amazon S3 path. Must match the pattern region/bucket/file")
	layer1Cmd.Flags().IPVar(&args.staticIP, "static-ip", nil, "Set up the static ip for eth0 and wlan0")

	layer1Cmd.MarkFlagRequired("login-user")
	layer1Cmd.MarkFlagRequired("login-password")
	layer1Cmd.MarkFlagRequired("deployer-user")
	layer1Cmd.MarkFlagRequired("deployer-password")
	layer1Cmd.MarkFlagRequired("host")
	layer1Cmd.MarkFlagRequired("host-name")
	layer1Cmd.MarkFlagRequired("s3-path")
	return layer1Cmd
}

func layer1(args Layer1Args) error {
	s3Region, s3Bucket, s3File, err := splitAwsPath(args.s3Path)
	if err != nil {
		return err
	}

	address := fmt.Sprintf("%s:%d", args.host, args.port)

	conn := SSHConnection{
		password:  args.loginPassword,
		useSSHKey: false,
	}

	err = conn.Connect(args.loginUser, address)
	if err != nil {
		if strings.Contains(err.Error(), "no supported methods remain") {
			println("SSH Connection error, layer 1 should be provisioned")
			return nil
		}
		return err
	}
	defer conn.close()

	err = setupDeployer(conn, Layer1Settings{
		loginUser:        args.loginUser,
		loginPassword:    args.loginPassword,
		deployerGroup:    args.deployerUser,
		deployerUser:     args.deployerUser,
		deployerPassword: args.deployerPassword,
		hostname:         args.hostname,
		s3Bucket:         s3Bucket,
		s3File:           s3File,
		s3Region:         s3Region,
		rootPassword:     args.rootPassword,
	})
	if err != nil {
		return err
	}

	if len(args.staticIP) != 0 {
		fmt.Printf("Setting up static ip %s\n", args.staticIP)
		setupNetworking(conn, interfaceArgs{
			ip:       args.staticIP,
			password: args.loginPassword,
		})
	}

	return nil
}

func setupDeployer(conn SSHConnection, settings Layer1Settings) error {
	if err := createDeployerGroup(conn, settings); err != nil {
		return err
	}
	if err := createDeployerUser(conn, settings); err != nil {
		return err
	}
	if len(settings.rootPassword) > 0 {
		if err := setRootPassword(conn, settings); err != nil {
			return nil
		}
	}
	if err := uploadsshKeys(conn, UploadsshKeysArgs{
		user:     settings.deployerUser,
		password: settings.loginPassword,
		group:    settings.deployerGroup,
		s3Bucket: settings.s3Bucket,
		s3File:   settings.s3File,
		s3Region: settings.s3Region,
	}); err != nil {
		return err
	}
	if err := setupsshdConfig(conn, settings); err != nil {
		return err
	}
	if err := setHostname(conn, settings); err != nil {
		return err
	}
	if err := disableLoginUser(conn, settings); err != nil {
		return err
	}
	return nil
}

func createDeployerGroup(conn SSHConnection, settings Layer1Settings) error {
	grepCmd := fmt.Sprintf("grep -q %s /etc/group", settings.deployerGroup)
	_, _, err := conn.run(grepCmd)

	if err == nil {
		fmt.Println("Deployer group already exists")
	} else {
		groupaddCmd := fmt.Sprintf("groupadd %s", settings.deployerGroup)
		stdout, stderr, err := conn.runSudoPassword(groupaddCmd, settings.loginPassword)
		if err != nil {
			return fmt.Errorf("error creating deployer group: %s [%s %s]", err, stdout, stderr)
		}
		fmt.Println("Deployer group created")
	}

	fmt.Println("Checking sudo access")
	_, _, err = conn.runSudoPassword("whoami", settings.loginPassword)
	if err != nil {
		return nil
	}
	fmt.Println("Updating sudoers file")
	_, _, err = conn.runSudoPassword("cp /etc/sudoers /etc/sudoers.bkp", settings.loginPassword)
	if err != nil {
		return err
	}

	initialSudoers, _, err := conn.runSudoPassword("cat /etc/sudoers", settings.loginPassword)
	if err != nil {
		return err
	}
	initialSudoers = strings.Trim(initialSudoers, "\n\r")

	extraSudoer := fmt.Sprintf("%s ALL=(ALL) NOPASSWD: ALL", settings.deployerGroup)
	if strings.Contains(initialSudoers, extraSudoer) {
		fmt.Println("Sudoer already setup")
		return nil
	}

	sudoersCmd := fmt.Sprintf("echo \"\n%s\n\" >> /etc/sudoers", extraSudoer)
	_, _, err = conn.runSudoPassword(sudoersCmd, settings.loginPassword)
	if err != nil {
		return err
	}
	// sudoers = sudoers.encode("utf8").replace(b"\r\n", b"\n")

	return nil
}

func createDeployerUser(conn SSHConnection, settings Layer1Settings) error {
	fmt.Println("Creating deployer user")
	_, _, err := conn.run("id " + settings.deployerUser)
	if err == nil {
		fmt.Println("Deployer user already created")
		return nil
	}

	useraddCmd := fmt.Sprintf("useradd -m -c 'deployer' -s /bin/bash -g '%s' ", settings.deployerGroup)
	useraddCmd += settings.deployerUser
	_, _, err = conn.runSudoPassword(useraddCmd, settings.loginPassword)
	if err != nil {
		return err
	}

	chpasswdCmd := fmt.Sprintf("echo %s:%s | chpasswd", settings.deployerUser, settings.deployerPassword)
	_, _, err = conn.runSudoPassword(chpasswdCmd, settings.loginPassword)
	if err != nil {
		return err
	}

	usermodCmd := fmt.Sprintf("usermod -a -G %s %s", settings.deployerGroup, settings.deployerUser)
	_, _, err = conn.runSudoPassword(usermodCmd, settings.loginPassword)
	if err != nil {
		return err
	}

	mkdirsshCmd := fmt.Sprintf("mkdir /home/%s/.ssh", settings.deployerUser)
	_, _, err = conn.runSudoPassword(mkdirsshCmd, settings.loginPassword)
	if err != nil {
		return err
	}

	chownCmd := fmt.Sprintf("chown -R %s:%s /home/%s", settings.deployerUser, settings.deployerGroup, settings.deployerUser)
	_, _, err = conn.runSudoPassword(chownCmd, settings.loginPassword)
	if err != nil {
		return err
	}

	return nil
}

func setRootPassword(conn SSHConnection, settings Layer1Settings) error {
	fmt.Println("Setting root password")
	chpasswdCmd := fmt.Sprintf("echo root:%s | chpasswd", settings.rootPassword)
	_, _, err := conn.runSudoPassword(chpasswdCmd, settings.loginPassword)
	if err != nil {
		return err
	}
	return nil
}

func setupsshdConfig(conn SSHConnection, settings Layer1Settings) error {
	config := "/etc/ssh/sshd_config"

	backupCmd := fmt.Sprintf("cp %s %s.backup", config, config)
	_, _, err := conn.runSudoPassword(backupCmd, settings.loginPassword)
	if err != nil {
		return err
	}

	usePamCmd := fmt.Sprintf("sed -i \"s/^UsePAM yes/UsePAM no/\" %s", config)
	_, _, err = conn.runSudoPassword(usePamCmd, settings.loginPassword)
	if err != nil {
		return err
	}

	permitRootLoginCmd := fmt.Sprintf("sed -i \"s/^PermitRootLogin yes/PermitRootLogin no/\" %s", config)
	_, _, err = conn.runSudoPassword(permitRootLoginCmd, settings.loginPassword)
	if err != nil {
		return err
	}

	passwordAuthCmd := fmt.Sprintf("sed -i \"s/^#PasswordAuthentication yes/PasswordAuthentication no/\" %s", config)
	_, _, err = conn.runSudoPassword(passwordAuthCmd, settings.loginPassword)
	if err != nil {
		return err
	}

	_, _, err = conn.runSudoPassword("service ssh reload", settings.loginPassword)
	if err != nil {
		return err
	}

	return nil
}

func setHostname(conn SSHConnection, settings Layer1Settings) error {
	println("Setting hostname")
	hostnameCmd := fmt.Sprintf("echo \"%s\" > /etc/hostname", settings.hostname)
	_, _, err := conn.runSudoPassword(hostnameCmd, settings.loginPassword)
	if err != nil {
		return err
	}

	hostCmd := fmt.Sprintf("echo \"127.0.0.1\t\t%s\" >> /etc/hosts", settings.hostname)
	_, _, err = conn.runSudoPassword(hostCmd, settings.loginPassword)
	if err != nil {
		return err
	}

	return nil
}

func disableLoginUser(conn SSHConnection, settings Layer1Settings) error {
	passwdCmd := fmt.Sprintf("passwd -d %s", settings.loginUser)
	_, _, err := conn.runSudoPassword(passwdCmd, settings.loginPassword)
	if err != nil {
		return err
	}

	usermodCmd := fmt.Sprintf("usermod -s /usr/sbin/nologin %s", settings.loginUser)
	_, _, err = conn.runSudoPassword(usermodCmd, settings.loginPassword)
	if err != nil {
		return err
	}
	return nil
}
