from pydantic import BaseSettings, DirectoryPath


class Settings(BaseSettings):
    host: str
    new_host: str

    hostname: str

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

    class Config:
        env_file = ".env"
        env_file_encoding = "utf-8"


settings = Settings()


if __name__ == "__main__":
    print(repr(settings))
