# Rpi-provisioner

_Have your raspberry pi ready to go using a couple commands._

When your Raspberry Pi with all your projects dies it's a real pain to set it up again. Install your favourite shell, update all packagaes, set up the ssh connection, the static ip address...

That's why this repo was created. The first version was created in Python, but the sudo password detection was really buggy, so now it's rewritten in go.

## Problems & Solutions

### Initial setup

What happens if you don't have an spare screen and keyboard? Don't worry, this script has your back. After flashing your raspbian image into your ssh card, execute the `boot` command. It will setup the ssh server and optionally a **wifi connection** to work the first time you turn your raspberry on. By default it will also add some lines to `cmdline.txt` to enable some features needed to run a k3s cluster. If you want to disable it, pass `--cmdline=""` to the `boot` command.

Note: you must pass the path of your sd card (the `BOOT_PATH` argument). In windows it will likely be `E:/`, `F:/` or something similar.

### Raspberry's initial IPv4

When you plug in your raspberry after enabling ssh connection, you can't know what its IPv4 is unless you have a spare screen or you have access to your router's configuration.

This is where the `find` command comes in really handy. You only have to specify your network IP (like `--subnet=192.168.0.1/24` or `--subnet=10.0.0.1/24`). Well, in reality you don't have to even do this, because by default the program will get your local IP (excluding the WSL interface) and use it with a 24-bit mask to build your presumably network IP, so `LOCAL_IP/24`.

There are some useful flags to make this command work, but the defaults will probably be just OK. For more info, refer to the [find command docs](#find).

### SSH Keys management

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

### Networking

You will probably ssh often into your rapsberry pi, you chances are you want to setup a static IP address. It's really simple to do it, just use the [network](#network) command.

## Commands

### boot

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
```

### find

```shell
$ rpi-provisioner find --help
Find your raspberry pi in your local network using SSH.

Usage:
  rpi-provisioner find [flags]

Flags:
  -h, --help              help for find
      --live              Print valid hosts right after found
      --password string   Password to login via ssh (default "raspberry")
      --port int          Port to connect via ssh (default 22)
      --subnet string     Subnet to find the raspberry
      --time              Show hosts processing time
      --timeout int       Timeout in ns to wait in ssh connections (default 1)
      --user string       User to login via ssh (default "pi")
```

More info:

- `--subnet`: this is the most important flag. You won't probably use it, but with this flag you can specify your local network's IP. If you left this blank, the program will try to generate it from your local IP address. If it is wrong, use this flag to really find your raspberry pi in your local network.
- `--live`: By default when you start the analysis, the valid raspberry's IP will only be shown at the end. You can use this flag to see as soon as it is discovered.
- `--user & --password`: login user and password to use via SSH. The default credentials for raspbian are `pi:raspberry`, as the default values for each flag. If you use another OS you can use this flags to change it.
- `--port`: just in case the default SSH port is not 22, use this flag to set it right.
- `--time`: instead of showing `Done` when the scan finishes, it will display `Done (x seconds)`, showing the analysis time.
- `--timeout`: Timeout in nanoseconds to wait in SSH connections. It is directly passed to the SSH Dial method. To be fair I don't really know if this works, so don't use it. By default is 1, but I don't know if it affects performance. If you know more about this flag, feel free to open an issue or a PR correcting the documentation.

### authorized-keys

```shell
$ rpi-provisioner authorized-keys --help
Download keys from the S3 bucket and update them.

Usage:
  rpi-provisioner authorized-keys [flags]

Flags:
  -h, --help               help for authorized-keys
      --host string        Server host
      --keys-path string   Local keys file path. You can select the public key file or a file containing multiple public keys.
      --password string    Login password
      --port int           Server SSH port (default 22)
      --s3-path string     Amazon S3 path. Must match the pattern region/bucket/file
      --ssh-key            Use ssh key
      --user string        Login user
```

As said before, it will download the public ssh keys from AWS and update them. You can use ssh with an already valid ssh-key or the user's password. If you want to use your ssh key use the flag `--ssh-key`. It will get your private ssh key located at `~/.ssh/id_rsa` by default. Right now the private key path is not configurable. If you want to use the password to log in, use the `--password` flag.

### network

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
      --ssh-key           Use ssh key
      --user string       Login user
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
  rpi-provisioner layer2 [flags]

Flags:
  -h, --help          help for layer2
      --host string   Server host
      --port int      Server SSH port (default 22)
      --user string   Login user
```

## Examples of how I really use each command

### boot example

```shell
rpi-provisioner boot --wifi-ssid $WIFI_SSID --wifi-pass $WIFI_PASS E:/
```

### find example

```shell
rpi-provisioner find --time --live
```

### authorized-keys example

rpi-provisioner authorized-keys --ssh-key --host $RASPBERRY_IP --user $USER --s3-path $S3_REGION/$S3_BUCKET/$S3_FILE

### layer1 example

```shell
rpi-provisioner layer1 --deployer-user $NEW_USER --deployer-password $NEW_PASSWORD --host $RASPBERRY_IP --hostname $HOSTNAME --s3-path $S3_REGION/$S3_BUCKET/$S3_FILE
```

### layer2 example

```shell
rpi-provisioner layer2 --user $USER --host $RASPBERRY_IP
```
