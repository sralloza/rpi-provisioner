package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"golang.org/x/crypto/ssh"
)

type SSHConnection struct {
	config    *ssh.ClientConfig
	conn      *ssh.Client
	password  string
	useSSHKey bool
}

func (c *SSHConnection) Connect(user string, address string) error {
	var auth []ssh.AuthMethod

	if c.useSSHKey {
		auth = append(auth, publicKey("~/.ssh/id_rsa"))
	} else {
		auth = append(auth, ssh.Password(c.password))
	}

	c.config = &ssh.ClientConfig{
		User:            user,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	conn, err := ssh.Dial("tcp", address, c.config)
	if err != nil {
		return fmt.Errorf("could not stablish ssh connection: %w", err)
	}
	c.conn = conn
	return nil
}

func (c SSHConnection) run(cmd string) (string, string, error) {
	return runCommand(cmd, c.conn)
}

func (c SSHConnection) runSudo(cmd string) (string, string, error) {
	return c.run(basicSudoStdin(cmd, ""))
}

func (c SSHConnection) runSudoPassword(cmd string, password string) (string, string, error) {
	return c.run(basicSudoStdin(cmd, password))
}

func (c SSHConnection) close() {
	c.conn.Close()
}

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
	keysPath string
}

func uploadsshKeys(conn SSHConnection, args UploadsshKeysArgs) (bool, error) {
	mkdirCmd := fmt.Sprintf("mkdir -p /home/%s/.ssh", args.user)
	_, _, err := conn.run(mkdirCmd)
	if err != nil {
		return false, fmt.Errorf("error creating user's ssh directory: %w", err)
	}

	catCmd := fmt.Sprintf("cat /home/%s/.ssh/authorized_keys", args.user)
	fileContent, _, err := conn.run(catCmd)

	var authorizedKeys []string
	if err != nil {
		authorizedKeys = []string{}
	} else {
		authorizedKeys = strings.Split(strings.Trim(fileContent, "\n"), "\n")
	}

	var newKeys []string
	if len(args.keysPath) != 0 {
		newKeys, err = getKeysFromFile(args.keysPath)
	} else {
		newKeys, err = getAWSSavedKeys(args.s3Bucket, args.s3File, args.s3Region)
	}
	if err != nil {
		return false, fmt.Errorf("error generating SSH auth: %w", err)
	}

	sort.Strings(authorizedKeys)

	finalKeys := append(authorizedKeys, newKeys...)
	finalKeys = removeDuplicateStr(finalKeys)
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

	updateKeysCmd := fmt.Sprintf("echo \"%s\" > /home/%s/.ssh/authorized_keys", newFileContent, args.user)
	_, _, err = conn.runSudoPassword(updateKeysCmd, args.password)
	if err != nil {
		return false, fmt.Errorf("error updating authorized_keys: %w", err)
	}

	sshFolder := fmt.Sprintf("/home/%s/.ssh", args.user)
	authorizedKeysPath := fmt.Sprintf("%s/authorized_keys", sshFolder)

	chmodsshCmd := fmt.Sprintf("chmod 700 %s", sshFolder)
	_, _, err = conn.runSudoPassword(chmodsshCmd, args.password)
	if err != nil {
		return false, fmt.Errorf("error setting permissions to ssh folder: %w", err)
	}

	chmodAkpath := fmt.Sprintf("chmod 600 %s", authorizedKeysPath)
	_, _, err = conn.runSudoPassword(chmodAkpath, args.password)
	if err != nil {
		return false, fmt.Errorf("error setting permissions to authorized_keys: %w", err)
	}

	ownership := fmt.Sprintf("%s:%s", args.user, args.group)
	chownsshCmd := fmt.Sprintf("chown %s %s", ownership, sshFolder)
	_, _, err = conn.runSudoPassword(chownsshCmd, args.password)
	if err != nil {
		return false, fmt.Errorf("error setting ownership of ssh folder: %w", err)
	}

	chownAkpCmd := fmt.Sprintf("chown %s %s", ownership, authorizedKeysPath)
	_, _, err = conn.runSudoPassword(chownAkpCmd, args.password)
	if err != nil {
		return false, fmt.Errorf("error setting ownership of authorized_keys: %w", err)
	}

	return true, nil
}

func getKeysFromFile(filepath string) ([]string, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return []string{}, err
	}

	keys := strings.Split(strings.Trim(string(data), "\n"), "\n")
	return keys, nil
}

func getAWSSavedKeys(bucket string, item string, region string) ([]string, error) {
	file, err := os.Create("tmpfile")
	if err != nil {
		return []string{}, err
	}

	defer file.Close()

	sess, _ := session.NewSession(&aws.Config{Region: aws.String(region)})

	downloader := s3manager.NewDownloader(sess)
	_, err = downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(item),
		})
	if err != nil {
		return []string{}, err
	}

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
