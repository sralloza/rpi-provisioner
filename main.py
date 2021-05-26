from fabric import Connection
from pathlib import Path
from config import settings

env = settings


def main():
    con = Connection("rfenix")
    start_provision(con)


def start_provision(con: Connection):
    """
    Start server provisioning
    """
    # Create a new directory for a new remote server
    ssh_keys_name = env.ssh_keys_dir / "fenix_prod_key"
    if not ssh_keys_name.is_file():
        con.local("ssh-keygen -t rsa -b 2048 -f {0} -N \"\"".format(ssh_keys_name))

    # Prevent root SSHing into the remote server
    con.run('sed "/etc/ssh/sshd_config" "^UsePAM yes" "UsePAM no"')
    con.run('sed "/etc/ssh/sshd_config" "^PermitRootLogin yes" "PermitRootLogin no"')
    con.run(
        'sed "/etc/ssh/sshd_config" "^#PasswordAuthentication yes" "PasswordAuthentication no"'
    )

    install_ansible_dependencies(con)
    create_deployer_group(con)
    create_deployer_user(con)
    upload_keys(con)
    set_selinux_permissive(con)
    con.run("service sshd reload")
    upgrade_server(con)


def create_deployer_group(con: Connection):
    """
    Create a user group for all project developers
    """
    con.run("groupadd {}".format(env.user_group))
    con.run("mv /etc/sudoers /etc/sudoers-backup")
    con.run(
        '(cat /etc/sudoers-backup; echo "%'
        + env.user_group
        + ' ALL=(ALL) ALL") > /etc/sudoers'
    )
    con.run("chmod 440 /etc/sudoers")


def create_deployer_user(con: Connection):
    """
    Create a user for the user group
    """
    con.run(
        'adduser -c "{}" -m -g {} {}'.format(
            env.full_name_user, env.user_group, env.user_name
        )
    )
    con.run("passwd {}".format(env.user_name))
    con.run("usermod -a -G {} {}".format(env.user_group, env.user_name))
    con.run("mkdir /home/{}/.ssh".format(env.user_name))
    con.run("chown -R {} /home/{}/.ssh".format(env.user_name, env.user_name))
    con.run("chgrp -R {} /home/{}/.ssh".format(env.user_group, env.user_name))


def upload_keys(con: Connection):
    """
    Upload the SSH public/private keys to the remote server via scp
    """
    scp_command = "scp {} {}/authorized_keys {}@{}:~/.ssh".format(
        env.ssh_keys_name + ".pub", env.ssh_keys_dir, env.user_name, env.host_string
    )
    con.local(scp_command)


def install_ansible_dependencies(con: Connection):
    """
    Install the python-dnf module so that Ansible
    can communicate with Fedora's Package Manager
    """
    # TODO: fix distro
    con.run("dnf install -y python-dnf")


def set_selinux_permissive(con: Connection):
    """
    Set SELinux to Permissive/Disabled Mode
    """
    # for permissive
    con.run("sudo setenforce 0")


def upgrade_server(con: Connection):
    """
    Upgrade the server as a root user
    """
    # TODO: fix distro
    con.run("dnf upgrade -y")
    # optional command (necessary for Fedora 25)
    con.run("dnf install -y python")
    con.run("reboot")


if __name__ == "__main__":
    main()
