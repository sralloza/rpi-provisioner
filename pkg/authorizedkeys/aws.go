package authorizedkeys

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func getAuthorizedKeysFromS3(keysUri string) ([]PublicSSHKey, error) {
	keysUri = strings.TrimPrefix(keysUri, "s3://")
	s3Region, s3Bucket, s3File, err := splitAwsPath(keysUri)
	if err != nil {
		return nil, fmt.Errorf("invalid s3 uri: %w", err)
	}

	sess, _ := session.NewSession(&aws.Config{Region: aws.String(s3Region)})

	buffer := &aws.WriteAtBuffer{}
	downloader := s3manager.NewDownloader(sess)
	_, err = downloader.Download(buffer,
		&s3.GetObjectInput{
			Bucket: aws.String(s3Bucket),
			Key:    aws.String(s3File),
		})
	if err != nil {
		return nil, fmt.Errorf("error downloading file from s3: %w", err)
	}

	return getKeysFromJson(buffer.Bytes())
}

func splitAwsPath(awsPath string) (string, string, string, error) {
	if len(awsPath) == 0 {
		return "", "", "", nil
	}
	chunks := strings.Split(awsPath, "/")
	if len(chunks) != 3 {
		AwsErrorMsg := "awsPath must match pattern s3://<region>/<bucket>/<file> (%#v)"
		return "", "", "", fmt.Errorf(AwsErrorMsg, awsPath)
	}
	return chunks[0], chunks[1], chunks[2], nil
}
