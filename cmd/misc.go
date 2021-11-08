package cmd

import (
	"fmt"
	"strings"
)

func splitAwsPath(awsPath string) (string, string, string, error) {
	if len(awsPath) == 0 {
		return "", "", "", nil
	}
	chunks := strings.Split(awsPath, "/")
	if len(chunks) != 3 {
		AwsErrorMsg := "awsPath must match pattern region/bucket/file (%#v)"
		return "", "", "", fmt.Errorf(AwsErrorMsg, awsPath)
	}
	return chunks[0], chunks[1], chunks[2], nil
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
