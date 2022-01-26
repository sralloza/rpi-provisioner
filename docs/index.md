# Raspberry Provisioner

> Have your raspberry pi ready to go using a couple commands.

It's a real pain when your Raspberry Pi dies due to sdcard corruption and you have to set it up again. Set up the SSH connection, a static IP address, set up your favourite shell, update the packages...

That's why [rpi-provisioner](./) was created. There is a command for (almost) any tedios part of having your raspberry up and running again.

Before creating this CLI I used to spend an entire afternoon just to setup my Raspberry again (which I use as a web server). Now, just about 15 minutes (most of the which is just executing `apt upgrade` remotely).

[rpi-provisioner](./) is created with the premise that someone can have the minimum hardware to operate it: a computer (may be a laptop), a Raspberry and a cable to plug it. You don't need an extra screen, mouse or keyboard to use it.

To begin using `rpi-provisioner`, go to the [commands](commands/index.md) index page or search for the feature you want below. Or you could go directly to find your question in [Questions and Answers](qna/index.md), it's your call.

## Features

Here is a list of all the features that [rpi-provisioner](./) has and the command that implements them.

- Enable SSH before first boot: [`boot` command](commands/boot.md)
- Set up WiFi connection before first boot: [`boot` command](commands/boot.md)
- Modify boot image to enable kubernetes cluster: [`boot` command](commands/boot.md)
- Find your Raspberry's IP in your local network: [`find` command](commands/find.md)
- Create new user: [`layer1` command](commands/layer1.md)
- Setup SSH access using SSH keys: [`layer1` command](commands/layer1.md), [`authorized-keys` command](commands/authorized-keys.md)
- Improve SSH security: [`layer1` command](commands/layer1.md)
- Set hostname: [`layer1` command](commands/layer1.md)
- Set static IP: [`layer1` command](commands/layer1.md), [`network` command](commands/network.md)
- Update system libraries: [`layer2` command](commands/layer2.md)
- Install `fish shell`: [`layer2` command](commands/layer2.md)
- Install `oh-my-fish`: [`layer2` command](commands/layer2.md)
- Install `docker`: [`layer2` command](commands/layer2.md)
- Install `docker-compose`: [`layer2` command](commands/layer2.md)

## Problems & Solutions

### Initial setup


### Networking
