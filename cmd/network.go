package cmd

import (
	"github.com/spf13/cobra"
	"github.com/sralloza/rpi-provisioner/pkg/networking"
)

func NewNetworkingCmd() *cobra.Command {
	args := networking.NetworkingArgs{}

	var networkingCmd = &cobra.Command{
		Use:   "network",
		Short: "Provision networking",
		Long:  `Set up static ip for eth0 and wlan0.`,
		RunE: func(cmd *cobra.Command, posArgs []string) error {
			networkManager := networking.NewNetworkingManager()
			if err := networkManager.Setup(args); err != nil {
				return err
			}

			return nil
		},
	}

	networkingCmd.Flags().BoolVar(&args.UseSSHKey, "ssh-key", false, "Use ssh key")
	networkingCmd.Flags().StringVar(&args.User, "user", "", "Login user")
	networkingCmd.Flags().StringVar(&args.Password, "password", "", "Login password")
	networkingCmd.Flags().StringVar(&args.Host, "host", "", "Server host")
	networkingCmd.Flags().IntVar(&args.Port, "port", 22, "Server SSH port")
	networkingCmd.Flags().IPVar(&args.PrimaryIP, "primary-ip", nil, "Static IP of the primary interface (eth0 if it's connected, wlan0 otherwise)")
	networkingCmd.Flags().IPVar(&args.SecondaryIP, "secondary-ip", nil, "Static IP of the secondary interface (wlan0)")

	networkingCmd.MarkFlagRequired("user")
	networkingCmd.MarkFlagRequired("host")
	networkingCmd.MarkFlagRequired("primary-ip")
	return networkingCmd
}
