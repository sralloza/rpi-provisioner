# `How can I set up a static IP address?`

You will probably ssh often into your rapsberry pi, you chances are you want to setup a static IP address. It's really simple to do it, just use the [network](../commands/network.md) command:

To login via user & password:

```shell
rpi-provisioner --host $CURRENT_IP --ip $NEW_IP --user $USER --password $PASSWORD
```

To login via ssh key (it's assumed to be at `$HOME/.ssh/id_rsa`):

```shell
rpi-provisioner --host $CURRENT_IP --ip $NEW_IP --user $USER --ssh-key
```
