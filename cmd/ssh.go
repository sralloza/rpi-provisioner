package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"golang.org/x/crypto/ssh"
)

func basicSudoStdin(cmd string, password string) string {
	return fmt.Sprintf("echo %s | sudo -S bash -c '%s'", password, cmd)
}

func runCommand(cmd string, conn *ssh.Client) (string, string, error) {
	debug, _ := rootCmd.Flags().GetBool("debug")
	sess, err := conn.NewSession()
	if err != nil {
		panic(err)
	}
	defer sess.Close()
	sessStdOut, err := sess.StdoutPipe()
	if err != nil {
		panic(err)
	}
	sessStderr, err := sess.StderrPipe()
	if err != nil {
		panic(err)
	}
	err = sess.Run(cmd)

	bufOut := new(strings.Builder)
	io.Copy(bufOut, sessStdOut)
	bufErr := new(strings.Builder)
	io.Copy(bufErr, sessStderr)

	if debug {
		fmt.Printf("ssh: %#v -> [%#v | %#v | %v]\n\n", cmd, bufOut.String(), bufErr.String(), err)
	}

	return bufOut.String(), bufErr.String(), err
}

type UploadsshKeysArgs struct {
	user     string
	password string
	group    string
	s3Bucket string
	s3File   string
	s3Region string
}

func dynamicSudo(cmd string, password string) string {
	if len(password) != 0 {
		return basicSudoStdin(cmd, password)
	}
	return fmt.Sprintf("sudo bash -c '%s'", cmd)
}

func uploadsshKeys(conn *ssh.Client, args UploadsshKeysArgs) error {
	fmt.Println("Updating SSH keys")

	catCmd := fmt.Sprintf("cat /home/%s/.ssh/authorized_keys", args.user)
	fileContent, _, err := runCommand(catCmd, conn)
	var authorizedKeys []string
	if err != nil {
		authorizedKeys = strings.Split(strings.Trim(fileContent, "\n"), "\n")
	} else {
		authorizedKeys = []string{}
	}

	newKeys, err := getSavedKeys(args.s3Bucket, args.s3File, args.s3Region)
	if err != nil {
		return err
	}
	finalKeys := append(authorizedKeys, newKeys...)
	finalKeys = removeDuplicateStr(finalKeys)

	newFileContent := strings.Trim(strings.Join(finalKeys, "\n"), "\n")
	updateKeysCmd := fmt.Sprintf("echo \"%s\" > /home/%s/.ssh/authorized_keys", newFileContent, args.user)
	_, _, err = runCommand(dynamicSudo(updateKeysCmd, args.password), conn)
	if err != nil {
		return err
	}

	sshFolder := fmt.Sprintf("/home/%s/.ssh", args.user)
	authorizedKeysPath := fmt.Sprintf("%s/authorized_keys", sshFolder)

	fmt.Println("Fixing permissions of user's .ssh files")
	chmodsshCmd := fmt.Sprintf("chmod 700 %s", sshFolder)
	_, _, err = runCommand(dynamicSudo(chmodsshCmd, args.password), conn)
	if err != nil {
		return err
	}

	chmodAkpath := fmt.Sprintf("chmod 600 %s", authorizedKeysPath)
	_, _, err = runCommand(dynamicSudo(chmodAkpath, args.password), conn)
	if err != nil {
		return err
	}

	ownership := fmt.Sprintf("%s:%s", args.user, args.group)
	chownsshCmd := fmt.Sprintf("chown %s %s", ownership, sshFolder)
	_, _, err = runCommand(dynamicSudo(chownsshCmd, args.password), conn)
	if err != nil {
		return err
	}

	chownAkpCmd := fmt.Sprintf("chown %s %s", ownership, authorizedKeysPath)
	_, _, err = runCommand(dynamicSudo(chownAkpCmd, args.password), conn)
	if err != nil {
		return err
	}

	return nil
}

func getSavedKeys(bucket string, item string, region string) ([]string, error) {
	file, err := os.Create("tmpfile")
	if err != nil {
		return []string{}, err
	}

	defer file.Close()

	sess, _ := session.NewSession(&aws.Config{Region: aws.String(region)})

	downloader := s3manager.NewDownloader(sess)
	numBytes, err := downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(item),
		})
	if err != nil {
		return []string{}, err
	}

	fmt.Println("Downloaded", item, numBytes, "bytes")

	data, err := ioutil.ReadFile("tmpfile")
	if err != nil {
		return []string{}, err
	}
	file.Close()

	err = os.Remove("tmpfile")
	if err != nil {
		return []string{}, err
	}

	var result map[string]string
	json.Unmarshal(data, &result)

	var publicKeys []string
	for _, p := range result {
		publicKeys = append(publicKeys, p)
	}

	return publicKeys, nil
}

func publicKey(path string) ssh.AuthMethod {
	key, err := ioutil.ReadFile(expandPath(path))
	if err != nil {
		panic(err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		panic(err)
	}
	return ssh.PublicKeys(signer)
}
