# `layer1`

```shell
$ rpi-provisioner layer1 --help
Layer 1 uses the default user and bash shell. It will perform the following tasks:
 - Create deployer user
 - Set hostname
 - Set up SSH config and keys
 - Disable pi login
 - [optional] static ip configuration

Usage:
  rpi-provisioner layer1 [flags]

Flags:
      --deployer-password string   Deployer password
      --deployer-user string       Deployer user
  -h, --help                       help for layer1
      --host string                Server host
      --hostname string            Server hostname
      --keys-path string           Local keys file path. You can select the public key file or a file containing multiple public keys.
      --login-password string      Login password
      --login-user string          Login user
      --port int                   Server SSH port (default 22)
      --root-password string       Root password
      --s3-path string             Amazon S3 path. Must match the pattern region/bucket/file
      --static-ip ip               Set up the static ip for eth0 and wlan0

Global Flags:
      --debug   Enable debug
```

**Important: it is highly recommended to reboot the raspberry after provisioning the layer 1. Doing so, the hostname will be effectively changed and installing the system dependencies will be less likely to return random errors.**
