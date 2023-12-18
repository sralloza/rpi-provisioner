package layer2

import (
	"fmt"
	"strings"

	"github.com/sralloza/rpi-provisioner/pkg/info"
	"github.com/sralloza/rpi-provisioner/ssh"
)

type Layer2Args struct {
	User string
	Host string
	Port int
}

func NewManager() *layer2Manager {
	return &layer2Manager{}
}

type layer2Manager struct {
	conn ssh.SSHConnection
}

func (m *layer2Manager) Provision(args Layer2Args) (error, error) {
	address := fmt.Sprintf("%s:%d", args.Host, args.Port)

	info.Title("Connecting to server")
	m.conn = ssh.SSHConnection{UseSSHKey: true}
	err := m.conn.Connect(args.User, address)
	if err != nil {
		info.Fail()
		return err, nil
	}
	info.Ok()
	defer m.conn.Close()

	return m.provisionLayer2(args)
}

func (m *layer2Manager) provisionLayer2(args Layer2Args) (error, error) {
	info.Title("Updating and upgrading packages")
	if err := m.installLibraries(); err != nil {
		info.Fail()
		return err, nil
	}
	info.Ok()

	info.Title("Installing zsh")
	if installed, err := m.installZsh(args); err != nil {
		info.Fail()
		return err, nil
	} else if installed {
		info.Ok()
	} else {
		info.Skipped()
	}

	info.Title("Installing oh-my-zsh")
	if installed, err := m.installOhMyZsh(args); err != nil {
		info.Fail()
		return err, nil
	} else if installed {
		info.Ok()
	} else {
		info.Skipped()
	}

	info.Title("Installing powerlevel10k")
	if installed, err := m.installPowerlevel10k(); err != nil {
		info.Fail()
		return err, nil
	} else if installed {
		info.Ok()
	} else {
		info.Skipped()
	}

	info.Title("Installing docker")
	installed, dockerInstallErr, err := m.installDocker(args)
	if err != nil {
		info.Fail()
		return err, dockerInstallErr
	} else if installed {
		info.Ok()
	} else {
		info.Skipped()
	}

	return nil, dockerInstallErr
}

func (m *layer2Manager) installLibraries() error {
	_, _, err := m.conn.RunSudo("apt-get update")
	if err != nil {
		return fmt.Errorf("error updating apt registry: %w", err)
	}

	_, _, err = m.conn.RunSudo("apt-get upgrade -y")
	if err != nil {
		return fmt.Errorf("error upgrading libraries: %w", err)
	}

	libraries := []string{
		"build-essential",
		"bat",
		"cmake",
		"cron",
		"curl",
		"git",
		"libffi-dev",
		"mailutils",
		"nano",
		// "python3-pip",
		// "python3",
		"tcpdump",
		"wget",
	}
	installCmd := fmt.Sprintf("apt-get install %s -y", strings.Join(libraries, " "))
	_, _, err = m.conn.RunSudo(installCmd)
	if err != nil {
		return fmt.Errorf("error installing needed libraries: %w", err)
	}

	return nil
}

func (m *layer2Manager) installZsh(args Layer2Args) (bool, error) {
	_, _, err := m.conn.Run("which zsh")
	if err == nil {
		return false, nil
	}

	_, _, err = m.conn.RunSudo("apt install zsh -y")
	if err != nil {
		return false, fmt.Errorf("error installing zsh: %w", err)
	}

	chshCmd := fmt.Sprintf("chsh -s /usr/bin/zsh %s", args.User)
	_, _, err = m.conn.RunSudo(chshCmd)
	if err != nil {
		return false, fmt.Errorf("error setting deployer's shell to zsh: %w", err)
	}

	return true, nil
}

func (m *layer2Manager) installOhMyZsh(args Layer2Args) (bool, error) {
	_, _, err := m.conn.Run("file ~/.oh-my-zsh -E")
	if err == nil {
		return false, nil
	}

	_, _, err = m.conn.Run("curl -L https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh > /tmp/omz.sh")
	if err != nil {
		return false, fmt.Errorf("error downloading oh-my-zsh installer: %w", err)
	}

	_, _, err = m.conn.Run("sh /tmp/omz.sh")
	if err != nil {
		return false, fmt.Errorf("error running oh-my-zsh installer: %w", err)
	}

	_, _, err = m.conn.Run("rm /tmp/omz.sh")
	if err != nil {
		return false, fmt.Errorf("error removing oh-my-zsh installer: %w", err)
	}
	return true, nil
}

func (m *layer2Manager) installPowerlevel10k() (bool, error) {
	_, _, err := m.conn.Run("file ~/.oh-my-zsh/custom/themes/powerlevel10k -E")
	if err != nil {
		_, _, err := m.conn.Run("git clone --depth=1 https://github.com/romkatv/powerlevel10k.git ${ZSH_CUSTOM:-$HOME/.oh-my-zsh/custom}/themes/powerlevel10k")
		if err != nil {
			return false, fmt.Errorf("error cloning powerlevel10k theme: %w", err)
		}
	}

	_, _, err = m.conn.Run("grep 'ZSH_THEME=\"powerlevel10k/powerlevel10k\"' .zshrc")
	if err == nil {
		return false, nil
	}

	_, _, err = m.conn.Run("sed -i 's/ZSH_THEME=\".*\"/ZSH_THEME=\"powerlevel10k\\/powerlevel10k\"/' ~/.zshrc")
	if err != nil {
		return false, fmt.Errorf("error setting ZSH_THEME: %w", err)
	}

	return true, nil
}

func (m *layer2Manager) installDocker(args Layer2Args) (bool, error, error) {
	_, _, err := m.conn.Run("which docker")
	if err == nil {
		return false, nil, nil
	}

	_, _, err = m.conn.Run("docker compose")
	if err != nil {
		return false, nil, fmt.Errorf("docker is installed but docker compose v2 is not")
	}

	_, _, err = m.conn.Run("curl -fsSL https://get.docker.com -o /tmp/get-docker.sh")
	if err != nil {
		return false, nil, fmt.Errorf("error downloading docker installer: %w", err)
	}

	// Docker instalation may fail, but a reboot should fix it
	// https://stackoverflow.com/questions/59752840/docker-socket-failed-with-result-service-start-limit-hit-after-protecting-doc
	var dockerInstallErr error
	_, _, err = m.conn.Run("sudo sh /tmp/get-docker.sh")
	if err != nil {
		dockerInstallErr = fmt.Errorf("error executing docker installer: %w", err)
	}

	_, _, err = m.conn.Run("rm /tmp/get-docker.sh")
	if err != nil {
		return false, dockerInstallErr, fmt.Errorf("error removing docker installer: %w", err)
	}

	_, _, err = m.conn.Run(fmt.Sprintf("sudo usermod -aG docker %s", args.User))
	if err != nil {
		return false, dockerInstallErr, fmt.Errorf("error adding deployer to docker group: %w", err)
	}

	return true, dockerInstallErr, nil
}
