package main

import (
	"11800222/killswitch"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
	//"github.com/vpn-kill-switch/killswitch"
)

// PadRight add spaces for aligning the output
func PadRight(str, pad string, length int) string {
	for {
		str += pad
		if len(str) > length {
			return str[0:length]
		}
	}
}

func exit1(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

var version string

func main() {
	var (
		ip    = flag.String("ip", "", "VPN peer `IPv4`, killswitch tries to find this automatically")
		d     = flag.Bool("d", false, "`Disable` load /etc/pf.conf rules")
		e     = flag.Bool("e", false, "`Enable` load the pf rules")
		p     = flag.Bool("p", false, "`Print` the pf rules")
		v     = flag.Bool("v", false, fmt.Sprintf("Print version: %s", version))
		w     = flag.Bool("w", false, "just add vpn pass to existed pf rules, when its interface is detected")
		leak  = flag.Bool("leak", false, "Allow ICMP (ping) and DNS requests outside VPN")
		local = flag.Bool("local", false, "Allow local network traffic")
	)

	flag.Parse()

	if *v {
		fmt.Printf("%s\n", version)
		os.Exit(0)
	}

	if *d {
		exec.Command("pfctl", "-e").CombinedOutput()
		_, err := exec.Command("pfctl",
			"-Fa",
			"-f",
			"/etc/pf.conf").CombinedOutput()
		if err != nil {
			exit1(fmt.Errorf(fmt.Sprintf("%s: %s\n%s",
				killswitch.Red("To disable use"),
				killswitch.Yellow("sudo killswitch -d"),
				err)),
			)
		}
		out, err := exec.Command("pfctl", "-sr").CombinedOutput()
		if err != nil {
			exit1(err)
		}
		fmt.Printf("%s\n", out)
		fmt.Println(killswitch.Yellow("killswitch disabled"))
		return
	}

	ks, err := killswitch.New(*ip)
	if err != nil {
		exit1(err)
	}

	err = ks.GetActive()
	if err != nil {
		exit1(err)
	}

	if *w {
		counter := 0
		for {
			counter++
			fmt.Println("waiting for active interfaces Counter:", counter)
			fmt.Println("current active interfaces:", ks.UpInterfaces, ks.P2PInterfaces)
			time.Sleep(1 * time.Second)

			ks, err = killswitch.New(*ip)
			if err != nil {
				exit1(err)
			}

			ks.Mu.Lock()
			activeErr := ks.GetActive()
			ks.Mu.Unlock()

			if activeErr != nil {
				fmt.Printf("active_err Error: %v\n", activeErr)
				exit1(err)
			}

			if len(ks.P2PInterfaces) > 0 && len(ks.UpInterfaces) > 0 {
				var ruleContent string
				// local network
				for k2 := range ks.UpInterfaces {
					ruleContent += fmt.Sprintf("int_%s = %q\n", k2, k2)
					ruleContent += fmt.Sprintf("pass from $int_%s:network to $int_%s:network\n", k2, k2)
				}
				// vpn
				for k := range ks.P2PInterfaces {
					ruleContent += fmt.Sprintf("pass on %s all\n", k)
				}

				err = os.WriteFile("/tmp/pf_rule_pass_for_vpn.conf", []byte(ruleContent), 0644)

				output, err := exec.Command("pfctl",
					"-a", "vpn_in_use",
					"-e", // important
					"-f",
					"/tmp/pf_rule_pass_for_vpn.conf").CombinedOutput()

				if err != nil {
					fmt.Printf("Error: %v\n", err)
				}
				fmt.Printf("Output: %s\n", output)
				break
			}
		}
		if err != nil {
			exit1(err)
		}
	}

	fmt.Println("Interface  MAC address         IP")
	for k, v := range ks.UpInterfaces {
		fmt.Printf("%s %s   %s\n", PadRight(k, " ", 10), v[0], v[1])
	}
	for k, v := range ks.P2PInterfaces {
		fmt.Printf("%s %s   %s\n", PadRight(k, " ", 10), PadRight(v[0], " ", 17), v[1])
	}
	// check for DNS leaks
	if ipDNS, err := killswitch.WhoamiDNS(); err == nil {
		if ipWWW, err := killswitch.WhoamiWWW(); err == nil {
			if ipDNS != ipWWW {
				fmt.Printf("\n%s:\n", killswitch.Red("DNS leaking"))
				fmt.Printf("Public IP address (DNS): %s\n", killswitch.Red(ipDNS))
				fmt.Printf("Public IP address (WWW): %s\n", killswitch.Red(ipWWW))
			} else {
				fmt.Printf("\nPublic IP address: %s\n", killswitch.Red(ipDNS))
			}
		}
	}

	if len(ks.P2PInterfaces) == 0 {
		// should not happen when -w is used
		exit1(fmt.Errorf(fmt.Sprintf("\n%s",
			killswitch.Red("No VPN interface found, verify VPN is connected")),
		))
	}

	fmt.Printf("PEER IP address:   %s\n", killswitch.Yellow(ks.PeerIP))

	if *ip != "" {
		if ipv4 := net.ParseIP(*ip); ipv4.To4() == nil {
			exit1(fmt.Errorf("%s is not a valid IPv4 address, use (\"%s -h\") for help.\n", *ip, os.Args[0]))
		}
	}

	ks.CreatePF(*leak, *local)

	if !*e {
		fmt.Printf("\n%s: %s\n", "To enable the kill switch run", killswitch.Green("sudo killswitch -e"))
		fmt.Printf("%s: %s\n", "To disable", killswitch.Yellow("sudo killswitch -d"))
	}

	if *p {
		fmt.Printf("PF rules to be loaded:\n")
		fmt.Println(ks.PFRules.String())
	}

	if err = ioutil.WriteFile("/tmp/killswitch.pf.conf",
		ks.PFRules.Bytes(),
		0644,
	); err != nil {
		exit1(err)
	}

	if *e {
		exec.Command("pfctl", "-e").CombinedOutput()
		_, err := exec.Command("pfctl",
			"-Fa",
			"-f",
			"/tmp/killswitch.pf.conf").CombinedOutput()
		if err != nil {
			exit1(fmt.Errorf(fmt.Sprintf("\n%s: %s",
				killswitch.Red("Kill switch is not enable, to enable use"),
				killswitch.Green("sudo killswitch -e"))),
			)
		}
		fmt.Printf("\n# %s\n", strings.Repeat("-", 62))
		fmt.Println("# Loading rules")
		fmt.Printf("# %s\n", strings.Repeat("-", 62))
		out, err := exec.Command("pfctl", "-sr").CombinedOutput()
		if err != nil {
			exit1(err)
		}
		fmt.Printf("%s\n", out)
		fmt.Println(killswitch.Green("killswitch enabled"))
	}
}
