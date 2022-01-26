# `authorized-keys`

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
      --ssh-key            Use SSH key
      --user string        Login user

Global Flags:
      --debug   Enable debug
```

As said before, it will download the public SSH keys from AWS and update them. You can use SSH with an already valid ssh-key or the user's password. If you want to use your SSH key use the flag `--ssh-key`. It will get your private SSH key located at `~/.ssh/id_rsa` by default. Right now the private key path is not configurable. If you want to use the password to log in, use the `--password` flag.
