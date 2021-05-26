from pathlib import Path
from typing import List

from pydantic import BaseConfig


class Settings(BaseConfig):
    user: str = "root"
    hosts: List[str] = ["<remote-server-ip>"]
    password: str = "<remote-server-password>"
    full_name_user: str = "<your-name>"
    user_group: str = "deployers"
    user_name: str = "deployer"
    ssh_keys_dir: Path = Path(__file__).with_name("ssh-keys")


settings = Settings()
