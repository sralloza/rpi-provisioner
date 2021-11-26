# Set up SSH and WiFi before first boot

What happens if you don't have an spare screen and keyboard? Don't worry, this script has your back. After flashing your raspbian image into your ssh card, execute the `boot` command. It will setup the ssh server and optionally a **wifi connection** to work the first time you turn your raspberry on. By default it will also add some lines to `cmdline.txt` to enable some features needed to run a k3s cluster. If you want to disable it, pass `--cmdline=""` to the `boot` command.

Note: you must pass the path of your sd card (the `BOOT_PATH` argument). In windows it will likely be `E:/`, `F:/` or something similar.
