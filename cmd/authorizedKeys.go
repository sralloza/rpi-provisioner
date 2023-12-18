package cmd

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/sralloza/rpi-provisioner/pkg/authorizedkeys"
)

func NewAuthorizedKeysCmd() *cobra.Command {
	args := authorizedkeys.AuthorizedKeysArgs{}
	var authorizedKeysCmd = &cobra.Command{
		Use:   "authorized-keys",
		Short: "Update authorized keys",
		Long:  `Download keys from the S3 bucket and update them.`,
		PreRunE: func(cmd *cobra.Command, posArgs []string) error {
			if !args.UseSSHKey && len(args.Password) == 0 {
				return errors.New("must pass --ssh-key or --password")
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, posArgs []string) error {
			return authorizedkeys.NewManager().Update(args)
		},
	}

	authorizedKeysCmd.Flags().BoolVar(&args.UseSSHKey, "ssh-key", false, "Use ssh key")
	authorizedKeysCmd.Flags().StringVar(&args.User, "user", "", "Login user")
	authorizedKeysCmd.Flags().StringVar(&args.Password, "password", "", "Login password")
	authorizedKeysCmd.Flags().StringVar(&args.Host, "host", "", "Server host")
	authorizedKeysCmd.Flags().IntVar(&args.Port, "port", 22, "Server SSH port")
	authorizedKeysCmd.Flags().StringVar(&args.KeysUri, "keys-uri", "", "Local keys file path. You can select the public key file or a file containing multiple public keys.")

	authorizedKeysCmd.MarkFlagRequired("user")
	authorizedKeysCmd.MarkFlagRequired("host")
	authorizedKeysCmd.MarkFlagRequired("keys-uri")

	return authorizedKeysCmd
}
