# `boot`

```shell
$ rpi-provisioner boot --help
Enable ssh, modify cmdline.txt and setup wifi connection

Usage:
  rpi-provisioner boot [BOOT_PATH] [flags]

Flags:
      --cmdline stringArray   Extra args to append to cmdline.txt (default [cgroup_enable=cpuset,cgroup_enable=memory,cgroup_memory=1])
      --country string        Country code (2 digits) (default "ES")
  -h, --help                  help for boot
      --wifi-pass string      WiFi password
      --wifi-ssid string      WiFi SSID

Global Flags:
      --debug   Enable debug
```

What happens if you don't have an spare screen and keyboard? Don't worry, this script has your back. After flashing your raspbian image into your ssh card, execute the `boot` command. It will setup the ssh server and optionally a **wifi connection** to work the first time you turn your raspberry on. By default it will also add some lines to `cmdline.txt` to enable some features needed to run a k3s cluster. If you want to disable it, pass `--cmdline=""` to the `boot` command.

Note: you must pass the path of your sd card (the `BOOT_PATH` argument). In windows it will likely be `E:/`, `F:/` or something similar.
