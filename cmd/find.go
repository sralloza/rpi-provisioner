package cmd

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/sralloza/rpi-provisioner/ssh"
)

type FindArgs struct {
	subnet    string
	user      string
	password  string
	useSSHKey bool
	live      bool
	time      bool
	port      int
	timeout   int
}

func NewFindCommand() *cobra.Command {
	args := FindArgs{}
	var findCmd = &cobra.Command{
		Use:   "find",
		Short: "Find your raspberry pi in your local network",
		Long:  `Find your raspberry pi in your local network using SSH.`,
		RunE: func(cmd *cobra.Command, posArgs []string) error {
			if err := findHost(args); err != nil {
				return err
			}
			return nil
		},
	}
	findCmd.Flags().StringVar(&args.subnet, "subnet", "", "Subnet to find the raspberry")
	findCmd.Flags().StringVar(&args.user, "user", "pi", "User to login via ssh")
	findCmd.Flags().StringVar(&args.password, "password", "raspberry", "Password to login via ssh")
	findCmd.Flags().BoolVar(&args.useSSHKey, "ssh-key", false, "Use SSH key")
	findCmd.Flags().IntVar(&args.port, "port", 22, "Port to connect via ssh")
	findCmd.Flags().BoolVar(&args.live, "live", false, "Print valid hosts right after found")
	findCmd.Flags().BoolVar(&args.time, "time", false, "Show hosts processing time")
	findCmd.Flags().IntVar(&args.timeout, "timeout", 1, "Timeout in ns to wait in ssh connections")
	return findCmd
}

func findHost(args FindArgs) error {
	if !args.useSSHKey && len(args.password) == 0 {
		return errors.New("must pass --ssh-key or --password")
	}
	CIDR := args.subnet
	if CIDR == "" {
		defaultCDIR, err := getDefaultCDIR()
		if err != nil {
			return err
		}
		CIDR = defaultCDIR
	}

	fmt.Printf("Getting IP addresses from CIDR %v...\n", CIDR)
	ipv4List, err := getIpsFromCIDR(CIDR)
	if err != nil {
		return err
	}
	fmt.Printf("Found %d IP addresses\n", len(ipv4List))

	fmt.Printf("Validating IP addresses (user: %s)...\n", args.user)
	start := time.Now()
	finder := Finder{totalIPs: ipv4List, findArgs: args}
	validIPs := finder.findValidSSHHosts()
	if args.time {
		elapsed := time.Since(start)
		fmt.Printf("Done (%s)\n", elapsed)
	} else {
		fmt.Println("Done")
	}

	fmt.Printf("Valid ips: %v\n", validIPs)
	return nil
}

type Finder struct {
	mu       sync.Mutex
	wg       sync.WaitGroup
	totalIPs []net.IP
	validIPs []net.IP
	findArgs FindArgs
}

func (f *Finder) findValidSSHHosts() []net.IP {
	for _, ip := range f.totalIPs {
		f.wg.Add(1)
		go f.checkSSHConnection(ip)
	}
	f.wg.Wait()
	return f.validIPs
}

func (f *Finder) checkSSHConnection(ipv4Addr net.IP) {
	defer f.wg.Done()
	connection := ssh.SSHConnection{
		Password:  f.findArgs.password,
		UseSSHKey: f.findArgs.useSSHKey,
		Debug:     false,
		Timeout:   1,
	}
	addr := fmt.Sprintf("%v:%d", ipv4Addr, f.findArgs.port)
	err := connection.Connect(f.findArgs.user, addr)
	if err == nil {
		f.mu.Lock()
		f.validIPs = append(f.validIPs, ipv4Addr)
		if f.findArgs.live {
			fmt.Printf("Found valid host: %v\n", ipv4Addr)
		}
		f.mu.Unlock()
	}
}
