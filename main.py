import os
from pathlib import Path

from fabric import Connection, Config
from invoke.exceptions import UnexpectedExit

from config import settings

env = settings
config = Config(
    overrides={"sudo": {"password": settings.password}, "run": {"hide": True}}
)


def main():
    con = Connection("rfenix", config=config)
    # start_provision(con)
    return update_keys(con)
    r = con.sudo("cat /root/a")
    print(repr(r))
    print(repr(r.stdout.strip()))


def start_provision(con: Connection):
    ensure_local_keys(con)
    setup_sshd_config(con)
    update_keys(con)

    create_deployer_group(con)
    create_deployer_user(con)
    install_ansible_dependencies(con)

    con.run("service sshd reload")
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
    # Prevent root SSHing into the remote server
    con.run('sed "/etc/ssh/sshd_config" "^UsePAM yes" "UsePAM no"')
    con.run('sed "/etc/ssh/sshd_config" "^PermitRootLogin yes" "PermitRootLogin no"')
    con.run(
        'sed "/etc/ssh/sshd_config" "^#PasswordAuthentication yes" "PasswordAuthentication no"'
    )


def create_deployer_group(con: Connection):
    con.run("groupadd {}".format(env.user_group))
    con.run("mv /etc/sudoers /etc/sudoers-backup")
    con.run(
        f'(cat /etc/sudoers-backup; echo "%{env.user_group} ALL=(ALL) ALL") > /etc/sudoers'
    )
    con.run("chmod 440 /etc/sudoers")


def create_deployer_user(con: Connection):
    con.run(
        'adduser -c "{.full_name_user}" -m -g {.user_group} {.user_name}'.format(env)
    )
    con.run("passwd {}".format(env.user_name))
    con.run("usermod -a -G {} {}".format(env.user_group, env.user_name))
    con.run("mkdir /home/{}/.ssh".format(env.user_name))
    con.run("chown -R {} /home/{}/.ssh".format(env.user_name, env.user_name))
    con.run("chgrp -R {} /home/{}/.ssh".format(env.user_group, env.user_name))


def update_keys(con: Connection):
    public_key_path = Path.home() / ".ssh/id_rsa.pub"
    authorized_keys_path = f"/home/{env.user}/.ssh/authorized_keys"

    public_key = public_key_path.read_text("utf8").strip()

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
        authorized_keys = "\n".join(new_current_keys)
        Path("tmp").write_text(authorized_keys, "utf8")
        con.put("tmp", authorized_keys_path)
        Path("tmp").unlink()

def install_ansible_dependencies(con: Connection):
    # TODO: fix distro
    con.run("dnf install -y python-dnf")


def upgrade_server(con: Connection):
    # TODO: fix distro
    con.run("dnf upgrade -y")
    # optional command (necessary for Fedora 25)
    con.run("dnf install -y python")
    con.run("reboot")


if __name__ == "__main__":
    main()
