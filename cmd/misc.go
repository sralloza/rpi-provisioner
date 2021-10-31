package cmd

import homedir "github.com/mitchellh/go-homedir"

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
