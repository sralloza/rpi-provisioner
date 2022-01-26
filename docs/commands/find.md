# `find`

```shell
$ rpi-provisioner find --help
Find your raspberry pi in your local network using SSH.

Usage:
  rpi-provisioner find [flags]

Flags:
  -h, --help              help for find
      --live              Print valid hosts right after found
      --password string   Password to login via SSH (default "raspberry")
      --port int          Port to connect via SSH (default 22)
      --subnet string     Subnet to find the raspberry
      --time              Show hosts processing time
      --timeout int       Timeout in ns to wait in SSH connections (default 1)
      --user string       User to login via SSH (default "pi")

Global Flags:
      --debug   Enable debug
```

More info:

- `--subnet`: this is the most important flag. You won't probably use it, but with this flag you can specify your local network's IP. If you left this blank, the program will try to generate it from your local IP address. If it is wrong, use this flag to really find your raspberry pi in your local network.
- `--live`: By default when you start the analysis, the valid raspberry's IP will only be shown at the end. You can use this flag to see as soon as it is discovered.
- `--user & --password`: login user and password to use via SSH. The default credentials for raspbian are `pi:raspberry`, as the default values for each flag. If you use another OS you can use this flags to change it.
- `--port`: just in case the default SSH port is not 22, use this flag to set it right.
- `--time`: instead of showing `Done` when the scan finishes, it will display `Done (x seconds)`, showing the analysis time.
- `--timeout`: Timeout in nanoseconds to wait in SSH connections. It is directly passed to the SSH Dial method. To be fair I don't really know if this works, so don't use it. By default is 1, but I don't know if it affects performance. If you know more about this flag, feel free to open an issue or a PR correcting the documentation.
