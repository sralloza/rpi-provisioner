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
	keysUri   string
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

			return nil
		},
		RunE: func(cmd *cobra.Command, posArgs []string) error {
			return updateAuthorizedKeys(args)
		},
	}

	authorizedKeysCmd.Flags().BoolVar(&args.useSSHKey, "ssh-key", false, "Use ssh key")
	authorizedKeysCmd.Flags().StringVar(&args.user, "user", "", "Login user")
	authorizedKeysCmd.Flags().StringVar(&args.password, "password", "", "Login password")
	authorizedKeysCmd.Flags().StringVar(&args.host, "host", "", "Server host")
	authorizedKeysCmd.Flags().IntVar(&args.port, "port", 22, "Server SSH port")
	authorizedKeysCmd.Flags().StringVar(&args.keysUri, "keys-uri", "", "Local keys file path. You can select the public key file or a file containing multiple public keys.")

	authorizedKeysCmd.MarkFlagRequired("user")
	authorizedKeysCmd.MarkFlagRequired("host")
	authorizedKeysCmd.MarkFlagRequired("keys-uri")

	return authorizedKeysCmd
}

func updateAuthorizedKeys(args authorizedKeysArgs) error {
	conn := ssh.SSHConnection{
		Password:  args.password,
		UseSSHKey: args.useSSHKey,
	}

	err := conn.Connect(args.user, fmt.Sprintf("%s:%d", args.host, args.port))
	if err != nil {
		return err
	}
	defer conn.Close()

	fmt.Println("Provisioning SSH keys...")
	if provisioned, err := ssh.UploadsshKeys(conn, ssh.UploadsshKeysArgs{
		User:     args.user,
		Password: args.password,
		Group:    args.user,
		KeysUri:  args.keysUri,
	}); err != nil {
		return err
	} else if provisioned {
		fmt.Println("SSH keys provisioned")
	} else {
		fmt.Println("SSH keys already provisioned")
	}

	return nil
}
