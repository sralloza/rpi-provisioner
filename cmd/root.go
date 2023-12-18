package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "rpi-provisioner",
	Short: "Setup a new raspberry in minutes",
	Long: `Features:
 - Create default user and enable both ssh and wifi before first boot
 - Setup ssh keys
 - Update system

After using this script use k3sup to launch the cluster.`,
	SilenceErrors: true,
	SilenceUsage:  true,
	Version: "1.4.0",
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	rootCmd.AddCommand(NewLayer1Cmd())
	rootCmd.AddCommand(NewLayer2Cmd())

	rootCmd.AddCommand(NewAuthorizedKeysCmd())
	rootCmd.AddCommand(NewNetworkingCmd())
	rootCmd.AddCommand(NewBootCmd())
	rootCmd.AddCommand(NewFindCommand())
}
