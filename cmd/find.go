package cmd

import (
	"github.com/spf13/cobra"
	"github.com/sralloza/rpi-provisioner/pkg/find"
)

func NewFindCommand() *cobra.Command {
	args := find.Args{}
	var findCmd = &cobra.Command{
		Use:   "find",
		Short: "Find your raspberry pi in your local network",
		Long:  `Find your raspberry pi in your local network using SSH.`,
		RunE: func(cmd *cobra.Command, posArgs []string) error {
			if err := find.FindHost(args); err != nil {
				return err
			}
			return nil
		},
	}
	findCmd.Flags().StringVar(&args.Subnet, "subnet", "", "Subnet to find the raspberry")
	findCmd.Flags().StringVar(&args.User, "user", "pi", "User to login via ssh")
	findCmd.Flags().StringVar(&args.Password, "password", "raspberry", "Password to login via ssh")
	findCmd.Flags().BoolVar(&args.UseSSHKey, "ssh-key", false, "Use SSH key to login instead of password")
	findCmd.Flags().IntVar(&args.Port, "port", 22, "Port to connect via ssh")
	return findCmd
}
