package ssh

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/pkg/sftp"
	"github.com/rs/zerolog"
	"github.com/sralloza/rpi-provisioner/pkg/logging"
	"golang.org/x/crypto/ssh"
)

type SSHConnection struct {
	config    *ssh.ClientConfig
	conn      *ssh.Client
	Password  string
	UseSSHKey bool
	Timeout   int64
	log       *zerolog.Logger
}

func (c *SSHConnection) Connect(user string, address string) error {
	c.log = logging.Get()
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

func (c SSHConnection) RunSudo(cmd string) (string, string, error) {
	return c.Run(c.basicSudoStdin(cmd, ""))
}

func (c SSHConnection) RunSudoPassword(cmd string, password string) (string, string, error) {
	return c.Run(c.basicSudoStdin(cmd, password))
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

func (c SSHConnection) basicSudoStdin(cmd string, password string) string {
	if len(password) == 0 {
		return fmt.Sprintf("sudo bash -c '%s'", cmd)
	}
	return fmt.Sprintf("echo %s | sudo -S bash -c '%s'", password, cmd)
}

func (c SSHConnection) Run(cmd string) (string, string, error) {
	c.log.Debug().Str("cmd", cmd).Msg("Running command via ssh")
	sess, err := c.conn.NewSession()
	if err != nil {
		return "", "", fmt.Errorf("could not stablish ssh session: %w", err)
	}
	defer sess.Close()
	sessStdOut, err := sess.StdoutPipe()
	if err != nil {
		return "", "", fmt.Errorf("could not get stdout pipe: %w", err)
	}
	sessStderr, err := sess.StderrPipe()
	if err != nil {
		return "", "", fmt.Errorf("could not get stderr pipe: %w", err)
	}
	err = sess.Run(cmd)

	bufOut := new(strings.Builder)
	io.Copy(bufOut, sessStdOut)
	bufErr := new(strings.Builder)
	io.Copy(bufErr, sessStderr)

	c.log.Debug().
		Str("cmd", cmd).
		Str("stdout", bufOut.String()).
		Str("stderr", bufErr.String()).
		Err(err).
		Msg("Command executed via ssh")

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
