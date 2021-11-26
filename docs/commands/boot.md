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
