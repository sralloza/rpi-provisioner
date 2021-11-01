package cmd

import (
	"fmt"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
)

func splitAwsPath(awsPath string) (string, string, string, error) {
	chunks := strings.Split(awsPath, "/")
	if len(chunks) != 3 {
		AwsErrorMsg := "awsPath must match pattern region/bucket/file (%#v)"
		return "", "", "", fmt.Errorf(AwsErrorMsg, awsPath)
	}
	return chunks[0], chunks[1], chunks[2], nil
}

func expandPath(path string) string {
	res, _ := homedir.Expand(path)
	return res
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
