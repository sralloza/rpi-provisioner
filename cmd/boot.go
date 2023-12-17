package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/sralloza/rpi-provisioner/pkg/boot"
)

type BootArgs struct {
	hostname string
	wifiCountry  string
	wifiSSID string
	wifiPass string
}

func NewBootCmd() *cobra.Command {
	args := BootArgs{}
	var bootCmd = &cobra.Command{
		Use:   "boot [BOOT_PATH]",
		Short: "Setup image before first boot",
		Long:  `Enable ssh, setup wifi connection and create default user (pi) the firstrun.sh script`,
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
			bm := boot.NewBootManager()
			return bm.Setup(posArgs[0], args.hostname, args.wifiCountry, args.wifiSSID, args.wifiPass)
		},
	}

	bootCmd.Flags().StringVar(&args.hostname, "hostname", "", "Hostname")
	bootCmd.Flags().StringVar(&args.wifiCountry, "wifi-country", "ES", "WiFi country code (2 digits)")
	bootCmd.Flags().StringVar(&args.wifiSSID, "wifi-ssid", "", "WiFi SSID")
	bootCmd.Flags().StringVar(&args.wifiPass, "wifi-pass", "", "WiFi password")

	bootCmd.MarkFlagRequired("hostname")

	return bootCmd
}

func isDirectory(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}

	return fileInfo.IsDir()
}
