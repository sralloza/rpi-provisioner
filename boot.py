import sys
from contextlib import contextmanager
from os import getenv
from pathlib import Path

from dotenv import load_dotenv
from jinja2 import Environment, FileSystemLoader, select_autoescape

from config import settings

load_dotenv()
DRIVE = Path("E:/")


@contextmanager
def info(start: str, end: str = "OK", add_dots: bool = True):
    if add_dots:
        start += "." * 3
    print(start, end=" ")
    try:
        yield
        print(end)
    except Exception as exc:
        print("FAILED:", repr(exc))
        # sys.exit(1)


def enable_ssh():
    with info("Creating ssh file to enable ssh"):
        DRIVE.joinpath("ssh").touch()


def render(template_path, env_vars):
    env = Environment(
        loader=FileSystemLoader(Path(__file__).parent),
        autoescape=select_autoescape(["html", "xml"]),
        trim_blocks=True,
        lstrip_blocks=True,
    )
    template = env.get_template(template_path)
    return template.render(**env_vars)


def setup_wifi_connection():
    with info("Copying wpa_supplicant.conf to setup WiFi connection"):
        env = {
            "wifi_ssid": getenv("WIFI_SSID"),
            "wifi_pass": getenv("WIFI_PASS"),
            "country_code": getenv("COUNTRY_CODE"),
        }
        wpa_data = render("wpa_supplicant.conf.j2", env)
        wpa_data = Path(__file__).with_name("wpa_supplicant.conf").read_bytes()
        DRIVE.joinpath("wpa_supplicant.conf").write_bytes(wpa_data)


def edit_cmdline():
    with info("Editing cmdline.txt to add static ip..."):
        cmd_path = DRIVE.joinpath("cmdline.txt")
        cmdline_lines = cmd_path.read_text("utf8").splitlines()
        append_text = f" ip={settings.new_host}"
        if not cmdline_lines[0].endswith(append_text):
            cmdline_lines[0] += append_text
            cmd_path.with_name("cmdline.txt.bkp").write_bytes(cmd_path.read_bytes())
        if len(cmdline_lines) == 1:
            cmdline_lines.append("")
        cmd_path.write_text("\n".join(cmdline_lines), "utf8")


def main():
    print(f"Using drive {DRIVE.as_posix()!r}")
    if not DRIVE.is_dir():
        print("Drive does not exist, exiting")
        # sys.exit(1)

    enable_ssh()
    setup_wifi_connection()
    # Disabled because it only works for eth
    # edit_cmdline()


if __name__ == "__main__":
    main()
