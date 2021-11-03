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

type authorizedKeysArgs struct {
	useSSHKey bool
	user      string
	password  string
	host      string
	port      int
	s3Path    string
}

func NewAuthorizedKeysCmd() *cobra.Command {
	args := authorizedKeysArgs{}
	var authorizedKeysCmd = &cobra.Command{
		Use:   "authorized-keys",
		Short: "Update authorized keys",
		Long:  `Download keys from the S3 bucket and update them.`,
		RunE: func(cmd *cobra.Command, rawArgs []string) error {
			return updateAuthorizedKeys(args)
		},
	}

	authorizedKeysCmd.Flags().BoolVar(&args.useSSHKey, "ssh-key", false, "Use ssh key")
	authorizedKeysCmd.Flags().StringVar(&args.user, "user", "", "Login user")
	authorizedKeysCmd.Flags().StringVar(&args.password, "password", "", "Login password")
	authorizedKeysCmd.Flags().StringVar(&args.host, "host", "", "Server host")
	authorizedKeysCmd.Flags().IntVar(&args.port, "port", 22, "Server SSH port")
	authorizedKeysCmd.Flags().StringVar(&args.s3Path, "s3-path", "", "Amazon S3 path. Must match the pattern region/bucket/file")

	authorizedKeysCmd.MarkFlagRequired("s3-path")

	return authorizedKeysCmd
}

func updateAuthorizedKeys(args authorizedKeysArgs) error {
	if !args.useSSHKey && len(args.password) == 0 {
		return errors.New("must pass --ssh-key or --password")
	}

	s3Region, s3Bucket, s3File, err := splitAwsPath(args.s3Path)
	if err != nil {
		return err
	}

	address := fmt.Sprintf("%s:%d", args.host, args.port)

	var auth []ssh.AuthMethod

	if args.useSSHKey {
		auth = append(auth, publicKey("~/.ssh/id_rsa"))
	} else {
		auth = append(auth, ssh.Password(args.password))
	}
	config := &ssh.ClientConfig{
		User:            args.user,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	conn, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return err
	}
	defer conn.Close()

	err = uploadsshKeys(conn, UploadsshKeysArgs{
		user:     args.user,
		password: args.password,
		group:    args.user,
		s3Bucket: s3Bucket,
		s3File:   s3File,
		s3Region: s3Region,
	})
	if err != nil {
		return err
	}
	return nil
}
