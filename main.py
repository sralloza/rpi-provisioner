import os
from pathlib import Path

from fabric import Connection, Config
from invoke.exceptions import UnexpectedExit

from config import settings

env = settings
config = Config(
    overrides={
        "sudo": {"password": settings.initial_login_password},
        # "run": {"hide": True},
    }
)


def main():
    con = Connection(
        host=env.host,
        user=env.initial_login_user,
        connect_kwargs={"password": env.initial_login_password},
        config=config,
    )
    start_provision(con)


def start_provision(con: Connection):
    ensure_local_keys(con)
    setup_sshd_config(con)

    create_deployer_group(con)
    create_deployer_user(con)
    update_keys(con)
    print("Done")
    return
    install_ansible_dependencies(con)

    upgrade_server(con)


def ensure_local_keys(con: Connection):
    ssh_folder = Path.home() / ".ssh"
    private_key = ssh_folder / "id_rsa"
    public_key = ssh_folder / "id_rsa.pub"

    os.makedirs(ssh_folder, exist_ok=True)

    current_files = sum([private_key.is_file(), public_key.is_file()])

    if not current_files in (0, 2):
        raise RuntimeError(f"Invalid key state ({current_files})")

    if current_files == 0:
        con.local('ssh-keygen -t rsa -b 2048 -f {0} -N ""'.format(private_key))


def setup_sshd_config(con: Connection):
    config = "/etc/ssh/sshd_config"
    con.run(f"sed -i 's/^UsePAM yes/UsePAM no/' {config}")
    con.run(f"sed -i 's/^PermitRootLogin yes/PermitRootLogin no/' {config}")
    con.run(
        f"sed -i 's/^#PasswordAuthentication yes/PasswordAuthentication no/' {config}"
    )
    con.run("service ssh reload")


def create_deployer_group(con: Connection):
    print("Creating deployer group")
    con.run("groupadd {}".format(env.deployer_group))
    if con.run("test -f /etc/sudoers", warn=True).failed:
        print("Creating /etc/sudoers")
        con.run("touch /etc/sudoers")
    else:
        print("Creating backup of /etc/sudoers")
        con.run("mv /etc/sudoers /etc/sudoers-backup")

    con.run(
        f'(cat /etc/sudoers-backup; echo "%{env.deployer_group} ALL=(ALL) ALL") > /etc/sudoers'
    )
    con.run("chmod 440 /etc/sudoers")


def create_deployer_user(con: Connection):
    print("Creating deployer user")
    con.run(
        f"adduser --gecos '{env.full_name_user}' --disabled-password --ingroup {env.deployer_group} {env.deployer_user}"
    )
    print("Setting password...")
    con.run(f"echo {env.deployer_user}:{env.deployer_password} | chpasswd")
    print("Password set")
    con.run("usermod -a -G {} {}".format(env.deployer_group, env.deployer_user))
    con.run("mkdir /home/{}/.ssh".format(env.deployer_user))
    con.run("chown -R {} /home/{}/.ssh".format(env.deployer_user, env.deployer_user))
    con.run(
        "chgrp -R {} /home/{}/.ssh".format(env.deployer_group, env.deployer_user)
    )


def update_keys(con: Connection):
    print("Updating keys")
    public_key_path = Path.home() / ".ssh/id_rsa.pub"
    user = env.deployer_user

    if user == "root":
        authorized_keys_path = f"/root/.ssh/authorized_keys"
        ssh_folder = f"/root/.ssh"
    else:
        authorized_keys_path = f"/home/{user}/.ssh/authorized_keys"
        ssh_folder = f"/home/{user}/.ssh"

    public_key = public_key_path.read_text("utf8").strip()

    con.run(f'mkdir -p "{ssh_folder}"')

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
        print("Updating authorized_keys")
        authorized_keys = "\n".join(new_current_keys) + "\n"
        Path("tmp").write_text(authorized_keys, "utf8")
        con.put("tmp", authorized_keys_path)
        Path("tmp").unlink()

    print("Fixing permissions of .ssh files")
    con.run(f"chmod 700 {ssh_folder}")
    con.run(f"chmod 600 {authorized_keys_path}")
    con.run(f"chown {env.deployer_user}:{env.deployer_password} {authorized_keys_path}")


def install_ansible_dependencies(con: Connection):
    # TODO: fix distro
    con.run("dnf install -y python-dnf")


def upgrade_server(con: Connection):
    # TODO: fix distro
    con.run("apt-get update")
    con.run("apt-get upgrade -y")
    # con.run("apt-get install sudo")

    # optional command (necessary for Fedora 25)
    con.run("apt install -y python")

    # TODO: uncomment in prod
    # con.run("reboot")


if __name__ == "__main__":
    main()
