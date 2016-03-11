package main

import (
	"fmt"
	"strconv"

	"github.com/digitalocean/godo"
)

func newDropLetMultiCreateReqeust(prefix, region, keyID string, count int) *godo.DropletMultiCreateRequest {

	names := []string{}
	// Start index at 1 so we can use it in the hostname.
	for i := 1; i <= count; i++ {
		names = append(names, fmt.Sprintf("%s-%s", prefix, strconv.Itoa(i)))
	}

	return &godo.DropletMultiCreateRequest{
		Names:  names,
		Region: region,
		Size:   "512mb",
		Image: godo.DropletCreateImage{
			Slug: "ubuntu-14-04-x64",
		},
		SSHKeys: []godo.DropletCreateSSHKey{
			godo.DropletCreateSSHKey{
				Fingerprint: keyID,
			},
		},
		Backups:           false,
		IPv6:              false,
		PrivateNetworking: false,
	}
}
