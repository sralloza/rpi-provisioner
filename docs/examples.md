
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

```shell
rpi-provisioner authorized-keys --ssh-key --host $RASPBERRY_IP --user $USER --s3-path $S3_REGION/$S3_BUCKET/$S3_FILE
```

### layer1 example

```shell
rpi-provisioner layer1 --deployer-user $NEW_USER --deployer-password $NEW_PASSWORD --host $RASPBERRY_IP --hostname $HOSTNAME --s3-path $S3_REGION/$S3_BUCKET/$S3_FILE
```

**Important: it is highly recommended to reboot the raspberry after provisioning the layer 1. Doing so, the hostname will be effectively changed and installing the system dependencies will be less likely to return random errors.**

### layer2 example

```shell
rpi-provisioner layer2 --user $USER --host $RASPBERRY_IP
```
