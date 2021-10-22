import sys
from contextlib import contextmanager
from functools import wraps
from os import getenv
from pathlib import Path

from dotenv import load_dotenv
from jinja2 import Environment, FileSystemLoader, select_autoescape

from config import settings

load_dotenv()
DRIVE = Path("E:/")


def call_count(func):
    called = 0

    @wraps(func)
    def wrapper(*args, **kwargs):
        nonlocal called
        called += 1
        kwargs["called"] = called
        return func(*args, **kwargs)

    return wrapper


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
        sys.exit(1)


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


@call_count
def add_to_cmdlines_txt(text: str, called: int = 0):
    print(called)
    append_text = " " + text
    cmd_path = DRIVE.joinpath("cmdline.txt")

    with info(f"Adding {text!r} to cmdline.txt"):
        cmdline_lines = cmd_path.read_text("utf8").splitlines()
        if append_text not in cmdline_lines[0]:
            cmdline_lines[0] += append_text
            if not called:
                cmd_path.with_name("cmdline.txt.bkp").write_bytes(cmd_path.read_bytes())
        if len(cmdline_lines) == 1:
            cmdline_lines.append("")
        cmd_path.write_text("\n".join(cmdline_lines), "utf8")


def edit_cmdline():
    # Disabled because it only works for ethernet connections
    # add_to_cmdlines_txt(f"ip={settings.new_host}")

    # Setup cmdlines for k3s
    add_to_cmdlines_txt("cgroup_enable=cpuset cgroup_enable=memory cgroup_memory=1")


def main():
    print(f"Using drive {DRIVE.as_posix()!r}")
    if not DRIVE.is_dir():
        print("Drive does not exist, exiting")
        sys.exit(1)

    enable_ssh()
    setup_wifi_connection()
    edit_cmdline()


if __name__ == "__main__":
    main()
