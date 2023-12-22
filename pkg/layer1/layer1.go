package layer1

import (
	"fmt"
	"net"
	"strings"

	"github.com/sralloza/rpi-provisioner/pkg/authorizedkeys"
	"github.com/sralloza/rpi-provisioner/pkg/info"
	"github.com/sralloza/rpi-provisioner/pkg/networking"
	"github.com/sralloza/rpi-provisioner/ssh"
)

type Layer1Args struct {
	LoginUser        string
	LoginPassword    string
	DeployerUser     string
	DeployerPassword string
	RootPassword     string
	Host             string
	Port             int
	KeysUri          string
	PrimaryIP        net.IP
	SecondaryIP      net.IP
}

func NewManager() *layer1Manager {
	return &layer1Manager{}
}

type layer1Manager struct {
	conn ssh.SSHConnection
}

func (m *layer1Manager) Provision(args Layer1Args) (bool, error) {
	address := fmt.Sprintf("%s:%d", args.Host, args.Port)

	m.conn = ssh.SSHConnection{
		Password:  args.LoginPassword,
		UseSSHKey: false,
	}

	info.Title("Connecting to %s", address)
	err := m.conn.Connect(args.LoginUser, address)
	if err != nil {
		if strings.Contains(err.Error(), "no supported methods remain") {
			info.Skipped()
			fmt.Println("SSH Connection error, layer 1 should be provisioned")
			return false, nil
		}
		info.Fail()
		return false, fmt.Errorf("SSH connection error: %w", err)
	}
	info.Ok()
	defer m.conn.Close()

	return m.provisionLayer1(args)
}

func (m *layer1Manager) provisionLayer1(args Layer1Args) (bool, error) {
	info.Title("Creating deployer group")
	if provisioned, err := m.createDeployerGroup(args); err != nil {
		info.Fail()
		return false, err
	} else if provisioned {
		info.Ok()
	} else {
		info.Skipped()
	}

	info.Title("Provisioning deployer sudo access")
	if provisioned, err := m.provisionSudoer(args); err != nil {
		info.Fail()
		return false, err
	} else if provisioned {
		info.Ok()
	} else {
		info.Skipped()
	}

	info.Title("Creating deployer user")
	if provisioned, err := m.createDeployerUser(args); err != nil {
		info.Fail()
		return false, err
	} else if provisioned {
		info.Ok()
	} else {
		info.Skipped()
	}

	if len(args.RootPassword) > 0 {
		info.Title("Provisioning root password")
		if provisioned, err := m.setRootPassword(args); err != nil {
			info.Fail()
			return false, err
		} else if provisioned {
			info.Ok()
		} else {
			info.Skipped()
		}
	}

	info.Title("Provisioning SSH keys")
	if provisioned, err := authorizedkeys.UploadsshKeys(m.conn, authorizedkeys.UploadsshKeysArgs{
		User:     args.DeployerUser,
		Password: args.LoginPassword,
		Group:    args.DeployerUser,
		KeysUri:  args.KeysUri,
	}); err != nil {
		info.Fail()
		return false, err
	} else if provisioned {
		info.Ok()
	} else {
		info.Skipped()
	}

	info.Title("Configuring SSHD")
	if provisioned, err := m.setupsshdConfig(args); err != nil {
		info.Fail()
		return false, err
	} else if provisioned {
		info.Ok()
	} else {
		info.Skipped()
	}

	info.Title("Disabling loginUser login")
	if provisioned, err := m.disableLoginUser(args); err != nil {
		info.Fail()
		return false, err
	} else if provisioned {
		info.Ok()
	} else {
		info.Skipped()
	}

	if len(args.PrimaryIP) > 0 {
		if len(args.SecondaryIP) > 0 {
			info.Title("Provisioning static IPs %s and %s", args.PrimaryIP, args.SecondaryIP)
		} else {
			info.Title("Provisioning static IP %s", args.PrimaryIP)
		}
		if provisioned, err := networking.SetupNetworking(m.conn, args.PrimaryIP, args.SecondaryIP, args.LoginPassword, args.Host); err != nil {
			info.Fail()
			return false, err
		} else if provisioned {
			info.Ok()
		} else {
			info.Skipped()
		}
	}

	return true, nil
}

func (m *layer1Manager) createDeployerGroup(args Layer1Args) (bool, error) {
	grepCmd := fmt.Sprintf("grep -q %s /etc/group", args.DeployerUser)
	_, _, err := m.conn.Run(grepCmd)

	if err == nil {
		return false, nil
	}
	groupaddCmd := fmt.Sprintf("groupadd %s", args.DeployerUser)
	stdout, stderr, err := m.conn.RunSudoPassword(groupaddCmd, args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error creating deployer group: %s [%s %s]", err, stdout, stderr)
	}
	return true, nil
}

