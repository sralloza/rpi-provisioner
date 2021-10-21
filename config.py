from pydantic import BaseSettings, DirectoryPath


class Settings(BaseSettings):
    # host: str = "192.168.1.93"
    host: str = "localhost"
    new_host: str = "192.168.0.98"

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
    services_docker_path: DirectoryPath


settings = Settings()


if __name__ == "__main__":
    print(repr(settings))
