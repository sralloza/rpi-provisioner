package authorizedkeys

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sralloza/rpi-provisioner/ssh"
)

type UploadsshKeysArgs struct {
	User     string
	Password string
	Group    string
	KeysUri  string
}

func UploadsshKeys(conn ssh.SSHConnection, args UploadsshKeysArgs) (bool, error) {
	mkdirCmd := fmt.Sprintf("mkdir -p /home/%s/.ssh", args.User)
	_, _, err := conn.Run(mkdirCmd)
	if err != nil {
		return false, fmt.Errorf("error creating user's ssh directory: %w", err)
	}

	catCmd := fmt.Sprintf("cat /home/%s/.ssh/authorized_keys", args.User)
	fileContent, _, err := conn.Run(catCmd)

	var authorizedKeys []string
	if err != nil {
		authorizedKeys = []string{}
	} else {
		authorizedKeys = strings.Split(strings.Trim(fileContent, "\n"), "\n")
	}

	newKeysInfo, err := Get(args.KeysUri)
	if err != nil {
		return false, fmt.Errorf("error getting authorized keys: %w", err)
	}

	newKeys := []string{}
	for _, key := range newKeysInfo {
		newKeys = append(newKeys, key.String())
	}

	finalKeys := removeDuplicateStr(newKeys)
	sort.Strings(finalKeys)

	newFileContent := strings.Trim(strings.Join(finalKeys, "\n"), "\n")

	if len(authorizedKeys) == len(finalKeys) {
		equal := true
		for i := 0; i < len(authorizedKeys); i++ {
			if authorizedKeys[i] != finalKeys[i] {
				equal = false
				continue
			}
		}
		if equal {
			return false, nil
		}
	}

	updateKeysCmd := fmt.Sprintf("echo \"%s\" > /home/%s/.ssh/authorized_keys", newFileContent, args.User)
	_, _, err = conn.RunSudoPassword(updateKeysCmd, args.Password)
	if err != nil {
		return false, fmt.Errorf("error updating authorized_keys: %w", err)
	}

	sshFolder := fmt.Sprintf("/home/%s/.ssh", args.User)
	authorizedKeysPath := fmt.Sprintf("%s/authorized_keys", sshFolder)

	chmodsshCmd := fmt.Sprintf("chmod 700 %s", sshFolder)
	_, _, err = conn.RunSudoPassword(chmodsshCmd, args.Password)
	if err != nil {
		return false, fmt.Errorf("error setting permissions to ssh folder: %w", err)
	}

	chmodAkpath := fmt.Sprintf("chmod 600 %s", authorizedKeysPath)
	_, _, err = conn.RunSudoPassword(chmodAkpath, args.Password)
	if err != nil {
		return false, fmt.Errorf("error setting permissions to authorized_keys: %w", err)
	}

	ownership := fmt.Sprintf("%s:%s", args.User, args.Group)
	chownsshCmd := fmt.Sprintf("chown %s %s", ownership, sshFolder)
	_, _, err = conn.RunSudoPassword(chownsshCmd, args.Password)
	if err != nil {
		return false, fmt.Errorf("error setting ownership of ssh folder: %w", err)
	}

	chownAkpCmd := fmt.Sprintf("chown %s %s", ownership, authorizedKeysPath)
	_, _, err = conn.RunSudoPassword(chownAkpCmd, args.Password)
	if err != nil {
		return false, fmt.Errorf("error setting ownership of authorized_keys: %w", err)
	}

	return true, nil
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
