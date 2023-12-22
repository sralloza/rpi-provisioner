package boot

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/sralloza/rpi-provisioner/pkg/info"
)

type BootArgs struct {
	Country  string
	WifiSSID string
	WifiPass string
}

type BootManager struct {
}

func NewBootManager() *BootManager {
	return &BootManager{}
}

func (b BootManager) Setup(bootPath, hostname, wifiCountry, wifiSSID, wifiPassword string) error {
	err := b.enableSSH(bootPath)
	if err != nil {
		return err
	}

	err = b.firstRunScript(bootPath, hostname, wifiSSID, wifiPassword, wifiCountry)
	if err != nil {
		return err
	}

	err = b.updateCmdArgs(bootPath)
	if err != nil {
		return err
	}

	return nil
}

func (b BootManager) enableSSH(bootPath string) error {
	info.Title("Enabling ssh")
	emptyFile, err := os.Create(filepath.Join(bootPath, "ssh"))
	if err != nil {
		info.Fail()
		return err
	}
	emptyFile.Close()
	info.Ok()
	return nil
}

type firstRunScriptData struct {
	Hostname    string
	WifiSSID    string
	WifiPass    string
	WifiCountry string
}

func (b BootManager) firstRunScript(bootPath, hostname, wifiSSID, wifiPass, country string) error {
	info.Title("Setting up first run script")

	tmplFile := "pkg/boot/firstrun.tmpl"
	tmpl, err := template.New(filepath.Base(tmplFile)).ParseFiles(tmplFile)
	if err != nil {
		info.Fail()
		return fmt.Errorf("error loading template: %w", err)
	}

	var buffer bytes.Buffer
	err = tmpl.Execute(&buffer, firstRunScriptData{
		Hostname:    hostname,
		WifiSSID:    wifiSSID,
		WifiPass:    wifiPass,
		WifiCountry: country,
	})
	if err != nil {
		info.Fail()
		return fmt.Errorf("error parsing template: %w", err)
	}

	err = os.WriteFile(filepath.Join(bootPath, "firstrun.sh"), buffer.Bytes(), 0)
	if err != nil {
		info.Fail()
		return fmt.Errorf("error writing first run script: %w", err)
	}

	info.Ok()
	return nil
}

func (b BootManager) updateCmdArgs(bootPath string) error {
	info.Title("Enabling firstrun script")

	cmdLinePath := filepath.Join(bootPath, "cmdline.txt")
	content, err := os.ReadFile(cmdLinePath)
	if err != nil {
		info.Fail()
		return fmt.Errorf("error reading cmdline.txt: %w", err)
	}

	cmdArgs := strings.Split(strings.Trim(string(content), "\n"), " ")
	cmdArgs = append(cmdArgs,
		"systemd.run=/boot/firstrun.sh",
		"systemd.run_success_action=reboot",
		"systemd.unit=kernel-command-line.target")

	cmdArgs = removeDuplicates(cmdArgs)
	sort.StringSlice(cmdArgs).Sort()

	newContent := strings.Join(cmdArgs, " ") + "\n"
	err = os.WriteFile(cmdLinePath, []byte(newContent), 0)
	if err != nil {
		info.Fail()
		return fmt.Errorf("error writing cmdline.txt: %w", err)
	}

	info.Ok()
	return nil
}

func removeDuplicates[T comparable](slice []T) []T {
	seen := make(map[T]bool)
	result := []T{}

	for _, val := range slice {
		if _, ok := seen[val]; !ok {
			seen[val] = true
			result = append(result, val)
		}
	}
	return result
}
