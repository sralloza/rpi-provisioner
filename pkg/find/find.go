package find

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/sralloza/rpi-provisioner/pkg/ssh"
)

type Args struct {
	Subnet    string
	User      string
	Password  string
	UseSSHKey bool
	Port      int
}

func FindHost(args Args) error {
	if !args.UseSSHKey && len(args.Password) == 0 {
		return errors.New("must pass --ssh-key or --password")
	}
	CIDR := args.Subnet
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

	fmt.Printf("Zear IP addresses (user: %s)...\n", args.User)
	start := time.Now()
	finder := Finder{totalIPs: ipv4List, findArgs: args}
	validIPs := finder.findValidSSHHosts()

	elapsed := time.Since(start)
	fmt.Printf("Done (%s): %d valid hosts out of %d\n", elapsed, len(validIPs), len(ipv4List))
	return nil
}

type Finder struct {
	mu       sync.Mutex
	wg       sync.WaitGroup
	totalIPs []net.IP
	validIPs []net.IP
	findArgs Args
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
		Password:  f.findArgs.Password,
		UseSSHKey: f.findArgs.UseSSHKey,
		Timeout:   1,
	}
	addr := fmt.Sprintf("%v:%d", ipv4Addr, f.findArgs.Port)
	err := connection.Connect(f.findArgs.User, addr)
	if err == nil {
		f.mu.Lock()
		f.validIPs = append(f.validIPs, ipv4Addr)
		fmt.Printf("Found valid host: %v\n", ipv4Addr)
		f.mu.Unlock()
	}
}
