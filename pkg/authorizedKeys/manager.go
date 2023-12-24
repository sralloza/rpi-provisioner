package authorizedkeys

import (
	"fmt"

	"github.com/sralloza/rpi-provisioner/pkg/info"
	"github.com/sralloza/rpi-provisioner/pkg/ssh"
)

type AuthorizedKeysArgs struct {
	UseSSHKey bool
	User      string
	Password  string
	Host      string
	Port      int
	KeysUri   string
}

func NewManager() *authorizedKeysManager {
	return &authorizedKeysManager{}
}

type authorizedKeysManager struct {
	conn ssh.SSHConnection
}

func (m *authorizedKeysManager) Update(args AuthorizedKeysArgs) error {
	address := fmt.Sprintf("%s:%d", args.Host, args.Port)

	info.Title("Connecting to %s", address)
	m.conn = ssh.SSHConnection{
		Password:  args.Password,
		UseSSHKey: args.UseSSHKey,
	}

	err := m.conn.Connect(args.User, address)
	if err != nil {
		info.Fail()
		return err
	}
	defer m.conn.Close()
	info.Ok()

	info.Title("Provisioning SSH authorized keys")
	if provisioned, err := UploadsshKeys(m.conn, UploadsshKeysArgs{
		User:     args.User,
		Password: args.Password,
		Group:    args.User,
		KeysUri:  args.KeysUri,
	}); err != nil {
		info.Fail()
		return err
	} else if provisioned {
		info.Ok()
	} else {
		info.Skipped()
	}

	return nil
}
