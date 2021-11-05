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

type BootArgs struct {
	country     string
	wifiSSID    string
	wifiPass    string
	cmdlineArgs []string
}

func NewBootCmd() *cobra.Command {
	args := BootArgs{}
	var bootCmd = &cobra.Command{
		Use:   "boot [BOOT_PATH]",
		Short: "Setup image before first boot",
		Long:  `Enable ssh, modify cmdline.txt and setup wifi connection`,
		Args: func(cmd *cobra.Command, posArgs []string) error {
			if len(posArgs) != 1 {
				return fmt.Errorf("BOOT_PATH is required")
			}
			bootPath := posArgs[0]
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
		PreRunE: func(cmd *cobra.Command, posArgs []string) error {
			if len(args.wifiPass) == 0 && len(args.wifiSSID) != 0 {
				return fmt.Errorf("you passed --wifi-ssid, you need to specify --wifi-pass")
			}
			if len(args.wifiPass) != 0 && len(args.wifiSSID) == 0 {
				return fmt.Errorf("you passed --wifi-pass, you need to specify --wifi-ssid")
			}
			return nil
		},

		RunE: func(cmd *cobra.Command, posArgs []string) error {
			return setupBoot(args, posArgs[0])
		},
	}

	defaultArgs := []string{"cgroup_enable=cpuset", "cgroup_enable=memory", "cgroup_memory=1"}

	bootCmd.Flags().StringVar(&args.country, "country", "ES", "Country code (2 digits)")
	bootCmd.Flags().StringVar(&args.wifiSSID, "wifi-ssid", "", "WiFi SSID")
	bootCmd.Flags().StringVar(&args.wifiPass, "wifi-pass", "", "WiFi password")
	bootCmd.Flags().StringArrayVar(&args.cmdlineArgs, "cmdline", defaultArgs, "Extra args to append to cmdline.txt")

	return bootCmd
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

func setup_wifi_connection(bootPath string, wifiSSID string, wifiPass string, country string) error {
	fmt.Print("Setting up WiFi connection... ")

	wpaSupplicant := fmt.Sprintf(`country=%s # Your 2-digit country code
ctrl_interface=DIR=/var/run/wpa_supplicant GROUP=netdev
network={
	ssid="%s"
	psk="%s"
	key_mgmt=WPA-PSK
}`, country, wifiSSID, wifiPass)

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

func setupBoot(args BootArgs, bootPath string) error {
	err := enableSSH(bootPath)
	if err != nil {
		return err
	}
	if len(args.wifiSSID) == 0 && len(args.wifiPass) == 0 {
		println("Skipping setting up Wifi connection")
	} else {
		err = setup_wifi_connection(args.country, bootPath, args.wifiSSID, args.wifiPass)
		if err != nil {
			return err
		}
	}
	err = addCmdlineArgs(bootPath, args.cmdlineArgs)
	if err != nil {
		return err
	}

	return nil
}
