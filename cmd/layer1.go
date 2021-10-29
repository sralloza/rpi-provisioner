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
	"io"
	"io/ioutil"
	"strings"

	homedir "github.com/mitchellh/go-homedir"

	"golang.org/x/crypto/ssh"

	"github.com/spf13/cobra"
)

type Settings struct {
	loginUser        string
	loginPassword    string
	deployerGroup    string
	deployerUser     string
	deployerPassword string
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
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("layer1 called")
		return layer1(cmd)
	},
}

func layer1(cmd *cobra.Command) error {
	loginUser, err := cmd.Flags().GetString("login-user")
	if err != nil {
		return err
	}
	if len(loginUser) == 0 {
		return errors.New("must pass --login-user")
	}

	loginPassword, err := cmd.Flags().GetString("login-password")
	if err != nil {
		return err
	}
	if len(loginPassword) == 0 {
		return errors.New("must pass --login-password")
	}

	deployerUser, err := cmd.Flags().GetString("deployer-user")
	if err != nil {
		return err
	}
	if len(deployerUser) == 0 {
		return errors.New("must pass --deployer-user")
	}

	deployerPassword, err := cmd.Flags().GetString("deployer-password")
	if err != nil {
		return err
	}
	if len(deployerPassword) == 0 {
		return errors.New("must pass --deployer-password")
	}

	host, err := cmd.Flags().GetString("host")
	if err != nil {
		return err
	}
	if len(host) == 0 {
		return errors.New("must pass --host")
	}

	port, err := cmd.Flags().GetInt("port")
	if err != nil {
		return err
	}

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
		return err
	}
	defer conn.Close()

	err = setupDeployer(conn, Settings{
		loginUser:        loginUser,
		loginPassword:    loginPassword,
		deployerGroup:    deployerUser,
		deployerUser:     deployerUser,
		deployerPassword: deployerPassword,
	})
	if err != nil {
		return err
	}
	return nil
}

func setupDeployer(conn *ssh.Client, settings Settings) error {
	if err := createDeployerGroup(conn, settings); err != nil {
		return err
	}
	return nil
}

func sudoStdin(cmd string, settings Settings) string {
	return fmt.Sprintf("echo %s | sudo -S bash -c '%s'", settings.loginPassword, cmd)
}

func createDeployerGroup(conn *ssh.Client, settings Settings) error {
	command := fmt.Sprintf("grep -q %s /etc/group", settings.deployerGroup)
	_, _, err := runCommand(command, conn)

	if err == nil {
		fmt.Println("Deployer group already exists")
	} else {
		command := sudoStdin(fmt.Sprintf("groupadd %s", settings.deployerGroup), settings)
		stdout, stderr, err := runCommand(command, conn)
		if err != nil {
			return fmt.Errorf("error creating deployer group: %s [%s %s]", err, stdout, stderr)
		}
		fmt.Println("Deployer group created")
	}

	fmt.Println("Checking sudo access")
	_, _, err = runCommand(sudoStdin("whoami", settings), conn)
	if err != nil {
		return nil
	}
	fmt.Println("Updating sudoers file")
	_, _, err = runCommand(sudoStdin("cp /etc/sudoers sudoers", settings), conn)
	if err != nil {
		return err
	}

	initialSudoers, _, err := runCommand(sudoStdin("cat /etc/sudoers", settings), conn)
	if err != nil {
		return err
	}
	initialSudoers = strings.Trim(initialSudoers, "\n\r")

	newSudoers := initialSudoers + "\n\n" + settings.deployerGroup + " ALL=(ALL) NOPASSWD: ALL\n"
	newSudoers = strings.ReplaceAll(newSudoers, "\r\n", "\n")

	// _, _, err = runCommand(sudoStdin+fmt.Sprintf("echo '%s' | %stee /etc/sudoers", newSudoers, sudoStdin), conn)
	_, _, err = runCommand(sudoStdin(fmt.Sprintf("echo \"%s\" > /etc/sudoers", newSudoers), settings), conn)
	if err != nil {
		return err
	}
	// sudoers = sudoers.encode("utf8").replace(b"\r\n", b"\n")

	return nil
}

func runCommand(cmd string, conn *ssh.Client) (string, string, error) {
	debug, _ := rootCmd.Flags().GetBool("debug")
	sess, err := conn.NewSession()
	if err != nil {
		panic(err)
	}
	defer sess.Close()
	sessStdOut, err := sess.StdoutPipe()
	if err != nil {
		panic(err)
	}
	sessStderr, err := sess.StderrPipe()
	if err != nil {
		panic(err)
	}
	err = sess.Run(cmd)

	bufOut := new(strings.Builder)
	io.Copy(bufOut, sessStdOut)
	bufErr := new(strings.Builder)
	io.Copy(bufErr, sessStderr)

	if debug {
		fmt.Printf("ssh: %#v -> [%#v | %#v | %v]\n", cmd, bufOut.String(), bufErr.String(), err)
	}

	return bufOut.String(), bufErr.String(), err
}

func expandPath(path string) string {
	res, _ := homedir.Expand(path)
	return res
}

func publicKey(path string) ssh.AuthMethod {
	key, err := ioutil.ReadFile(expandPath(path))
	if err != nil {
		panic(err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		panic(err)
	}
	return ssh.PublicKeys(signer)
}

func init() {
	rootCmd.AddCommand(layer1Cmd)
	layer1Cmd.Flags().String("login-user", "", "Login user")
	layer1Cmd.Flags().String("login-password", "", "Login password")
	layer1Cmd.Flags().String("deployer-user", "", "Deployer user")
	layer1Cmd.Flags().String("deployer-password", "", "Deployer password")
	layer1Cmd.Flags().String("host", "", "Server host")
	layer1Cmd.Flags().Int("port", 22, "Server SSH port")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// layer1Cmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// layer1Cmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
