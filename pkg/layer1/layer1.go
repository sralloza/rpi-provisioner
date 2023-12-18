package layer1

import (
	"fmt"
	"net"
	"strings"

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
	StaticIP         net.IP
}

func ProvisionLayer1(args Layer1Args) (bool, error) {
	address := fmt.Sprintf("%s:%d", args.Host, args.Port)

	conn := ssh.SSHConnection{
		Password:  args.LoginPassword,
		UseSSHKey: false,
	}

	info.Title("Connecting to %s", address)
	err := conn.Connect(args.LoginUser, address)
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
	defer conn.Close()

	info.Title("Creating deployer group")
	if provisioned, err := createDeployerGroup(conn, args); err != nil {
		info.Fail()
		return false, err
	} else if provisioned {
		info.Ok()
	} else {
		info.Skipped()
	}

	info.Title("Provisioning deployer sudo access")
	if provisioned, err := provisionSudoer(conn, args); err != nil {
		info.Fail()
		return false, err
	} else if provisioned {
		info.Ok()
	} else {
		info.Skipped()
	}

	info.Title("Creating deployer user")
	if provisioned, err := createDeployerUser(conn, args); err != nil {
		info.Fail()
		return false, err
	} else if provisioned {
		info.Ok()
	} else {
		info.Skipped()
	}

	if len(args.RootPassword) > 0 {
		info.Title("Provisioning root password")
		if provisioned, err := setRootPassword(conn, args); err != nil {
			info.Fail()
			return false, err
		} else if provisioned {
			info.Ok()
		} else {
			info.Skipped()
		}
	}

	info.Title("Provisioning SSH keys")
	if provisioned, err := ssh.UploadsshKeys(conn, ssh.UploadsshKeysArgs{
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
	if provisioned, err := setupsshdConfig(conn, args); err != nil {
		info.Fail()
		return false, err
	} else if provisioned {
		info.Ok()
	} else {
		info.Skipped()
	}

	info.Title("Disabling loginUser login")
	if provisioned, err := disableLoginUser(conn, args); err != nil {
		info.Fail()
		return false, err
	} else if provisioned {
		info.Ok()
	} else {
		info.Skipped()
	}

	if len(args.StaticIP) != 0 {
		info.Title("Provisioning static ip %s", args.StaticIP)
		if provisioned, err := networking.SetupNetworking(conn, args.StaticIP, args.LoginPassword); err != nil {
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

func createDeployerGroup(conn ssh.SSHConnection, args Layer1Args) (bool, error) {
	grepCmd := fmt.Sprintf("grep -q %s /etc/group", args.DeployerUser)
	_, _, err := conn.Run(grepCmd)

	if err == nil {
		return false, nil
	}
	groupaddCmd := fmt.Sprintf("groupadd %s", args.DeployerUser)
	stdout, stderr, err := conn.RunSudoPassword(groupaddCmd, args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error creating deployer group: %s [%s %s]", err, stdout, stderr)
	}
	return true, nil
}

func provisionSudoer(conn ssh.SSHConnection, args Layer1Args) (bool, error) {
	initialSudoers, _, err := conn.RunSudoPassword("cat /etc/sudoers", args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error getting sudoers info: %w", err)
	}
	initialSudoers = strings.Trim(initialSudoers, "\n\r")

	extraSudoer := fmt.Sprintf("%s ALL=(ALL) NOPASSWD: ALL", args.DeployerUser)
	if strings.Contains(initialSudoers, extraSudoer) {
		return false, nil
	}
	_, _, err = conn.RunSudoPassword("cp /etc/sudoers /etc/sudoers.bkp", args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error creating sudoers backup: %w", err)
	}

	sudoersCmd := fmt.Sprintf("echo \"\n%s\n\" >> /etc/sudoers", extraSudoer)
	_, _, err = conn.RunSudoPassword(sudoersCmd, args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error updating sudoers: %w", err)
	}
	return true, nil
}

func createDeployerUser(conn ssh.SSHConnection, args Layer1Args) (bool, error) {
	_, _, err := conn.Run("id " + args.DeployerUser)
	if err == nil {
		return false, nil
	}

	useraddCmd := fmt.Sprintf("useradd -m -c 'deployer' -s /bin/bash -g '%s' ", args.DeployerUser)
	useraddCmd += args.DeployerUser
	_, _, err = conn.RunSudoPassword(useraddCmd, args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error executing useradd: %w", err)
	}

	chpasswdCmd := fmt.Sprintf("echo %s:%s | chpasswd", args.DeployerUser, args.DeployerPassword)
	_, _, err = conn.RunSudoPassword(chpasswdCmd, args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error setting deployer password: %w", err)
	}

	usermodCmd := fmt.Sprintf("usermod -a -G %s %s", args.DeployerUser, args.DeployerUser)
	_, _, err = conn.RunSudoPassword(usermodCmd, args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error setting deployer group: %w", err)
	}

	mkdirsshCmd := fmt.Sprintf("mkdir /home/%s/.ssh", args.DeployerUser)
	_, _, err = conn.RunSudoPassword(mkdirsshCmd, args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error setting deployer ssh folder: %w", err)
	}

	chownCmd := fmt.Sprintf("chown -R %s:%s /home/%s", args.DeployerUser, args.DeployerUser, args.DeployerUser)
	_, _, err = conn.RunSudoPassword(chownCmd, args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error changing deployer's home dir: %w", err)
	}

	return true, nil
}

func setRootPassword(conn ssh.SSHConnection, args Layer1Args) (bool, error) {
	chpasswdCmd := fmt.Sprintf("echo root:%s | chpasswd", args.RootPassword)
	_, _, err := conn.RunSudoPassword(chpasswdCmd, args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error setting root password: %w", err)
	}
	return true, nil
}

func setupsshdConfig(conn ssh.SSHConnection, args Layer1Args) (bool, error) {
	config := "/etc/ssh/sshd_config"
	changes := []string{"UsePAM yes", "PermitRootLogin yes", "PasswordAuthentication yes"}

	catCmd := fmt.Sprintf("cat %s", config)
	data, _, err := conn.RunSudoPassword(catCmd, args.LoginPassword)
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
	_, _, err = conn.RunSudoPassword(backupCmd, args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error creating backup of sshd config: %w", err)
	}

	usePamCmd := fmt.Sprintf("sed -i \"s/^#*UsePAM yes/UsePAM no/\" %s", config)
	_, _, err = conn.RunSudoPassword(usePamCmd, args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error resticting sshd PAM use: %w", err)
	}

	permitRootLoginCmd := fmt.Sprintf("sed -i \"s/^#*PermitRootLogin yes/PermitRootLogin no/\" %s", config)
	_, _, err = conn.RunSudoPassword(permitRootLoginCmd, args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error disabling ssh root login: %w", err)
	}

	passwordAuthCmd := fmt.Sprintf("sed -i \"s/^#*PasswordAuthentication yes/PasswordAuthentication no/\" %s", config)
	_, _, err = conn.RunSudoPassword(passwordAuthCmd, args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error disabling ssh password auth: %w", err)
	}

	_, _, err = conn.RunSudoPassword("service ssh reload", args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error reloading ssh service: %w", err)
	}

	return true, nil
}

func disableLoginUser(conn ssh.SSHConnection, args Layer1Args) (bool, error) {
	passwdCmd := fmt.Sprintf("passwd -d %s", args.LoginUser)
	_, _, err := conn.RunSudoPassword(passwdCmd, args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error removing login user's password: %w", err)
	}

	usermodCmd := fmt.Sprintf("usermod -s /usr/sbin/nologin %s", args.LoginUser)
	_, _, err = conn.RunSudoPassword(usermodCmd, args.LoginPassword)
	if err != nil {
		return false, fmt.Errorf("error removing login user's shell: %w", err)
	}
	return true, nil
}
