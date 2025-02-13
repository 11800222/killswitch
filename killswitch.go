package killswitch

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"sync"
)

// Network struct
type Network struct {
	Interfaces    []net.Interface
	UpInterfaces  map[string][]string
	P2PInterfaces map[string][]string
	PeerIP        string
	PFRules       bytes.Buffer
	Mu            sync.Mutex
}

// New returns a Network struct
func New(peerIP string) (*Network, error) {
	var (
		ip  net.IP
		err error
	)
	if peerIP != "" {
		ip = net.ParseIP(peerIP)
	} else {
		if ip, err = UGSX(); err != nil {
			return nil, err
		}
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	return &Network{
		Interfaces:    ifaces,
		UpInterfaces:  make(map[string][]string),
		P2PInterfaces: make(map[string][]string),
		PeerIP:        ip.String(),
	}, nil
}

// GetActive finds active interfaces
func (n *Network) GetActive() error {
	fmt.Println("Interfaces: ", n.Interfaces)
	for _, i := range n.Interfaces {
		if i.Flags&net.FlagUp == 0 {
			fmt.Println("Interface down", i.Name)
			continue // interface down
		}
		if i.Flags&net.FlagLoopback != 0 {
			fmt.Println(" Loopback interface", i.Name)
			continue // loopback interface
		}
		addrs, err := i.Addrs()
		if err != nil {
			return err
		}
		for _, addr := range addrs {
			fmt.Println(" Interface up", i.Name)
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				fmt.Println(" IP is nil or loopback", i.Name)
				continue
			}
			ip = ip.To4()
			if ip == nil {
				fmt.Println(" Not an ipv4 address", i.Name)
				continue // not an ipv4 address
			}
			if i.Flags&net.FlagPointToPoint != 0 {
				n.P2PInterfaces[i.Name] = []string{i.HardwareAddr.String(), ip.String()}
			} else {
				fmt.Println("flag point to point", i.Name)
				// get mask
				mask := ip.DefaultMask()
				prefixSize, _ := mask.Size()
				n.UpInterfaces[i.Name] = []string{i.HardwareAddr.String(), fmt.Sprintf("%s/%d", ip.String(), prefixSize)}
			}
		}
	}
	if n.UpInterfaces == nil {
		return errors.New("No active connections, verify you are connected to the network")
	}
	return nil
}
