from pathlib import Path
from fabric import Connection

ROOT_PATH = Path(__file__).parent

ssh_keys_dir = ROOT_PATH / "ssh-keys"

c = Connection("rfenix")
ssh_keys_name = ssh_keys_dir / "test_prod_key"
# c.local("ssh-keygen -t rsa -b 2048 -f {0} -N \"\"".format(ssh_keys_name))
c.local(["echo hola", "echo adios"])
