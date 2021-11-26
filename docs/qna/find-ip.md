# Raspberry's initial IPv4

When you plug in your raspberry after enabling ssh connection, you can't know what its IPv4 is unless you have a spare screen or you have access to your router's configuration.

This is where the `find` command comes in really handy. You only have to specify your network IP (like `--subnet=192.168.0.1/24` or `--subnet=10.0.0.1/24`). Actually, you don't have to even do this, because by default the program will get your local IP (excluding the WSL interface) and use it with a 24-bit mask to build your presumably network IP, so `$LOCAL_IP/24`.

!!! info "Find your raspberry's IP with other login"
    Note that the default user and password of the `find` command are the default ones for Raspbian (`pi:raspberry`). If you have changed the user or the password, use the flags `--user` and `--password` to find your raspberry. Otherwise, the script will try to login with the default user and password and it will fail, which may look like that the rasbperry's SSH is not working or it's shutdown.

There are some useful flags to make this command work, but the defaults will probably be just OK. For more info, refer to the [find command docs](../commands/find.md).
