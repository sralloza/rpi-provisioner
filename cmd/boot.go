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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// bootCmd represents the boot command
var bootCmd = &cobra.Command{
	Use:   "boot [BOOT_PATH]",
	Short: "Setup image before first boot",
	Long:  `Enable ssh, modify cmdline.txt and setup wifi connection`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("BOOT_PATH is required")
		}
		bootPath := args[0]
		if !isDirectory(bootPath) {
			return fmt.Errorf("'%s' is not a directory", bootPath)
		}

		cmdLinePath := filepath.Join(bootPath, "cmdline.txt")
		_, err := os.Stat(cmdLinePath)
		if err != nil {
			return fmt.Errorf("cmdline.txt ('%s') does not exist", cmdLinePath)
		}
		return nil
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		wifi_ssid, _ := cmd.Flags().GetString("wifi-ssid")
		wifi_pass, _ := cmd.Flags().GetString("wifi-pass")

		if len(wifi_pass) == 0 && len(wifi_ssid) != 0 {
			return fmt.Errorf("You passed --wifi-ssid, you need to specify --wifi-pass")
		}
		if len(wifi_pass) != 0 && len(wifi_ssid) == 0 {
			return fmt.Errorf("You passed --wifi-pass, you need to specify --wifi-ssid")
		}
		return nil
	},

	Run: func(cmd *cobra.Command, args []string) {
		run(cmd, args[0])
	},
}

func isDirectory(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}

	return fileInfo.IsDir()
}

func enableSSH(bootPath string) error {
	fmt.Print("Enabling ssh... ")
	emptyFile, err := os.Create(filepath.Join(bootPath, "ssh"))
	if err != nil {
		fmt.Println("FAIL")
		return err
	}
	emptyFile.Close()
	fmt.Println("OK")
	return nil
}

func setup_wifi_connection(bootPath string, wifiSSID string, wifiPass string) error {
	fmt.Print("Setting up WiFi connection... ")

	wpaSupplicant := fmt.Sprintf(`country=ES # Your 2-digit country code
ctrl_interface=DIR=/var/run/wpa_supplicant GROUP=netdev
network={
	ssid="%s"
	psk="%s"
	key_mgmt=WPA-PSK
}`, wifiSSID, wifiPass)

	emptyFile, err := os.Create(filepath.Join(bootPath, "wpa_supplicant.conf"))
	if err != nil {
		fmt.Println("FAIL")
		return err
	}
	emptyFile.WriteString(wpaSupplicant)
	emptyFile.Close()
	fmt.Println("OK")
	return nil
}

func addCmdlineArgs(bootPath string, args []string) error {
	fmt.Print("Adding args to cmdline... ")
	cmdLinePath := filepath.Join(bootPath, "cmdline.txt")
	content, err := ioutil.ReadFile(cmdLinePath)

	if err != nil {
		fmt.Println("FAILED [read]")
		return err
	}

	parsedContent := strings.Replace(string(content), "\n", "", -1)
	parsedContent = strings.Replace(parsedContent, "\r", "", -1)

	currentArgs := strings.Split(parsedContent, " ")
	newArgs := removeDuplicateStr(append(currentArgs, args...))
	newContent := []byte(strings.Join(newArgs, " "))

	if err := ioutil.WriteFile(cmdLinePath, newContent, 0644); err != nil {
		fmt.Print("FAILED [write]")
		return err
	}

	fmt.Print("OK")
	return nil

}

func removeDuplicateStr(strSlice []string) []string {
	allKeys := make(map[string]bool)
	list := []string{}
	for _, item := range strSlice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

func run(cmd *cobra.Command, bootPath string) {
	wifiSSID, _ := cmd.Flags().GetString("wifi-ssid")
	wifiPass, _ := cmd.Flags().GetString("wifi-pass")
	cmdlineArgs, _ := cmd.Flags().GetStringArray("cmdline")

	enableSSH(bootPath)
	if len(wifiSSID) == 0 && len(wifiPass) == 0 {
		println("Skipping setting up Wifi connection")
	} else {
		setup_wifi_connection(bootPath, wifiSSID, wifiPass)
	}
	addCmdlineArgs(bootPath, cmdlineArgs)
}

func init() {
	rootCmd.AddCommand(bootCmd)
	defaultArgs := []string{"cgroup_enable=cpuset", "cgroup_enable=memory", "cgroup_memory=1"}

	bootCmd.Flags().String("wifi-ssid", "", "WiFi SSID")
	bootCmd.Flags().String("wifi-pass", "", "WiFi password")
	bootCmd.Flags().StringArray("cmdline", defaultArgs, "Extra args to append to cmdline.txt")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// bootCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// bootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
