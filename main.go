/*cloud-proxy is a utility for creating multiple instances
and starting socks proxies via SSH after creation.*/
package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"text/template"
	"time"

	"golang.org/x/crypto/sha3"
)

var (
	sshLocation   = flag.String("key-location", "~/.ssh/id_rsa", "SSH key location")
	count         = flag.Int("count", 5, "Amount of droplets to deploy")
	name          = flag.String("name", "cloud-proxy", "Droplet name prefix")
	doRegionFlag  = flag.String("doRegions", "*", "Comma separated list of regions to deploy droplets to, defaults to all.")
	awsRegionFlag = flag.String("awsRegions", "*", "Comma separated list of regions to deploy droplets to, defaults to all.")
	force         = flag.Bool("force", false, "Bypass built-in protections that prevent you from deploying more than 50 droplets")
	startPort     = flag.Int("start-tcp", 55555, "TCP port to start first proxy on and increment from")
	awsProvider   = flag.Bool("aws", false, "Use AWS as provider")
	doProvider    = flag.Bool("do", false, "Use DigitalOcean as provider")
	showversion   = flag.Bool("v", false, "Print version and exit")
	version       = "2.0.0"
)

type digitaloceanInfo struct {
	Name   string
	Region string
}

type AwsInfo struct {
	AMI           map[string]string // AMI["sa-east-1"] = "cloud-proxy-ami-sa-east-1"
	SecurityGroup map[string]string // SecurityGroup["sa-east-1"] = "cloud-proxy-securitygroup-sa-east-1"
	Instances     []Instance
}

type Instance struct {
	Name          string
	Region        string
	AMI           string
	SecurityGroup string
}

func main() {
	flag.Parse()
	if *showversion {
		fmt.Println(version)
		os.Exit(0)
	}

	if !*doProvider && !*awsProvider {
		log.Fatal("Need at least one provider")
	}

	providers := []string{}
	doInfos := []digitaloceanInfo{}
	awsInfo := AwsInfo{}

	if *doProvider {
		providers = append(providers, "digitalocean")
	}
	if *awsProvider {
		providers = append(providers, "aws")
		awsInfo.AMI = make(map[string]string)
		awsInfo.SecurityGroup = make(map[string]string)
	}

	if *count > 50 && !*force {
		log.Fatalln("-count greater than 50")
	}

	for x := 0; x < *count; x++ {
		var region string

		providerIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(providers))))
		if err != nil {
			log.Fatal(err)
		}

		switch providers[providerIndex.Int64()] {
		case "digitalocean":
			if *doRegionFlag == "*" {
				region = randomRegion(doRegions)
			} else {
				numRegions := strings.Split(*doRegionFlag, ",")
				if len(numRegions) > 1 {
					region = randomRegion(numRegions)
				} else {
					region = numRegions[0]
				}
			}
			computerName := fmt.Sprintf("%s-%s", *name, namePostfix())
			cs := digitaloceanInfo{}
			cs.Name = computerName
			cs.Region = region
			doInfos = append(doInfos, cs)
		case "aws":
			if *awsRegionFlag == "*" {
				region = randomRegion(awsRegions)
			} else {
				numRegions := strings.Split(*awsRegionFlag, ",")
				if len(numRegions) > 1 {
					region = randomRegion(numRegions)
				} else {
					region = numRegions[0]
				}
			}
			computerName := fmt.Sprintf("%s-%s-%s", *name, namePostfix(), region)
			amiName := fmt.Sprintf("%s-ami-%s", *name, region)
			securityGroupName := fmt.Sprintf("%s-securitygroup-%s", *name, region)
			awsInfo.AMI[region] = amiName
			awsInfo.SecurityGroup[region] = securityGroupName
			awsInfo.Instances = append(awsInfo.Instances, Instance{Name: computerName, Region: region, AMI: amiName, SecurityGroup: securityGroupName})
		}

	}

	createTerraformFile("do-infrastructure.tf", doTemplate, doInfos)
	createTerraformFile("aws-infrastructure.tf", awsTemplate, awsInfo)

	executeTerraform([]string{"apply", "-var-file=secrets.tfvars", "-auto-approve"})

	computerUsers := make(map[string]string)
	for _, instance := range awsInfo.Instances {
		name := fmt.Sprintf("%s-IP", instance.Name)
		computerUsers[name] = "ec2-user"
	}
	for _, computer := range doInfos {
		name := fmt.Sprintf("%s-IP", computer.Name)
		computerUsers[name] = "root"
	}

	printConfigs(*startPort, printProxyChains, printSocksd)

	tunnelProcesses := createTunnels(computerUsers)

	log.Println("Please CTRL-C to destroy infrastructure")

	// Catch CTRL-C and delete droplets.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	executeTerraform([]string{"destroy", "-var-file=secrets.tfvars", "-auto-approve"})
	for _, process := range tunnelProcesses {
		if err := process.Kill(); err != nil {
			log.Fatal(err)
		}
	}

}

