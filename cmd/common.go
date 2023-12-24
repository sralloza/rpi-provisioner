package cmd

var restartCleanDHCP string = "\nWarning: you must restart the server to remove old DHCP leases\n" +
	"  Consider rebooting the server and then execute the network command again\n" +
	"    ssh %s@%s sudo reboot\n"

var networkWarning string = "Warning: you have enabled static IP.\n" +
	"  You might lose connectivity to the server during the configuration\n\n"
