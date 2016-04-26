package main

import (
	"bufio"
	"fmt"
	"os/exec"

	"github.com/digitalocean/godo"
)

// Machine is just a wrapper around a created droplet.
type Machine struct {
	ID        int
	Name      string
	IPv4      string
	SSHActive bool
	Stderr    *bufio.Reader
	Listener  string
	CMD       *exec.Cmd
}

// IsReady ensures that the machine has an IP address.
func (m *Machine) IsReady() bool {
	return m.IPv4 != ""
}

// GetIPs populates the IPv4 address of the machine.
func (m *Machine) GetIPs(client *godo.Client) error {
	droplet, _, err := client.Droplets.Get(m.ID)
	if err != nil {
		return err
	}
	m.IPv4, err = droplet.PublicIPv4()
	if err != nil {
		return err
	}
	return nil
}

// StartSSHProxy starts a socks proxy on 127.0.0.1 or the desired port.
func (m *Machine) StartSSHProxy(port, sshKeyLocation string) error {
	m.Listener = port
	m.CMD = exec.Command("ssh", "-N", "-D", port, "-o", "StrictHostKeyChecking=no", "-i", sshKeyLocation, fmt.Sprintf("root@%s", m.GetIP()))
	stderr, err := m.CMD.StderrPipe()
	if err != nil {
		return err
	}
	m.Stderr = bufio.NewReader(stderr)
	if err := m.CMD.Start(); err != nil {
		return err
	}
	m.SSHActive = true
	return nil
}

// Destroy deletes the droplet.
func (m *Machine) Destroy(client *godo.Client) error {
	_, err := client.Droplets.Delete(m.ID)
	return err
}

// GetIP returns the IPv4 address.
func (m *Machine) GetIP() string {
	return m.IPv4
}

func dropletsToMachines(droplets []godo.Droplet) []Machine {
	m := []Machine{}
	for _, d := range droplets {
		m = append(m, Machine{
			ID:   d.ID,
			Name: d.Name,
		})
	}
	return m
}

// PrintStdError reads from stderr from ssh and prints it to stdout.
func (m *Machine) PrintStdError() {
	for {
		str, err := m.Stderr.ReadString('\n')
		if err != nil && str != "" {
			fmt.Printf("From %s SSH stderr\n", m.Name)
			fmt.Println(str)
		}
	}

}

func printProxyChains(machines []Machine) {
	for _, m := range machines {
		fmt.Printf("socks5 127.0.0.1 %s\n", m.Listener)
	}
}

func printSocksd(machines []Machine) {
	fmt.Printf("\"upstreams\": [\n")
	for i, m := range machines {
		fmt.Printf("{\"type\": \"socks5\", \"address\": \"127.0.0.1:%s\"}", m.Listener)
		if i < len(machines)-1 {
			fmt.Printf(",\n")
		}
	}
	fmt.Printf("\n]\n")
}
