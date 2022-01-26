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
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "rpi-provisioner",
	Short: "Set up a new raspberry in minutes",
	Long: `Features:
 - Enable SSH and wifi before first boot
 - Set up SSH keys
 - Update system

After using this script use k3sup to launch the cluster.`,
	SilenceErrors: true,
	SilenceUsage:  true,
	Version: "1.3.0-rc3",
}

var DebugFlag bool

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

	rootCmd.PersistentFlags().BoolVar(&DebugFlag, "debug", false, "Enable debug")
}
