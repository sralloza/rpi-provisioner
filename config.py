from pathlib import Path
from typing import List

from pydantic import BaseConfig


class Settings(BaseConfig):
    user: str = "sralloza"
    host: str = "localhost"
    login_user: str = "root"
    password: str = "password"
    full_name_user: str = "Diego Alloza"
    user_group: str = "sralloza"
    user_name: str = "sralloza"
    ssh_keys_dir: Path = Path(__file__).with_name("ssh-keys")


settings = Settings()
