# Examples of how I really use each command

## boot

```shell
rpi-provisioner boot --wifi-ssid $WIFI_SSID --wifi-pass $WIFI_PASS E:/
```

## find

```shell
rpi-provisioner find --time --live
```

## authorized-keys

```shell
rpi-provisioner authorized-keys --ssh-key --host $RASPBERRY_IP --user $USER --s3-path $S3_REGION/$S3_BUCKET/$S3_FILE
```

## network

```shell
rpi-provisioner --host $CURRENT_IP --ip $NEW_IP --user $USER --ssh-key
```

## layer1

```shell
rpi-provisioner layer1 --deployer-user $NEW_USER --deployer-password $NEW_PASSWORD --host $RASPBERRY_IP --hostname $HOSTNAME --s3-path $S3_REGION/$S3_BUCKET/$S3_FILE
```

!!! danger "Reboot after layer1"
    It is highly recommended to reboot the raspberry after provisioning the layer 1. Doing so, the hostname will be effectively changed and installing the system dependencies will be less likely to return random errors.

## layer2

```shell
rpi-provisioner layer2 --user $USER --host $RASPBERRY_IP
```

!!! warning "Slow execution"
    The second SSH command that `layer2` sends to the raspberry is `sudo apt-get upgrade -y`, so it's normal to take some time, even to appear *blocked*.
