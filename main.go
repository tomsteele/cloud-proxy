/*cloud-proxy is a utility for creating multiple DO droplets
and starting socks proxies via SSH after creation.*/
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/digitalocean/godo"
)

var (
	token       = flag.String("token", "", "DO API key")
	sshLocation = flag.String("key-location", "~/.ssh/id_rsa", "SSH key location")
	keyID       = flag.String("key", "", "SSH key fingerprint")
	count       = flag.Int("count", 5, "Amount of droplets to deploy")
	name        = flag.String("name", "cloud-proxy", "Droplet name prefix")
	regions     = flag.String("regions", "*", "Comma separated list of regions to deploy droplets to, defaults to all.")
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

	availableRegions, err := doRegions(client)
	if err != nil {
		log.Fatalf("There was an error getting a list of regions:\nError: %s\n", err.Error())
	}

	regionCountMap, err := regionMap(availableRegions, *regions, *count)
	if err != nil {
		log.Fatalf("%s\n", err.Error())
	}
	droplets := []godo.Droplet{}

	for region, c := range regionCountMap {
		log.Printf("Creating %d droplets to region %s", c, region)
		drops, _, err := client.Droplets.CreateMultiple(newDropLetMultiCreateRequest(*name, region, *keyID, c))
		if err != nil {
			log.Printf("There was an error creating the droplets:\nError: %s\n", err.Error())
			log.Fatalln("You may need to do some manual clean up!")
		}
		droplets = append(droplets, drops...)
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

func regionMap(slugs []string, regions string, count int) (map[string]int, error) {
	allowedSlugs := strings.Split(regions, ",")
	regionCountMap := make(map[string]int)

	if regions != "*" {
		for _, s := range slugs {
			for _, a := range allowedSlugs {
				if s == a {
					if len(regionCountMap) == count {
						break
					}
					regionCountMap[s] = 0
				}
			}
		}
	} else {
		for _, s := range slugs {
			if len(regionCountMap) == count {
				break
			}
			regionCountMap[s] = 0
		}
	}

	if len(regionCountMap) == 0 {
		return regionCountMap, errors.New("There are no regions to use")
	}

	perRegionCount := count / len(regionCountMap)
	perRegionCountRemainder := count % len(regionCountMap)

	for k := range regionCountMap {
		regionCountMap[k] = perRegionCount
	}

	if perRegionCountRemainder != 0 {
		c := 0
		for k, v := range regionCountMap {
			if c >= perRegionCountRemainder {
				break
			}
			regionCountMap[k] = v + 1
			c++
		}
	}
	return regionCountMap, nil
}
