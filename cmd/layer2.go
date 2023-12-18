package cmd

import (
	"fmt"

	"github.com/sralloza/rpi-provisioner/pkg/layer2"

	"github.com/spf13/cobra"
)

func NewLayer2Cmd() *cobra.Command {
	args := layer2.Layer2Args{}
	var layer2Cmd = &cobra.Command{
		Use:   "layer2",
		Short: "Provision layer 2",
		Long: `Layer 2 uses the deployer user and bash. It will perform the following tasks:
- Update and upgrade packages
- Install some useful libraries
- Install zsh
- Install oh-my-zsh
- Install docker
`,
		RunE: func(cmd *cobra.Command, posArgs []string) error {
			err, dockerInstallErr := layer2.NewManager().Provision(args)
			if err != nil {
				return err
			}
			if dockerInstallErr != nil {
				fmt.Printf("\nDocker instalation failed, will probably be fixed with a reboot\n"+
					"  Consider rebooting the server and then execute the layer2 command again\n"+
					"    ssh %s@%s sudo reboot\n", args.User, args.Host)
			}
			fmt.Println("Layer 2 provisioned")
			return nil
		},
	}

	layer2Cmd.Flags().StringVar(&args.User, "user", "", "Login user")
	layer2Cmd.Flags().StringVar(&args.Host, "host", "", "Server host")
	layer2Cmd.Flags().IntVar(&args.Port, "port", 22, "Server SSH port")

	layer2Cmd.MarkFlagRequired("user")
	layer2Cmd.MarkFlagRequired("host")

	return layer2Cmd
}