func printConfigs(port int, configs ...func(int)) {
	for _, config := range configs {
		config(port)
	}
}

func printProxyChains(port int) {
	log.Println("proxychains config")
	for x := 0; x < *count; x++ {
		fmt.Printf("socks5 127.0.0.1 %d\n", port)
		port++
	}
}

func printSocksd(port int) {
	log.Println("socksd config")
	fmt.Printf("\"upstreams\": [\n")
	for x := 0; x < *count; x++ {
		fmt.Printf("{\"type\": \"socks5\", \"address\": \"127.0.0.1:%d\"}", port)
		port++
		if x < (*count - 1) {
			fmt.Printf(",\n")
		}
	}
	fmt.Printf("\n]\n")
}

func randomRegion(regions []string) string {
	regionIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(regions))))
	if err != nil {
		log.Fatal(err)
	}
	return regions[regionIndex.Int64()]
}

func createTerraformFile(fileName, templateData string, data interface{}) {
	fh, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err)
	}
	t := template.Must(template.New("tmpl").Parse(templateData))
	if err := t.Execute(fh, data); err != nil {
		log.Fatal(err)
	}
}

func createTunnels(computerUsers map[string]string) []*os.Process {
	time.Sleep(10 * time.Second)
	output, err := exec.Command("terraform", "output").Output()
	if err != nil {
		log.Fatal(err)
	}

	tunnelProcesses := []*os.Process{}

	for _, line := range strings.Split(string(output), "\n") {
		splitOutput := strings.Split(line, "=")
		if len(splitOutput) != 2 {
			continue
		}
		ip := strings.TrimSpace(splitOutput[1])
		ip2 := strings.Trim(ip, "\"")
		computerName := strings.TrimSpace(splitOutput[0])
		port := fmt.Sprintf("%d", *startPort)
		var host string
		if user, ok := computerUsers[computerName]; ok {
			host = fmt.Sprintf("%s@%s", user, ip2)
		}
		fmt.Printf("creating tunnel to %s on %s\n", host, port)
		cmd := exec.Command("ssh", "-D", port, "-N", "-o", "StrictHostKeyChecking=no", "-i", *sshLocation, host)
		cmd.Stderr = os.Stderr
		cmd.Start()
		tunnelProcesses = append(tunnelProcesses, cmd.Process)
		*startPort++
	}
	return tunnelProcesses
}

func executeTerraform(args []string) {
	cmd := exec.Command("terraform", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Start()
	cmd.Wait()
}

func namePostfix() string {
	var buffer [32]byte
	if _, err := rand.Read(buffer[:]); err != nil {
		log.Fatal(err)
	}
	hash := sha3.New256()
	if _, err := hash.Write(buffer[:]); err != nil {
		log.Fatal(err)
	}
	output := hash.Sum(nil)
	return hex.EncodeToString(output[0:8])
}
