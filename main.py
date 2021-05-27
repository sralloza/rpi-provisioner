import os
from pathlib import Path
from time import sleep

import click
from fabric import Config, Connection
from invoke.exceptions import UnexpectedExit
from paramiko.ssh_exception import BadAuthenticationType
from passlib.context import CryptContext

from config import settings

env = settings
config1 = Config(
    overrides={
        "sudo": {"password": settings.initial_login_password},
        # "run": {"hide": True},
    }
)
config2 = Config(
    overrides={
        "sudo": {"password": settings.initial_login_password},
        "run": {"shell": "/usr/bin/fish"},
    }
)


def main():
    con1 = Connection(
        host=env.host,
        user=env.initial_login_user,
        connect_kwargs={"password": env.initial_login_password},
        config=config1,
    )
    con2 = Connection(
        host=env.host,
        user=env.deployer_user,
        config=config1,
    )
    con3 = Connection(
        host=env.host,
        user=env.deployer_user,
        config=config2,
    )

    try:
        with con1 as con:
            con.sudo("whoami")
            setup_deployer(con)
    except BadAuthenticationType as exc:
        if exc.allowed_types != ["publickey"]:
            raise
        info("First login failed, deployer should be already created")

    sleep(1)
    with con2 as con:
        con.sudo("whoami")
        setup_server(con)

    with con3 as con:
        con.sudo("whoami")
        deploy_services(con)


def info(msg):
    click.secho(msg, fg="bright_green")


def setup_deployer(con: Connection):
    create_deployer_group(con)
    create_deployer_user(con)

    ensure_local_keys(con)
    update_keys(con)
    setup_sshd_config(con)


def setup_server(con: Connection):
    install_libraries(con)

    info("Done")


def create_deployer_group(con: Connection):
    info("Creating deployer group")
    if con.run(f"grep -q {env.deployer_group} /etc/group", warn=True).ok:
        info("Deployer group already exists")
    else:
        con.sudo(f"groupadd {env.deployer_group}")

    current_sudoers = con.sudo("cat /etc/sudoers").stdout.strip()
    con.sudo("cp /etc/sudoers /etc/sudoers.backup")

    info("Updating sudoers file")

    # TODO: only allow to run sudo tee without password
    sudoers = current_sudoers + f"\n\n%{env.deployer_group} ALL=(ALL) NOPASSWD: ALL\n"

    Path("tmp").write_text(sudoers, "utf8")
    con.put("tmp", "/tmp/sudoers")
    con.sudo("chown root:root /tmp/sudoers")
    con.sudo("chmod 440 /tmp/sudoers")
    con.sudo("sed 's/^M$//' -i /tmp/sudoers")
    con.sudo(f"mv /tmp/sudoers /etc/sudoers")
    Path("tmp").unlink()

    # Check that sudo is not broken due to sudoers file
    con.sudo("whoami")


def create_deployer_user(con: Connection):
    info("Creating deployer user")
    if con.run(f"id {env.deployer_user}", warn=True).ok:
        return info("Deployer user already exists")

    password = CryptContext(schemes=["sha256_crypt"]).hash(env.deployer_password)
    info(password)

    con.sudo(
        f"useradd -m -c '{env.full_name_user}' -s /bin/bash "
        f"-g {env.deployer_group} -p '{password}' {env.deployer_user}"
    )
    con.sudo("usermod -a -G {} {}".format(env.deployer_group, env.deployer_user))
    con.sudo("mkdir /home/{}/.ssh".format(env.deployer_user))
    con.sudo("chown -R {} /home/{}/.ssh".format(env.deployer_user, env.deployer_user))
    con.sudo("chgrp -R {} /home/{}/.ssh".format(env.deployer_group, env.deployer_user))


def ensure_local_keys(con: Connection):
    ssh_folder = Path.home() / ".ssh"
    private_key = ssh_folder / "id_rsa"
    public_key = ssh_folder / "id_rsa.pub"

    os.makedirs(ssh_folder, exist_ok=True)

    current_files = sum([private_key.is_file(), public_key.is_file()])

    if not current_files in (0, 2):
        raise RuntimeError(f"Invalid key state ({current_files})")

    if current_files == 0:
        info("Creating local ssh keys")
        con.local('ssh-keygen -t rsa -b 2048 -f {0} -N ""'.format(private_key))


