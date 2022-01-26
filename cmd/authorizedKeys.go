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
	"github.com/sralloza/rpi-provisioner/ssh"
)

type authorizedKeysArgs struct {
	useSSHKey bool
	user      string
	password  string
	host      string
	port      int
	s3Path    string
	keysPath  string
}

func NewAuthorizedKeysCmd() *cobra.Command {
	args := authorizedKeysArgs{}
	var authorizedKeysCmd = &cobra.Command{
		Use:   "authorized-keys",
		Short: "Update authorized keys",
		Long:  `Download keys from the S3 bucket and update them.`,
		PreRunE: func(cmd *cobra.Command, posArgs []string) error {
			if !args.useSSHKey && len(args.password) == 0 {
				return errors.New("must pass --ssh-key or --password")
			}

			if len(args.keysPath) != 0 && len(args.s3Path) != 0 {
				return errors.New("must pass one of --keys-path or --s3-path")
			}
			if len(args.keysPath) == 0 && len(args.s3Path) == 0 {
				return errors.New("must pass one of --keys-path or --s3-path")
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, posArgs []string) error {
			return updateAuthorizedKeys(args)
		},
	}

	authorizedKeysCmd.Flags().BoolVar(&args.useSSHKey, "ssh-key", false, "Use SSH key")
	authorizedKeysCmd.Flags().StringVar(&args.user, "user", "", "Login user")
	authorizedKeysCmd.Flags().StringVar(&args.password, "password", "", "Login password")
	authorizedKeysCmd.Flags().StringVar(&args.host, "host", "", "Server host")
	authorizedKeysCmd.Flags().IntVar(&args.port, "port", 22, "Server SSH port")
	authorizedKeysCmd.Flags().StringVar(&args.s3Path, "s3-path", "", "Amazon S3 path. Must match the pattern region/bucket/file")
	authorizedKeysCmd.Flags().StringVar(&args.keysPath, "keys-path", "", "Local keys file path. You can select the public key file or a file containing multiple public keys.")

	authorizedKeysCmd.MarkFlagRequired("user")
	authorizedKeysCmd.MarkFlagRequired("host")

	return authorizedKeysCmd
}

func updateAuthorizedKeys(args authorizedKeysArgs) error {
	s3Region, s3Bucket, s3File, err := splitAwsPath(args.s3Path)
	if err != nil {
		return err
	}

	address := fmt.Sprintf("%s:%d", args.host, args.port)

	conn := ssh.SSHConnection{
		Password:  args.password,
		UseSSHKey: args.useSSHKey,
		Debug: DebugFlag,
	}

	err = conn.Connect(args.user, address)
	if err != nil {
		return err
	}
	defer conn.Close()

	fmt.Println("Provisioning SSH keys...")
	if provisioned, err := ssh.UploadsshKeys(conn, ssh.UploadsshKeysArgs{
		User:     args.user,
		Password: args.password,
		Group:    args.user,
		S3Bucket: s3Bucket,
		S3File:   s3File,
		S3Region: s3Region,
		KeysPath: args.keysPath,
	}); err != nil {
		return err
	} else if provisioned {
		fmt.Println("SSH keys provisioned")
	} else {
		fmt.Println("SSH keys already provisioned")
	}

	return nil
}
