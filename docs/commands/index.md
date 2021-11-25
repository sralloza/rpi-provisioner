# Commands

All commands have a debug global flag: `--debug`. It will enable debug mode, showing exactly which commands are executed via SSH, their output and their error. The error only appears if the command returns a non zero status code, otherwise it will be `<nil>`.

Logs format: `ssh: "REMOTE_COMMAND" -> ["COMMAND_STDOUT" | "COMMAND_STDERR" | ERROR]`

You will probably use the commands in this order:

1. Before the first boot, you need to enable `SSH` and the `WiFi` connection (if you don't have ethernet available). You can do it modifying files of the sdcard before the first time you turn the raspberry on. The [boot](./boot.md) command manages it.
2. After setting up the sdcard with the [boot](boot.md) command you will probably plug in and power your raspberry pi for the first time. In case you don't have a spare screen, keyboard and mouse or if you don't have access to your router configuration, you will have no idea what IP is assigned to your raspberry. Here is where the [find](find.md) command comes handy, go and take a look at its own docs.
3. Just after the first boot, you will find a simple bash command with an insecure ssh connection using the default user and password (`pi` and `raspberry` as everybody knows). The next step is to create a new user (from now on called the *deployer* user) to use instead of the default one, set up the ssh keys (because it's a pain to write the password every time you want to open a SSH connection), maybe set up a static IP address or setting a custom hostname (instead of calling it `raspberry` as the default), etc. The [layer1](layer1.md) is in charge of all of this.
