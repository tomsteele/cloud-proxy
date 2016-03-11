/*cloud-proxy is a utility for creating multiple DO droplets
and starting socks proxies via SSH after creation.*/
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/iamthemuffinman/logsip"
)

var (
	token     = flag.String("token", "", "DO API key")
	keyID     = flag.String("key", "", "SSH key fingerprint")
	count     = flag.Int("count", 5, "Amount of droplets to deploy")
	name      = flag.String("name", "cloud-proxy", "Droplet name prefix")
	region    = flag.String("region", "nyc3", "Region to deploy droplets to")
	force     = flag.Bool("force", false, "Bypass built-in protections that prevent you from deploying more than 50 droplets")
	startPort = flag.Int("start-tcp", 55555, "TCP port to start first proxy on and increment from")
	version   = flag.Bool("v", false, "Print version and exit")
	VERSION   = "1.0.0"
)

func main() {
	flag.Parse()
	if *version {
		fmt.Println(VERSION)
		os.Exit(0)
	}
	log := logsip.Default()

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
	log.Infoln("Droplets deployed. Waiting 100 seconds...")
	time.Sleep(100 * time.Second)

	// For each droplet, poll it once, start SSH proxy, and then track it.
	machines := dropletsToMachines(droplets)
	for i := range machines {
		m := &machines[i]
		if err := m.GetIPs(client); err != nil {
			log.Warnf("There was an error getting the IPv4 address of droplet name: %s\nError: %s\n", m.Name, err.Error())
		}
		if m.IsReady() {
			if err := m.StartSSHProxy(strconv.Itoa(*startPort)); err != nil {
				log.Warnf("Could not start SSH proxy on droplet name: %s\nError: %s\n", m.Name, err.Error())
			} else {
				log.Infof("SSH proxy started on port %d on droplet name %s\n", *startPort, m.Name)
				go m.PrintStdError()
			}
			*startPort++
		} else {
			log.Warnf("Droplet name: %s is not ready yet. Skipping...\n", m.Name)
		}
	}
	fmt.Println(machines)

	log.Infoln("proxychains config")
	printProxyChains(machines)
	log.Infoln("socksd config")
	printSocksd(machines)

	// Catch CTRL-C and delete droplets.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	for _, m := range machines {
		if err := m.Destroy(client); err != nil {
			log.Warnf("Could not delete droplet name: %s\n", m.Name)
		} else {
			log.Infof("Deleted droplet name: %s", m.Name)
		}
	}
}
