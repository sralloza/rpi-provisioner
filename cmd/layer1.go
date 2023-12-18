package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/sralloza/rpi-provisioner/pkg/layer1"
)

func NewLayer1Cmd() *cobra.Command {
	args := layer1.Layer1Args{}
	var layer1Cmd = &cobra.Command{
		Use:   "layer1",
		Short: "Provision layer 1",
		Long: `Layer 1 uses the default user and bash shell. It will perform the following tasks:
 - Create deployer user
 - Setup ssh config and keys
 - Disable pi login
 - [optional] static ip configuration
 `,
		RunE: func(cmd *cobra.Command, posArgs []string) error {
			provisioned, err := layer1.ProvisionLayer1(args)
			if err != nil {
				return err
			}

			fmt.Println("\nLayer 1 provisioned successfully")
			if provisioned {
				fmt.Println(
					"\nNote: you must restart the server to apply the hostname change " +
						"and suppress the security risk warning")
				fmt.Printf("  ssh %s@%s sudo reboot\n", args.DeployerUser, args.Host)
			}

			fmt.Println("\nContinue with layer 2 or SSH into server:")
			fmt.Printf("  ssh %s@%s\n", args.DeployerUser, args.Host)
			return nil
		},
	}

	layer1Cmd.Flags().StringVar(&args.LoginUser, "login-user", "pi", "Login user")
	layer1Cmd.Flags().StringVar(&args.LoginPassword, "login-password", "raspberry", "Login password")
	layer1Cmd.Flags().StringVar(&args.DeployerPassword, "deployer-user", "", "Deployer user")
	layer1Cmd.Flags().StringVar(&args.DeployerUser, "deployer-password", "", "Deployer password")
	layer1Cmd.Flags().StringVar(&args.RootPassword, "root-password", "", "Root password")
	layer1Cmd.Flags().StringVar(&args.Host, "host", "", "Server host")
	layer1Cmd.Flags().IntVar(&args.Port, "port", 22, "Server SSH port")
	layer1Cmd.Flags().StringVar(&args.KeysUri, "keys-uri", "", "Keys uri. Can be a AWS S3 URI, HTTP(S) or a file path.")
	layer1Cmd.Flags().IPVar(&args.StaticIP, "static-ip", nil, "Set up the static ip for eth0 and wlan0")

	layer1Cmd.MarkFlagRequired("deployer-user")
	layer1Cmd.MarkFlagRequired("deployer-password")
	layer1Cmd.MarkFlagRequired("host")
	layer1Cmd.MarkFlagRequired("keysUri")
	return layer1Cmd
}
