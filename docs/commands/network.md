# `network`

```shell
$ rpi-provisioner network --help
Set up static ip for eth0 and wlan0.

Usage:
  rpi-provisioner network [flags]

Flags:
  -h, --help              help for network
      --host string       Server host
      --ip ip             New IP
      --password string   Login password
      --port int          Server SSH port (default 22)
      --ssh-key           Use SSH key
      --user string       Login user

Global Flags:
      --debug   Enable debug
```

This commands just edits the dhcpd config to set an static IP Address for both eth0 and wlan0. It provisions the same IP Adress for both interfaces, but it gives priority to eth0.
