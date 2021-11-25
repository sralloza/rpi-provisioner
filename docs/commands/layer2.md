# `layer2`

```shell
$ rpi-provisioner layer2 --help
Layer 2 uses the deployer user and bash. It will perform the following tasks:
- Update and upgrade packages
- Install libraries: build-essential, cmake, cron, curl, git, libffi-dev, nano, python3-pip, python3, wget
- Install fish
- Install docker

Usage:
  rpi-provisioner layer2 [flags]

Flags:
  -h, --help          help for layer2
      --host string   Server host
      --port int      Server SSH port (default 22)
      --user string   Login user

Global Flags:
      --debug   Enable debug
```
