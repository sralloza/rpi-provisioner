# `How can I set up a static IP address?`

You will probably SSH often into your rapsberry pi, you chances are you want to set up a static IP address. It's really simple to do it, just use the [network](../commands/network.md) command:

To login via user & password:

```shell
rpi-provisioner --host $CURRENT_IP_OR_HOSTNAME --ip $NEW_IP --user $USER --password $PASSWORD
```

To login via SSH key (it's assumed to be at `$HOME/.ssh/id_rsa`):

```shell
rpi-provisioner --host $CURRENT_IP_OR_HOSTNAME --ip $NEW_IP --user $USER --ssh-key
```
