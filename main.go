/*cloud-proxy is a utility for creating multiple DO droplets
and starting socks proxies via SSH after creation.*/
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
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
	version     = "1.3.0"
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

	var droplets []godo.Droplet

	for region, c := range regionCountMap {
		log.Printf("Creating %d droplets to region %s", c, region)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		drops, _, err := client.Droplets.CreateMultiple(ctx, newDropLetMultiCreateRequest(*name, region, *keyID, c))
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
		m := machines[i]
		if err := m.GetIPs(client); err != nil {
			log.Printf("There was an error getting the IPv4 address of droplet name: %s\nError: %s\n", m.Name, err.Error())
		}
		if m.IsReady() {
			if err := m.StartSSHProxy(strconv.Itoa(*startPort), *sshLocation); err != nil {
				log.Printf("Could not start SSH proxy on droplet name: %s\nError: %s\n", m.Name, err.Error())
			} else {
				log.Printf("SSH proxy started on port %d on droplet name: %s IP: %s\n", *startPort, m.Name, m.IPv4)
			}
			*startPort++
		} else {
			log.Printf("Droplet name: %s is not ready yet. Skipping...\n", m.Name)
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
	go func() {
		<-c
		cleanUp(machines, client)
		os.Exit(1)
	}()
	var newLine string
	header := strings.Repeat("-", 60)
	if runtime.GOOS == "windows" {
		newLine = "\r\n"
	} else {
		newLine = "\n"
	}
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println(header)
		fmt.Printf("[L]ist [C]onnect [D]isconnect [Q]uit [H]elp: ")
		text, _ := reader.ReadString('\n')
		text = strings.Replace(text, newLine, "", -1)
		lowerText := strings.ToLower(text)
		splitText := strings.Fields(lowerText)
		var id int
		var err error
		switch len(splitText) {
		case 0:
			continue
		case 2, 3:
			id, err = strconv.Atoi(splitText[1])
			if err != nil {
				log.Println(err)
				continue
			}
		}
		switch splitText[0] {
		case "l":
			for _, machine := range machines {
				fmt.Printf("ID: %d\tName: %s\tIP: %s:%s\tConnected: %v\n",
					machine.ID, machine.Name, machine.IPv4, machine.Listener, machine.SSHActive)
			}
		case "c":
			modifyTunnel(machines, id, connect(splitText[2], *sshLocation))
		case "d":
			modifyTunnel(machines, id, disconnect)
		case "h":
			fmt.Println(`
l              List current machines and connections
c [id] [port]  Create a socks proxy using the ID and then port
d [id]         Disconnect socks proxy via the host ID
q              Quit program
h              This message`)
		case "q":
			cleanUp(machines, client)
			os.Exit(0)
		default:
		}
	}
}

func modifyTunnel(machines []*Machine, id int, f func(*Machine)) {
	for _, machine := range machines {
		if machine.ID == id {
			f(machine)
		}
	}
}

func connect(port, sshLocation string) func(*Machine) {
	return func(machine *Machine) {
		if machine.SSHActive {
			fmt.Println("[WARNING] Machine already has an active socks proxy. Please disconnect the tunnel before creating a new one.")
			return
		}
		if err := machine.StartSSHProxy(port, sshLocation); err != nil {
			log.Println(err)
		}
	}
}

func disconnect(machine *Machine) {
	if !machine.SSHActive {
		fmt.Println("[WARNING] Machine does not have an active tunnel")
		return
	}
	deleteTunnel(machine)
}

func deleteTunnel(machine *Machine) {
	if err := machine.CMD.Process.Kill(); err != nil {
		log.Println(err)
	}
	machine.done <- true
	machine.Listener = ""
	machine.SSHActive = false
}

func cleanUp(machines []*Machine, client *godo.Client) {
	fmt.Println("Cleaning up, and exiting")
	for _, m := range machines {
		deleteTunnel(m)
		if err := m.Destroy(client); err != nil {
			log.Printf("Could not delete droplet name: %s\n", m.Name)
		} else {
			log.Printf("Deleted droplet name: %s\n", m.Name)
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
