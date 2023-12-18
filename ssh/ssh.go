package ssh

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/pkg/sftp"
	"github.com/sralloza/rpi-provisioner/pkg/logging"
	"golang.org/x/crypto/ssh"
)

type SSHConnection struct {
	config    *ssh.ClientConfig
	conn      *ssh.Client
	Password  string
	UseSSHKey bool
	Timeout   int64
}

func (c *SSHConnection) Connect(user string, address string) error {
	var auth []ssh.AuthMethod

	if c.UseSSHKey {
		auth = append(auth, publicKey("~/.ssh/id_rsa"))
	} else {
		auth = append(auth, ssh.Password(c.Password))
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

func (c SSHConnection) Run(cmd string) (string, string, error) {
	return runCommand(cmd, c.conn)
}

func (c SSHConnection) RunSudo(cmd string) (string, string, error) {
	return c.Run(basicSudoStdin(cmd, ""))
}

func (c SSHConnection) RunSudoPassword(cmd string, password string) (string, string, error) {
	return c.Run(basicSudoStdin(cmd, password))
}

func (c SSHConnection) Close() {
	c.conn.Close()
}

func (c SSHConnection) WriteToFile(dstPath string, content []byte) error {
	// open an SFTP session over an existing ssh connection.
	sftp, err := sftp.NewClient(c.conn)
	if err != nil {
		return fmt.Errorf("could not stablish sftp connection: %w", err)
	}
	defer sftp.Close()

	// Create the destination file
	dstFile, err := sftp.Create(dstPath)
	if err != nil {
		return fmt.Errorf("could not create remote file in sftp connection: %w", err)
	}
	defer dstFile.Close()

	buffer := strings.NewReader(string(content))

	// write to file
	if _, err := dstFile.ReadFrom(buffer); err != nil {
		return fmt.Errorf("could not write to remote file in sftp connection: %w", err)
	}
	return nil
}

func basicSudoStdin(cmd string, password string) string {
	return fmt.Sprintf("echo %s | sudo -S bash -c '%s'", password, cmd)
}

func runCommand(cmd string, conn *ssh.Client) (string, string, error) {
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

	logger := logging.Get()
	logger.Debug().
		Str("cmd", cmd).
		Str("stdout", bufOut.String()).
		Str("stderr", bufErr.String()).
		Msg("Running command via ssh")

	return bufOut.String(), bufErr.String(), err
}

type UploadsshKeysArgs struct {
	User     string
	Password string
	Group    string
	KeysUri  string
}

func expandPath(path string) string {
	res, _ := homedir.Expand(path)
	return res
}

func publicKey(path string) ssh.AuthMethod {
	key, err := os.ReadFile(expandPath(path))
	if err != nil {
		panic(err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		panic(err)
	}
	return ssh.PublicKeys(signer)
}
