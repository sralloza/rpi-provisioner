# Rpi-provision

_Have your raspberry pi ready to go using a couple commands._

When your Raspberry Pi with all your projects dies it's a real pain to set it up again. Install your favourite shell, update all packagaes, set up the ssh connection, the static ip address...

That's why this repo was created. The first version was created in Python, but the sudo password detection was really buggy, so now it's rewritten in go.

## Problems & Solutions

### SSH Keys

I have some PCs with ssh keys, so naturally I would want to be able to ssh into the Raspberry from any of my PCs.

But, what happens if I change one key? Do I have to manually add the key to each Raspberry I have?

With this script, no. You just have to change your public ssh in the file. You just need to write your public ssh keys into a json file and upload it to an s3 bucket.

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
Enable ssh, modify cmdline.txt and setup wifi connection

Usage:
  rpi-provision boot [BOOT_PATH] [flags]

Flags:
      --cmdline stringArray   Extra args to append to cmdline.txt (default [cgroup_enable=cpuset,cgroup_enable=memory,cgroup_memory=1])
  -h, --help                  help for boot
      --wifi-pass string      WiFi password
      --wifi-ssid string      WiFi SSID

Global Flags:
      --debug   Enable debug
```

What happens if you don't have an spare screen and keyboard? Don't worry, this script has your back. After flashing your raspbian image into your ssh card, execute the `boot` command. It will setup the ssh server and optionally a wifi connection to work the first time you turn your raspberry on. By default it will also add some lines to `cmdline.txt` to enable some features needed to run a k3s cluster. If you want to disable it, pass `--cmdline=""` to the `boot` command.

Note: you must pass the path of your sd card (the `BOOT_PATH` argument). In windows it will likely be `E:/`, `F:/` or something similar.

### authorized-keys

```shell
Download keys from the S3 bucket and update them.

Usage:
  rpi-provision authorized-keys [flags]

Flags:
  -h, --help              help for authorized-keys
      --host string       Server host
      --password string   Login password
      --port int          Server SSH port (default 22)
      --s3-path string    Amazon S3 path. Must match the pattern region/bucket/file
      --ssh-key           Use ssh key
      --user string       Login user

Global Flags:
      --debug   Enable debug
```

As said before, it will download the public ssh keys from AWS and update them. You can use ssh with an already valid ssh-key or the user's password. If you want to use your ssh key use the flag `--ssh-key`. It will get your private ssh key located at `~/.ssh/id_rsa` by default. Right now the private key path is not configurable. If you want to use the password to log in, use the `--password` flag.

### network

### layer1

### layer2
