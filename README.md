# Rpi-provision

_Have your raspberry pi ready to go using a couple commands._

When your Raspberry Pi with all your projects dies it's a real pain to set it up again. Install your favourite shell, update all packagaes, set up the ssh connection, the static ip address...

That's why this repo was created. The first version was created in Python, but the sudo password detection was really buggy, so now it's rewritten in go.

## Problems & Solutions

### SSH Keys

I have some PCs with ssh keys, so naturally I would want to be able to ssh into the Raspberry from any of my PCs.

But, what happens if I change one key? Do I have to manually add the key to each Raspberry I have?

With this script, no. You just have to change your public ssh in the file. You just need to write your public ssh keys into a json file and upload it to an s3 bucket.

Note: if you don't want to use AWS S3 to store your keys file, use the `--keys-path` command to specify the path to the file where you store your public keys.

Example:

```json
{
  "key-id-1": "public-ssh-key-1",
  "key-id-1": "public-ssh-key-1"
}
```

Then you set your AWS env vars (`AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`). If you don't have them, a simple google search will tell you how to generate them. You will need to tell the script where your file containing the public ssh keys is in AWS. You do it with the `--s3-path` flag: `--s3-path=<REGION>/<BUCKET_NAME>/<FILE_NAME>`. If you don't use this convention the script will complain and raise an error.

We have covered how to store your public ssh keys. How can you update the ssh keys in your raspberry's authorized_keys? Simple, just use the [authorized-keys](#authorized-keys) command (or the [layer1](#layer1) command if you set up the raspberry for the first time).

## Commands

### boot

```shell
$ rpi-provisioner boot --help
Enable ssh, modify cmdline.txt and setup wifi connection

Usage:
  rpi-provision boot [BOOT_PATH] [flags]

Flags:
      --cmdline stringArray   Extra args to append to cmdline.txt (default [cgroup_enable=cpuset,cgroup_enable=memory,cgroup_memory=1])
      --country string        Country code (2 digits) (default "ES")
  -h, --help                  help for boot
      --wifi-pass string      WiFi password
      --wifi-ssid string      WiFi SSID

Global Flags:
      --debug   Enable debug                                                                                              21:33:29
```

What happens if you don't have an spare screen and keyboard? Don't worry, this script has your back. After flashing your raspbian image into your ssh card, execute the `boot` command. It will setup the ssh server and optionally a wifi connection to work the first time you turn your raspberry on. By default it will also add some lines to `cmdline.txt` to enable some features needed to run a k3s cluster. If you want to disable it, pass `--cmdline=""` to the `boot` command.

Note: you must pass the path of your sd card (the `BOOT_PATH` argument). In windows it will likely be `E:/`, `F:/` or something similar.

### authorized-keys

```shell
$ rpi-provisioner authorized-keys --help
Download keys from the S3 bucket and update them.

Usage:
  rpi-provision authorized-keys [flags]

Flags:
  -h, --help               help for authorized-keys
      --host string        Server host
      --keys-path string   Local keys file path. You can select the public key file or a file containing multiple public keys.
      --password string    Login password
      --port int           Server SSH port (default 22)
      --s3-path string     Amazon S3 path. Must match the pattern region/bucket/file
      --ssh-key            Use ssh key
      --user string        Login user

Global Flags:
      --debug   Enable debug
```

As said before, it will download the public ssh keys from AWS and update them. You can use ssh with an already valid ssh-key or the user's password. If you want to use your ssh key use the flag `--ssh-key`. It will get your private ssh key located at `~/.ssh/id_rsa` by default. Right now the private key path is not configurable. If you want to use the password to log in, use the `--password` flag.

### network

```shell
$ rpi-provisioner network --help
Set up static ip for eth0 and wlan0.

Usage:
  rpi-provision network [flags]

Flags:
  -h, --help              help for network
      --host string       Server host
      --ip ip             New IP
      --password string   Login password
      --port int          Server SSH port (default 22)
      --ssh-key           Use ssh key
      --user string       Login user

Global Flags:
      --debug   Enable debug
```

This commands just edits the dhcpd config to set an static IP Address for both eth0 and wlan0. It provisions the same IP Adress for both interfaces, but it gives priority to eth0.

### layer1

```shell
$ rpi-provisioner layer1 --help
Layer 1 uses the default user and bash shell. It will perform the following tasks:
 - Create deployer user
 - Set hostname
 - Setup ssh config and keys
 - Disable pi login
 - [optional] static ip configuration

Usage:
  rpi-provision layer1 [flags]

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

### layer2

```shell
$ rpi-provisioner layer2 --help
Layer 2 uses the deployer user and bash. It will perform the following tasks:
- Update and upgrade packages
- Install libraries: build-essential, cmake, cron, curl, git, libffi-dev, nano, python3-pip, python3, wget
- Install fish
- Install docker

Usage:
  rpi-provision layer2 [flags]

Flags:
  -h, --help          help for layer2
      --host string   Server host
      --port int      Server SSH port (default 22)
      --user string   Login user

Global Flags:
      --debug   Enable debug
```
