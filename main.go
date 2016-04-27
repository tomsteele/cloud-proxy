/*cloud-proxy is a utility for creating multiple DO droplets
and starting socks proxies via SSH after creation.*/
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"time"
)

var (
	token       = flag.String("token", "", "DO API key")
	sshLocation = flag.String("key-location", "~/.ssh/id_rsa", "SSH key location")
	keyID       = flag.String("key", "", "SSH key fingerprint")
	count       = flag.Int("count", 5, "Amount of droplets to deploy")
	name        = flag.String("name", "cloud-proxy", "Droplet name prefix")
	region      = flag.String("region", "nyc3", "Region to deploy droplets to")
	force       = flag.Bool("force", false, "Bypass built-in protections that prevent you from deploying more than 50 droplets")
	startPort   = flag.Int("start-tcp", 55555, "TCP port to start first proxy on and increment from")
	showversion = flag.Bool("v", false, "Print version and exit")
	version     = "1.1.0"
)

func main() {
	flag.Parse()
	if *showversion {
		fmt.Println(version)
		os.Exit(0)
	}
	if *token == "" {
		log.Fatalln("-token required")
	}

	if *keyID == "" {
		log.Fatalln("-key required")
	}
	if *count > 50 && !*force {
		log.Fatalln("-count greater than 50")
	}

	client := newDOClient(*token)
	droplets, _, err := client.Droplets.CreateMultiple(newDropLetMultiCreateReqeust(*name, *region, *keyID, *count))
	if err != nil {
		log.Fatalf("There was an error creating the droplets:\nError: %s\n", err.Error())
	}
	log.Println("Droplets deployed. Waiting 100 seconds...")
	time.Sleep(100 * time.Second)

	// For each droplet, poll it once, start SSH proxy, and then track it.
	machines := dropletsToMachines(droplets)
	for i := range machines {
		m := &machines[i]
		if err := m.GetIPs(client); err != nil {
			log.Println("There was an error getting the IPv4 address of droplet name: %s\nError: %s\n", m.Name, err.Error())
		}
		if m.IsReady() {
			if err := m.StartSSHProxy(strconv.Itoa(*startPort), *sshLocation); err != nil {
				log.Println("Could not start SSH proxy on droplet name: %s\nError: %s\n", m.Name, err.Error())
			} else {
				log.Println("SSH proxy started on port %d on droplet name: %s IP: %s\n", *startPort, m.Name, m.IPv4)
				go m.PrintStdError()
			}
			*startPort++
		} else {
			log.Println("Droplet name: %s is not ready yet. Skipping...\n", m.Name)
		}
	}

	log.Println("proxychains config")
	printProxyChains(machines)
	log.Println("socksd config")
	printSocksd(machines)

	log.Println("Please CTRL-C to destroy droplets")

	// Catch CTRL-C and delete droplets.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	for _, m := range machines {
		if err := m.Destroy(client); err != nil {
			log.Println("Could not delete droplet name: %s\n", m.Name)
		} else {
			log.Println("Deleted droplet name: %s", m.Name)
		}
	}
}
