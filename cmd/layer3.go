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
	"fmt"

	"github.com/spf13/cobra"
)

// layer3Cmd represents the layer3 command
var layer3Cmd = &cobra.Command{
	Use:   "layer3",
	Short: "Provision layer 3",
	Long: `Layer 3 uses the deployer user and the fish shell. It consists of:
 - Install k3s?
`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("layer3 called")
	},
}

func init() {
	rootCmd.AddCommand(layer3Cmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// layer3Cmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// layer3Cmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
