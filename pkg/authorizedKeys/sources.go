package authorizedkeys

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sralloza/rpi-provisioner/pkg/logging"
)

type PublicSSHKey struct {
	Alias string `json:"alias"`
	Type  string `json:"type"`
	Key   string `json:"key"`
}

func (p PublicSSHKey) String() string {
	return fmt.Sprintf("%s %s %s", p.Type, p.Key, p.Alias)
}

func Get(keysUri string) ([]PublicSSHKey, error) {
	if strings.HasPrefix(keysUri, "s3://") {
		return getAuthorizedKeysFromS3(keysUri)
	}

	if strings.HasPrefix(keysUri, "http://") || strings.HasPrefix(keysUri, "https://") {
		return getAuthorizedKeysFromHTTP(keysUri)
	}

	return getAuthorizedKeysFromFile(keysUri)
}

func getKeysFromJson(fileContent []byte) ([]PublicSSHKey, error) {
	var result []PublicSSHKey
	if err := json.Unmarshal(fileContent, &result); err != nil {
		log := logging.Get()
		log.Error().Err(err).Str("data", string(fileContent)).Msg("error decoding keys json")
		return nil, fmt.Errorf("error decoding keys json: %w", err)
	}

	return result, nil
}
