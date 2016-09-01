package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/digitalocean/godo"
)

func doRegions(client *godo.Client) ([]string, error) {
	slugs := []string{}
	regions, _, err := client.Regions.List(&godo.ListOptions{})
	if err != nil {
		return slugs, err
	}
	for _, r := range regions {
		slugs = append(slugs, r.Slug)
	}
	return slugs, nil
}

func newDropLetMultiCreateRequest(prefix, region, keyID string, count int) *godo.DropletMultiCreateRequest {

	names := []string{}
	for i := 0; i < count; i++ {
		b := make([]byte, 6)
		rand.Read(b)
		names = append(names, fmt.Sprintf("%s-%s", prefix, base64.StdEncoding.EncodeToString(b)))
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
