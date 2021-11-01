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

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

// authorizedKeysCmd represents the authorizedKeys command
var authorizedKeysCmd = &cobra.Command{
	Use:   "authorized-keys",
	Short: "Update authorized keys",
	Long:  `Download keys from the S3 bucket and update them.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return updateAuthorizedKeys(cmd)
	},
}

func updateAuthorizedKeys(cmd *cobra.Command) error {
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
	if len(user) == 0 {
		return errors.New("must specify --user")
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

	address := fmt.Sprintf("%s:%d", host, port)

	var auth []ssh.AuthMethod

	if usesshKey {
		auth = append(auth, publicKey("~/.ssh/id_rsa"))
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

	err = uploadsshKeys(conn, UploadsshKeysArgs{
		user:     user,
		password: password,
		group:    user,
		s3Bucket: s3Bucket,
		s3File:   s3File,
		s3Region: s3Region,
	})
	if err != nil {
		return err
	}
	return nil
}

func init() {
	rootCmd.AddCommand(authorizedKeysCmd)

	authorizedKeysCmd.Flags().Bool("ssh-key", false, "Use ssh key")
	authorizedKeysCmd.Flags().String("user", "", "Login user")
	authorizedKeysCmd.Flags().String("password", "", "Login password")
	authorizedKeysCmd.Flags().String("host", "", "Server host")
	authorizedKeysCmd.Flags().Int("port", 22, "Server SSH port")
	authorizedKeysCmd.Flags().String("s3-path", "", "Amazon S3 path. Must match the pattern region/bucket/file")

	authorizedKeysCmd.MarkFlagRequired("s3-path")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// authorizedKeysCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// authorizedKeysCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
