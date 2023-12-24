package authorizedkeys

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/carlmjohnson/requests"
)

func getAuthorizedKeysFromHTTP(keysUri string) ([]PublicSSHKey, error) {
	var err error
	if strings.HasPrefix(keysUri, "https://drive.google.com/file/d") {
		keysUri, err = patchGoogleDriveURL(keysUri)
		if err != nil {
			return nil, err
		}
	}

	var body bytes.Buffer
	err = requests.URL(keysUri).
		ToBytesBuffer(&body).
		Fetch(context.Background())

	if err != nil {
		return nil, fmt.Errorf("error fetching authorized keys from %s: %w", keysUri, err)
	}
	return getKeysFromJson(body.Bytes())
}

func patchGoogleDriveURL(input string) (string, error) {
	r := regexp.MustCompile(`https://drive.google.com/file/d/([^/]+)`)
	matches := r.FindStringSubmatch(input)
	if len(matches) == 0 {
		return "", fmt.Errorf("error parsing google drive url: %s", input)
	}
	return fmt.Sprintf("https://drive.google.com/uc?export=download&id=%s", matches[1]), nil
}
