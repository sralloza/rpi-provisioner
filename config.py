from pydantic import BaseSettings


class Settings(BaseSettings):
    # host: str = "192.168.1.93"
    host: str = "localhost"

    initial_login_user: str = "pi"
    initial_login_group: str = "pi"
    initial_login_password: str = "raspberry"

    root_password: str = "rootp"

    deployer_user: str = "deployer"
    deployer_password: str = "deployer"
    deployer_group: str = "deployer"
    full_name_user: str = "Deployer"

    github_token: str
    production: bool = False


settings = Settings()
