package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "rpi-provisioner",
	Short: "Setup a new raspberry in minutes",
	Long: `Features:
	- Setup your Raspberry Pi without a screen, keyboard or ethernet connection (WiFi connection required).
	- The router will assign a random IP address to the Raspberry Pi (DHCP protocol). You don't have to use nmap or open your router's admin panel to find the IP address, this command will find it.
	- Improve the security of your Raspberry Pi by creating a new user, disabling the default user and forcing the use of SSH keys.
	- Manage the authorized_keys file in your Raspberry Pi. You can use a local file, a file in S3 or a URL.
	- Setup a static IP address for your Raspberry Pi.
	- Install zsh and oh-my-zsh with some useful plugins.
	- Install tailscale to access your raspberry pi from anywhere.
	- Install docker and docker-compose to facilitate the deployment of your applications.
`,
	SilenceErrors: true,
	SilenceUsage:  true,
	Version: "2.0.0-rc1",
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
