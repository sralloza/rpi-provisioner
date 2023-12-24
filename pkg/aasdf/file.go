package authorizedkeys

import (
	"fmt"
	"os"
)

func getAuthorizedKeysFromFile(keysUri string) ([]PublicSSHKey, error) {
	data, err := os.ReadFile(keysUri)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}

	return getKeysFromJson(data)
}
