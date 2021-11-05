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
	keysPath         string
	staticIP         net.IP
}

func NewLayer1Cmd() *cobra.Command {
	args := Layer1Args{}
	var layer1Cmd = &cobra.Command{
		Use:   "layer1",
		Short: "Provision layer 1",
		Long: `Layer 1 uses the default user and bash shell. It will perform the following tasks:
 - Create deployer user
 - Set hostname
 - Setup ssh config and keys
 - Disable pi login
 - [optional] static ip configuration
 `,
		PreRunE: func(cmd *cobra.Command, posArgs []string) error {
			if len(args.keysPath) != 0 && len(args.s3Path) != 0 {
				return errors.New("must pass one of --keys-path or --s3-path")
			}
			if len(args.keysPath) == 0 && len(args.s3Path) == 0 {
				return errors.New("must pass one of --keys-path or --s3-path")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, posArgs []string) error {
			fmt.Println("Provisioning layer 1...")
			if err := ProvisionLayer1(args); err != nil {
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
	layer1Cmd.Flags().StringVar(&args.keysPath, "keys-path", "", "Local keys file path. You can select the public key file or a file containing multiple public keys.")
	layer1Cmd.Flags().IPVar(&args.staticIP, "static-ip", nil, "Set up the static ip for eth0 and wlan0")

	layer1Cmd.MarkFlagRequired("login-user")
	layer1Cmd.MarkFlagRequired("login-password")
	layer1Cmd.MarkFlagRequired("deployer-user")
	layer1Cmd.MarkFlagRequired("deployer-password")
	layer1Cmd.MarkFlagRequired("host")
	layer1Cmd.MarkFlagRequired("host-name")
	return layer1Cmd
}

func ProvisionLayer1(args Layer1Args) error {
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
			fmt.Println("SSH Connection error, layer 1 should be provisioned")
			return nil
		}
		return fmt.Errorf("SSH connection error: %w", err)
	}
	defer conn.close()

	fmt.Println("Creating deployer group...")
	if provisioned, err := createDeployerGroup(conn, args); err != nil {
		return err
	} else if provisioned {
		fmt.Println("Deployer group created")
	} else {
		fmt.Println("Deployer group already created")
	}

	fmt.Println("Provisioning deployer sudo access...")
	if provisioned, err := provisionSudoer(conn, args); err != nil {
		return err
	} else if provisioned {
		fmt.Println("Deployer sudo access provisioned")
	} else {
		fmt.Println("Deployer sudo access already provisioned")
	}

	fmt.Println("Creating deployer user...")
	if provisioned, err := createDeployerUser(conn, args); err != nil {
		return err
	} else if provisioned {
		fmt.Println("Deployer user created")
	} else {
		fmt.Println("Deployer user already created")
	}

	if len(args.rootPassword) > 0 {
		fmt.Println("Provisioning sudo password...")
		if provisioned, err := setRootPassword(conn, args); err != nil {
			return nil
		} else if provisioned {
			fmt.Println("Root password provisioned")
		} else {
			fmt.Println("Root password already provisioned")
		}
	}

	fmt.Println("Provisioning SSH keys...")
	if provisioned, err := uploadsshKeys(conn, UploadsshKeysArgs{
		user:     args.deployerUser,
		password: args.loginPassword,
		group:    args.deployerUser,
		s3Bucket: s3Bucket,
		s3File:   s3File,
		s3Region: s3Region,
		keysPath: args.keysPath,
	}); err != nil {
		return err
	} else if provisioned {
		fmt.Println("SSH keys provisioned")
	} else {
		fmt.Println("SSH keys already provisioned")
	}

	fmt.Println("Configuring SSHD...")
	if provisioned, err := setupsshdConfig(conn, args); err != nil {
		return err
	} else if provisioned {
		fmt.Println("SSHD configured")
	} else {
		fmt.Println("SSHD already configured")
	}

	fmt.Println("Provisioning hostname...")
	if provisioned, err := setHostname(conn, args); err != nil {
		return err
	} else if provisioned {
		fmt.Println("Hostname provisioned")
	} else {
		fmt.Println("Hostname already provisioned")
	}

	fmt.Println("Disable loginUser login...")
	if provisioned, err := disableLoginUser(conn, args); err != nil {
		return err
	} else if provisioned {
		fmt.Println("LoginUser login disabled")
	} else {
		fmt.Println("LoginUser login already disabled")
	}

	if len(args.staticIP) != 0 {
		fmt.Printf("Provisioning static ip %s...\n", args.staticIP)
		if provisioned, err := setupNetworking(conn, interfaceArgs{
			ip:       args.staticIP,
			password: args.loginPassword,
		}); err != nil {
			return err
		} else if provisioned {
			fmt.Println("Static IP provisioned")
		} else {
			fmt.Println("Static IP already provisioned")
		}
	}

	return nil
}

func createDeployerGroup(conn SSHConnection, args Layer1Args) (bool, error) {
	grepCmd := fmt.Sprintf("grep -q %s /etc/group", args.deployerUser)
	_, _, err := conn.run(grepCmd)

	if err == nil {
		return false, nil
	}
	groupaddCmd := fmt.Sprintf("groupadd %s", args.deployerUser)
	stdout, stderr, err := conn.runSudoPassword(groupaddCmd, args.loginPassword)
	if err != nil {
		return false, fmt.Errorf("error creating deployer group: %s [%s %s]", err, stdout, stderr)
	}
	return true, nil
}

func provisionSudoer(conn SSHConnection, args Layer1Args) (bool, error) {
	initialSudoers, _, err := conn.runSudoPassword("cat /etc/sudoers", args.loginPassword)
	if err != nil {
		return false, fmt.Errorf("error getting sudoers info: %w", err)
	}
	initialSudoers = strings.Trim(initialSudoers, "\n\r")

	extraSudoer := fmt.Sprintf("%s ALL=(ALL) NOPASSWD: ALL", args.deployerUser)
	if strings.Contains(initialSudoers, extraSudoer) {
		return false, nil
	}
	_, _, err = conn.runSudoPassword("cp /etc/sudoers /etc/sudoers.bkp", args.loginPassword)
	if err != nil {
		return false, fmt.Errorf("error creating sudoers backup: %w", err)
	}

	sudoersCmd := fmt.Sprintf("echo \"\n%s\n\" >> /etc/sudoers", extraSudoer)
	_, _, err = conn.runSudoPassword(sudoersCmd, args.loginPassword)
	if err != nil {
		return false, fmt.Errorf("error updating sudoers: %w", err)
	}
	return true, nil
}

func createDeployerUser(conn SSHConnection, args Layer1Args) (bool, error) {
	_, _, err := conn.run("id " + args.deployerUser)
	if err == nil {
		return false, nil
	}

	useraddCmd := fmt.Sprintf("useradd -m -c 'deployer' -s /bin/bash -g '%s' ", args.deployerUser)
	useraddCmd += args.deployerUser
	_, _, err = conn.runSudoPassword(useraddCmd, args.loginPassword)
	if err != nil {
		return false, fmt.Errorf("error executing useradd: %w", err)
	}

	chpasswdCmd := fmt.Sprintf("echo %s:%s | chpasswd", args.deployerUser, args.deployerPassword)
	_, _, err = conn.runSudoPassword(chpasswdCmd, args.loginPassword)
	if err != nil {
		return false, fmt.Errorf("error setting deployer password: %w", err)
	}

	usermodCmd := fmt.Sprintf("usermod -a -G %s %s", args.deployerUser, args.deployerUser)
	_, _, err = conn.runSudoPassword(usermodCmd, args.loginPassword)
	if err != nil {
		return false, fmt.Errorf("error setting deployer group: %w", err)
	}

	mkdirsshCmd := fmt.Sprintf("mkdir /home/%s/.ssh", args.deployerUser)
	_, _, err = conn.runSudoPassword(mkdirsshCmd, args.loginPassword)
	if err != nil {
		return false, fmt.Errorf("error setting deployer ssh folder: %w", err)
	}

	chownCmd := fmt.Sprintf("chown -R %s:%s /home/%s", args.deployerUser, args.deployerUser, args.deployerUser)
	_, _, err = conn.runSudoPassword(chownCmd, args.loginPassword)
	if err != nil {
		return false, fmt.Errorf("error changing deployer's home dir: %w", err)
	}

	return true, nil
}

func setRootPassword(conn SSHConnection, args Layer1Args) (bool, error) {
	chpasswdCmd := fmt.Sprintf("echo root:%s | chpasswd", args.rootPassword)
	_, _, err := conn.runSudoPassword(chpasswdCmd, args.loginPassword)
	if err != nil {
		return false, fmt.Errorf("error setting root password: %w", err)
	}
	return true, nil
}

func setupsshdConfig(conn SSHConnection, args Layer1Args) (bool, error) {
	config := "/etc/ssh/sshd_config"
	changes := []string{"UsePAM yes", "PermitRootLogin yes", "PasswordAuthentication yes"}

	catCmd := fmt.Sprintf("cat %s", config)
	data, _, err := conn.runSudoPassword(catCmd, args.loginPassword)
	if err != nil {
		return false, fmt.Errorf("error getting current sshd config: %w", err)
	}

	currentConfig := string(data)
	needChanges := false
	for _, change := range changes {
		if strings.Contains(currentConfig, change) {
			needChanges = true
		}
	}
	if !needChanges {
		return false, nil
	}

	backupCmd := fmt.Sprintf("cp %s %s.backup", config, config)
	_, _, err = conn.runSudoPassword(backupCmd, args.loginPassword)
	if err != nil {
		return false, fmt.Errorf("error creating backup of sshd config: %w", err)
	}

	usePamCmd := fmt.Sprintf("sed -i \"s/^#?UsePAM yes/UsePAM no/\" %s", config)
	_, _, err = conn.runSudoPassword(usePamCmd, args.loginPassword)
	if err != nil {
		return false, fmt.Errorf("error resticting sshd PAM use: %w", err)
	}

	permitRootLoginCmd := fmt.Sprintf("sed -i \"s/^#?PermitRootLogin yes/PermitRootLogin no/\" %s", config)
	_, _, err = conn.runSudoPassword(permitRootLoginCmd, args.loginPassword)
	if err != nil {
		return false, fmt.Errorf("error disabling ssh root login: %w", err)
	}

	passwordAuthCmd := fmt.Sprintf("sed -i \"s/^#?PasswordAuthentication yes/PasswordAuthentication no/\" %s", config)
	_, _, err = conn.runSudoPassword(passwordAuthCmd, args.loginPassword)
	if err != nil {
		return false, fmt.Errorf("error disabling ssh password auth: %w", err)
	}

	_, _, err = conn.runSudoPassword("service ssh reload", args.loginPassword)
	if err != nil {
		return false, fmt.Errorf("error reloading ssh service: %w", err)
	}

	return true, nil
}

func setHostname(conn SSHConnection, args Layer1Args) (bool, error) {
	hostnameData, _, err := conn.runSudoPassword("cat /etc/hostname", args.loginPassword)
	if err != nil {
		return false, fmt.Errorf("error getting current hostname: %w", err)
	}

	currentHostname := strings.Trim(string(hostnameData), "\n")
	needChangeHostname := currentHostname != args.hostname

	if needChangeHostname {
		hostnameCmd := fmt.Sprintf("echo \"%s\" > /etc/hostname", args.hostname)
		_, _, err = conn.runSudoPassword(hostnameCmd, args.loginPassword)
		if err != nil {
			return false, fmt.Errorf("error changing hostname: %w", err)
		}
	}

	newHostsLine := fmt.Sprintf("127.0.0.1\t\t%s", args.hostname)

	hostsContentData, _, err := conn.runSudoPassword("cat /etc/hosts", args.loginPassword)
	if err != nil {
		return false, fmt.Errorf("error getting saved hosts: %w", err)
	}

	hostsContent := string(hostsContentData)
	needUpdateHosts := !strings.Contains(hostsContent, newHostsLine)

	if needUpdateHosts {
		hostCmd := fmt.Sprintf("echo \"127.0.0.1\t\t%s\" >> /etc/hosts", args.hostname)
		_, _, err = conn.runSudoPassword(hostCmd, args.loginPassword)
		if err != nil {
			return false, fmt.Errorf("error updating hosts: %w", err)
		}
	}

	return needChangeHostname || needUpdateHosts, nil
}

func disableLoginUser(conn SSHConnection, args Layer1Args) (bool, error) {
	passwdCmd := fmt.Sprintf("passwd -d %s", args.loginUser)
	_, _, err := conn.runSudoPassword(passwdCmd, args.loginPassword)
	if err != nil {
		return false, fmt.Errorf("error removing login user's password: %w", err)
	}

	usermodCmd := fmt.Sprintf("usermod -s /usr/sbin/nologin %s", args.loginUser)
	_, _, err = conn.runSudoPassword(usermodCmd, args.loginPassword)
	if err != nil {
		return false, fmt.Errorf("error removing login user's shell: %w", err)
	}
	return true, nil
}
