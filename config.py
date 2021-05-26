from pydantic import BaseConfig


class Settings(BaseConfig):
    host: str = "localhost"

    initial_login_user: str = "root"
    initial_login_password: str = "password"

    deployer_user: str = "deployer"
    deployer_password: str = "deployer"
    deployer_group: str = "deployer"
    full_name_user: str = "Deployer"


settings = Settings()