def update_keys(con: Connection):
    info("Updating keys")
    public_key_path = Path.home() / ".ssh/id_rsa.pub"
    user = env.deployer_user

    if user == "root":
        authorized_keys_path = f"/root/.ssh/authorized_keys"
        ssh_folder = f"/root/.ssh"
    else:
        authorized_keys_path = f"/home/{user}/.ssh/authorized_keys"
        ssh_folder = f"/home/{user}/.ssh"

    public_key = public_key_path.read_text("utf8").strip()

    con.sudo(f'mkdir -p "{ssh_folder}"')

    try:
        result = con.run(f"cat {authorized_keys_path}")
        current_keys = result.stdout.strip().splitlines()
    except UnexpectedExit:
        current_keys = []

    current_keys.sort()
    new_current_keys = [x for x in current_keys if not x.startswith("#")]
    new_current_keys.append(public_key)
    new_current_keys = list(set(new_current_keys))
    new_current_keys.sort()

    if new_current_keys != current_keys:
        info("Updating authorized_keys")
        authorized_keys = "\n".join(new_current_keys) + "\n"
        Path("tmp").write_text(authorized_keys, "utf8")
        con.put("tmp", "/tmp/authorized_keys")
        con.sudo(f"mv /tmp/authorized_keys {authorized_keys_path}")
        Path("tmp").unlink()

    info("Fixing permissions of user's .ssh files")
    con.sudo(f"chmod 700 {ssh_folder}")
    con.sudo(f"chmod 600 {authorized_keys_path}")

    ownership = f"{env.deployer_user}:{env.deployer_password}"
    con.sudo(f"chown {ownership} {ssh_folder}")
    con.sudo(f"chown {ownership} {authorized_keys_path}")


def setup_sshd_config(con: Connection):
    config = "/etc/ssh/sshd_config"
    con.sudo(f"cp {config} {config}.backup")
    con.sudo(f"sed -i 's/^UsePAM yes/UsePAM no/' {config}")
    con.sudo(f"sed -i 's/^PermitRootLogin yes/PermitRootLogin no/' {config}")
    con.sudo(
        f"sed -i 's/^#PasswordAuthentication yes/PasswordAuthentication no/' {config}"
    )
    con.sudo("service ssh reload")


def install_libraries(con: Connection):
    con.sudo("apt-get update")
    con.sudo("apt-get upgrade -y")

    libraries = (
        "build-essential",
        "cmake",
        "curl",
        "git",
        "nano",
        "python3",
        "python3-pip",
    )
    con.sudo(f"apt-get install {' '.join(libraries)} -y")

    install_fish(con)
    install_docker(con)


def install_fish(con: Connection):
    con.run(
        "echo 'deb http://download.opensuse.org/repositories/shells:/fish:/release:/3/Debian_10/ /' | sudo tee /etc/apt/sources.list.d/shells:fish:release:3.list"
    )
    con.run(
        "curl -fsSL https://download.opensuse.org/repositories/shells:fish:release:3/Debian_10/Release.key | gpg --dearmor | sudo tee /etc/apt/trusted.gpg.d/shells_fish_release_3.gpg > /dev/null"
    )
    con.sudo("apt update")
    con.sudo("apt install fish -y")

    con.sudo(f"chsh -s /usr/bin/fish {env.deployer_user}")

    # Oh My Fish
    con.run("curl -L https://get.oh-my.fish > /tmp/omf.sh")
    con.run("fish /tmp/omf.sh --noninteractive")
    con.run("rm /tmp/omf.sh")
    con.run("ps")
    con.run("echo omf install agnoster | fish")
    con.run("echo omf theme agnoster | fish")
    con.run("echo omf install bang-bang | fish")


def install_docker(con: Connection):
    con.run("curl -fsSL https://get.docker.com -o /tmp/get-docker.sh")
    con.sudo("sh /tmp/get-docker.sh")
    con.run("rm /tmp/get-docker.sh")
    con.sudo(f"usermod -aG docker {env.deployer_user}")
    con.run("python3 -m pip install docker-compose")
    con.run(f"echo fish_add_path /home/{env.deployer_user}/.local/bin/ | fish")


def deploy_services(con: Connection):
    con.run(f"git clone https://{env.github_token}@github.com/sralloza/services.git /srv")

if __name__ == "__main__":
    main()
