# Commands

The command subpages only contains the commands ussage, which can be obtained executing `rpi-provisioner <COMMAND> --help`. To find a particular solution to your use case, refer to the [QnA](../qna/index.md).

## Debug flag

All commands have a debug global flag: `--debug`. It will enable debug mode, showing exactly which commands are executed via SSH, their output and their error. The error only appears if the command returns a non zero status code, otherwise it will be `<nil>`.

Logs format: `ssh: "REMOTE_COMMAND" -> ["COMMAND_STDOUT" | "COMMAND_STDERR" | ERROR]`