func (m *layer1Manager) provisionSudoer(args Layer1Args) (bool, error) {
	initialSudoers, _, err := m.conn.RunSudoPassword("cat /etc/sudoers", args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error getting sudoers info: %w", err)
	}
	initialSudoers = strings.Trim(initialSudoers, "\n\r")

	extraSudoer := fmt.Sprintf("%s ALL=(ALL) NOPASSWD: ALL", args.DeployerUser)
	if strings.Contains(initialSudoers, extraSudoer) {
		return false, nil
	}
	_, _, err = m.conn.RunSudoPassword("cp /etc/sudoers /etc/sudoers.bkp", args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error creating sudoers backup: %w", err)
	}

	sudoersCmd := fmt.Sprintf("echo \"\n%s\n\" >> /etc/sudoers", extraSudoer)
	_, _, err = m.conn.RunSudoPassword(sudoersCmd, args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error updating sudoers: %w", err)
	}
	return true, nil
}

func (m *layer1Manager) createDeployerUser(args Layer1Args) (bool, error) {
	_, _, err := m.conn.Run("id " + args.DeployerUser)
	if err == nil {
		return false, nil
	}

	useraddCmd := fmt.Sprintf("useradd -m -c 'deployer' -s /bin/bash -g '%s' ", args.DeployerUser)
	useraddCmd += args.DeployerUser
	_, _, err = m.conn.RunSudoPassword(useraddCmd, args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error executing useradd: %w", err)
	}

	chpasswdCmd := fmt.Sprintf("echo %s:%s | chpasswd", args.DeployerUser, args.DeployerPassword)
	_, _, err = m.conn.RunSudoPassword(chpasswdCmd, args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error setting deployer password: %w", err)
	}

	usermodCmd := fmt.Sprintf("usermod -a -G %s %s", args.DeployerUser, args.DeployerUser)
	_, _, err = m.conn.RunSudoPassword(usermodCmd, args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error setting deployer group: %w", err)
	}

	mkdirsshCmd := fmt.Sprintf("mkdir /home/%s/.ssh", args.DeployerUser)
	_, _, err = m.conn.RunSudoPassword(mkdirsshCmd, args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error setting deployer ssh folder: %w", err)
	}

	chownCmd := fmt.Sprintf("chown -R %s:%s /home/%s", args.DeployerUser, args.DeployerUser, args.DeployerUser)
	_, _, err = m.conn.RunSudoPassword(chownCmd, args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error changing deployer's home dir: %w", err)
	}

	return true, nil
}

func (m *layer1Manager) setRootPassword(args Layer1Args) (bool, error) {
	chpasswdCmd := fmt.Sprintf("echo root:%s | chpasswd", args.RootPassword)
	_, _, err := m.conn.RunSudoPassword(chpasswdCmd, args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error setting root password: %w", err)
	}
	return true, nil
}

func (m *layer1Manager) setupsshdConfig(args Layer1Args) (bool, error) {
	config := "/etc/ssh/sshd_config"
	changes := []string{"UsePAM yes", "PermitRootLogin yes", "PasswordAuthentication yes"}

	catCmd := fmt.Sprintf("cat %s", config)
	data, _, err := m.conn.RunSudoPassword(catCmd, args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error getting current sshd config: %w", err)
	}

	currentConfig := string(data)
	needChanges := false
	for _, change := range changes {
		if strings.Contains(currentConfig, change) {
			needChanges = true
		}
	}
	if !needChanges {
		return false, nil
	}

	backupCmd := fmt.Sprintf("cp %s %s.backup", config, config)
	_, _, err = m.conn.RunSudoPassword(backupCmd, args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error creating backup of sshd config: %w", err)
	}

	usePamCmd := fmt.Sprintf("sed -i \"s/^#*UsePAM yes/UsePAM no/\" %s", config)
	_, _, err = m.conn.RunSudoPassword(usePamCmd, args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error resticting sshd PAM use: %w", err)
	}

	permitRootLoginCmd := fmt.Sprintf("sed -i \"s/^#*PermitRootLogin yes/PermitRootLogin no/\" %s", config)
	_, _, err = m.conn.RunSudoPassword(permitRootLoginCmd, args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error disabling ssh root login: %w", err)
	}

	passwordAuthCmd := fmt.Sprintf("sed -i \"s/^#*PasswordAuthentication yes/PasswordAuthentication no/\" %s", config)
	_, _, err = m.conn.RunSudoPassword(passwordAuthCmd, args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error disabling ssh password auth: %w", err)
	}

	_, _, err = m.conn.RunSudoPassword("service ssh reload", args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error reloading ssh service: %w", err)
	}

	return true, nil
}

func (m *layer1Manager) disableLoginUser(args Layer1Args) (bool, error) {
	passwdCmd := fmt.Sprintf("passwd -d %s", args.LoginUser)
	_, _, err := m.conn.RunSudoPassword(passwdCmd, args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error removing login user's password: %w", err)
	}

	usermodCmd := fmt.Sprintf("usermod -s /usr/sbin/nologin %s", args.LoginUser)
	_, _, err = m.conn.RunSudoPassword(usermodCmd, args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error removing login user's shell: %w", err)
	}
	return true, nil
}
