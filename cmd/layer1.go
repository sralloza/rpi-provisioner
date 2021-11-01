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

	"golang.org/x/crypto/ssh"

	"github.com/spf13/cobra"
)

type Layer1Settings struct {
	loginUser        string
	loginPassword    string
	deployerGroup    string
	deployerUser     string
	deployerPassword string
	s3Bucket         string
	s3File           string
	s3Region         string
	rootPassword     string
}

// layer1Cmd represents the layer1 command
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
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Provisioning layer 1...")
		return layer1(cmd)
	},
}

func layer1(cmd *cobra.Command) error {
	loginUser, err := cmd.Flags().GetString("login-user")
	if err != nil {
		return err
	}

	loginPassword, err := cmd.Flags().GetString("login-password")
	if err != nil {
		return err
	}

	deployerUser, err := cmd.Flags().GetString("deployer-user")
	if err != nil {
		return err
	}

	deployerPassword, err := cmd.Flags().GetString("deployer-password")
	if err != nil {
		return err
	}

	rootPassword, err := cmd.Flags().GetString("root-password")
	if err != nil {
		return err
	}

	host, err := cmd.Flags().GetString("host")
	if err != nil {
		return err
	}

	s3Path, err := cmd.Flags().GetString("s3-path")
	if err != nil {
		return err
	}
	
	s3Region, s3Bucket, s3File, err := splitAwsPath(s3Path)
	if err != nil {
		return err
	}

	port, err := cmd.Flags().GetInt("port")
	if err != nil {
		return err
	}

	staticIP, _ := cmd.Flags().GetIP("static-ip")

	address := fmt.Sprintf("%s:%d", host, port)

	config := &ssh.ClientConfig{
		User: loginUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(loginPassword),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	conn, err := ssh.Dial("tcp", address, config)
	if err != nil {
		if strings.Index(err.Error(), "no supported methods remain") != -1 {
			println("SSH Connection error, layer 1 should be provisioned")
			return nil
		}
		return err
	}
	defer conn.Close()

	err = setupDeployer(conn, Layer1Settings{
		loginUser:        loginUser,
		loginPassword:    loginPassword,
		deployerGroup:    deployerUser,
		deployerUser:     deployerUser,
		deployerPassword: deployerPassword,
		s3Bucket:         s3Bucket,
		s3File:           s3File,
		s3Region:         s3Region,
		rootPassword:     rootPassword,
	})
	if err != nil {
		return err
	}

	if len(staticIP) != 0 {
		fmt.Printf("Setting up static ip %s\n", staticIP)
		setupNetworking(conn, interfaceArgs{
			ip:       staticIP,
			password: loginPassword,
		})
	}

	return nil
}

func setupDeployer(conn *ssh.Client, settings Layer1Settings) error {
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
	if err := disableLoginUser(conn, settings); err != nil {
		return err
	}
	return nil
}

func sudoStdinLogin(cmd string, settings Layer1Settings) string {
	return basicSudoStdin(cmd, settings.loginPassword)
}

func sudoStdinDeployer(cmd string, settings Layer1Settings) string {
	return basicSudoStdin(cmd, settings.deployerPassword)
}

func createDeployerGroup(conn *ssh.Client, settings Layer1Settings) error {
	command := fmt.Sprintf("grep -q %s /etc/group", settings.deployerGroup)
	_, _, err := runCommand(command, conn)

	if err == nil {
		fmt.Println("Deployer group already exists")
	} else {
		command := sudoStdinLogin(fmt.Sprintf("groupadd %s", settings.deployerGroup), settings)
		stdout, stderr, err := runCommand(command, conn)
		if err != nil {
			return fmt.Errorf("error creating deployer group: %s [%s %s]", err, stdout, stderr)
		}
		fmt.Println("Deployer group created")
	}

	fmt.Println("Checking sudo access")
	_, _, err = runCommand(sudoStdinLogin("whoami", settings), conn)
	if err != nil {
		return nil
	}
	fmt.Println("Updating sudoers file")
	_, _, err = runCommand(sudoStdinLogin("cp /etc/sudoers sudoers", settings), conn)
	if err != nil {
		return err
	}

	initialSudoers, _, err := runCommand(sudoStdinLogin("cat /etc/sudoers", settings), conn)
	if err != nil {
		return err
	}
	initialSudoers = strings.Trim(initialSudoers, "\n\r")

	extraSudoer := fmt.Sprintf("%s ALL=(ALL) NOPASSWD: ALL", settings.deployerGroup)
	if strings.Index(initialSudoers, extraSudoer) != -1 {
		fmt.Println("Sudoer already setup")
		return nil
	}

	newSudoers := fmt.Sprintf("%s\n\n%s\n", initialSudoers, extraSudoer)
	newSudoers = strings.ReplaceAll(newSudoers, "\r\n", "\n")

	// _, _, err = runCommand(sudoStdin+fmt.Sprintf("echo '%s' | %stee /etc/sudoers", newSudoers, sudoStdin), conn)
	_, _, err = runCommand(sudoStdinLogin(fmt.Sprintf("echo \"%s\" > /etc/sudoers", newSudoers), settings), conn)
	if err != nil {
		return err
	}
	// sudoers = sudoers.encode("utf8").replace(b"\r\n", b"\n")

	return nil
}

func createDeployerUser(conn *ssh.Client, settings Layer1Settings) error {
	fmt.Println("Creating deployer user")
	_, _, err := runCommand("id "+settings.deployerUser, conn)
	if err == nil {
		fmt.Println("Deployer user already created")
		return nil
	}

	useraddCmd := fmt.Sprintf("useradd -m -c 'deployer' -s /bin/bash -g '%s' ", settings.deployerGroup)
	useraddCmd += settings.deployerUser
	_, _, err = runCommand(sudoStdinLogin(useraddCmd, settings), conn)
	if err != nil {
		return err
	}

	chpasswdCmd := fmt.Sprintf("echo %s:%s | chpasswd", settings.deployerUser, settings.deployerPassword)
	_, _, err = runCommand(sudoStdinLogin(chpasswdCmd, settings), conn)
	if err != nil {
		return err
	}

	usermodCmd := fmt.Sprintf("usermod -a -G %s %s", settings.deployerGroup, settings.deployerUser)
	_, _, err = runCommand(sudoStdinLogin(usermodCmd, settings), conn)
	if err != nil {
		return err
	}

	mkdirsshCmd := fmt.Sprintf("mkdir /home/%s/.ssh", settings.deployerUser)
	_, _, err = runCommand(sudoStdinLogin(mkdirsshCmd, settings), conn)
	if err != nil {
		return err
	}

	chownCmd := fmt.Sprintf("chown -R %s:%s /home/%s", settings.deployerUser, settings.deployerGroup, settings.deployerUser)
	_, _, err = runCommand(sudoStdinLogin(chownCmd, settings), conn)
	if err != nil {
		return err
	}

	return nil
}

func setRootPassword(conn *ssh.Client, settings Layer1Settings) error {
	fmt.Println("Setting root password")
	chpasswdCmd := fmt.Sprintf("echo root:%s | chpasswd", settings.rootPassword)
	_, _, err := runCommand(sudoStdinLogin(chpasswdCmd, settings), conn)
	if err != nil {
		return err
	}
	return nil
}

func setupsshdConfig(conn *ssh.Client, settings Layer1Settings) error {
	config := "/etc/ssh/sshd_config"

	backupCmd := fmt.Sprintf("cp %s %s.backup", config, config)
	_, _, err := runCommand(sudoStdinLogin(backupCmd, settings), conn)
	if err != nil {
		return err
	}

	usePamCmd := fmt.Sprintf("sed -i \"s/^UsePAM yes/UsePAM no/\" %s", config)
	_, _, err = runCommand(sudoStdinLogin(usePamCmd, settings), conn)
	if err != nil {
		return err
	}

	permitRootLoginCmd := fmt.Sprintf("sed -i \"s/^PermitRootLogin yes/PermitRootLogin no/\" %s", config)
	_, _, err = runCommand(sudoStdinLogin(permitRootLoginCmd, settings), conn)
	if err != nil {
		return err
	}

	passwordAuthCmd := fmt.Sprintf("sed -i \"s/^#PasswordAuthentication yes/PasswordAuthentication no/\" %s", config)
	_, _, err = runCommand(sudoStdinLogin(passwordAuthCmd, settings), conn)
	if err != nil {
		return err
	}

	_, _, err = runCommand(sudoStdinLogin("service ssh reload", settings), conn)
	if err != nil {
		return err
	}

	return nil
}

func disableLoginUser(conn *ssh.Client, settings Layer1Settings) error {
	passwdCmd := fmt.Sprintf("passwd -d %s", settings.loginUser)
	_, _, err := runCommand(sudoStdinLogin(passwdCmd, settings), conn)
	if err != nil {
		return err
	}

	usermodCmd := fmt.Sprintf("usermod -s /usr/sbin/nologin %s", settings.loginUser)
	_, _, err = runCommand(sudoStdinLogin(usermodCmd, settings), conn)
	if err != nil {
		return err
	}
	return nil
}

func init() {
	rootCmd.AddCommand(layer1Cmd)
	layer1Cmd.Flags().String("login-user", "", "Login user")
	layer1Cmd.Flags().String("login-password", "", "Login password")
	layer1Cmd.Flags().String("deployer-user", "", "Deployer user")
	layer1Cmd.Flags().String("deployer-password", "", "Deployer password")
	layer1Cmd.Flags().String("root-password", "", "Root password")
	layer1Cmd.Flags().String("host", "", "Server host")
	layer1Cmd.Flags().Int("port", 22, "Server SSH port")
	layer1Cmd.Flags().String("s3-path", "", "Amazon S3 path. Must match the pattern region/bucket/file")
	layer1Cmd.Flags().IP("static-ip", nil, "Set up the static ip for eth0 and wlan0")

	layer1Cmd.MarkFlagRequired("login-user")
	layer1Cmd.MarkFlagRequired("login-password")
	layer1Cmd.MarkFlagRequired("deployer-user")
	layer1Cmd.MarkFlagRequired("deployer-password")
	layer1Cmd.MarkFlagRequired("host")
	layer1Cmd.MarkFlagRequired("s3-path")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// layer1Cmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// layer1Cmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
