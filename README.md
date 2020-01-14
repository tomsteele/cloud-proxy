# cloud-proxy
cloud-proxy creates multiple cloud instances and then starts local socks proxies using SSH. After exiting, the droplets are deleted.

### Warning
This tool will deploy as many droplets as you desire, and will make a best effort to delete them after use. However, you are ultimately going to pay the bill for these droplets, and it is up to you, and you alone to ensure they actually get deleted.

### Install
Download a compiled release [here](https://github.com/tomsteele/cloud-proxy/releases/latest).  
Download terraform [here](https://www.terraform.io/downloads.html).  
Currently the only supported and tested OS is Linux:
```
$ ./cloud-proxy
```
### Usage
```
Usage of ./cloud-proxy:
  -aws
    	Use AWS as provider
  -awsRegions string
    	Comma separated list of regions to deploy droplets to, defaults to all. (default "*")
  -count int
    	Amount of droplets to deploy (default 5)
  -do
    	Use DigitalOcean as provider
  -doRegions string
    	Comma separated list of regions to deploy droplets to, defaults to all. (default "*")
  -force
    	Bypass built-in protections that prevent you from deploying more than 50 droplets
  -key-location string
    	SSH key location (default "~/.ssh/id_rsa")
  -name string
    	Droplet name prefix (default "cloud-proxy")
  -start-tcp int
    	TCP port to start first proxy on and increment from (default 55555)
  -v	Print version and exit
```

### Getting Started
To use cloud-proxy with DO you will need to have a DO API token, you can get one [here](https://cloud.digitalocean.com/settings/api/tokens).
To use cloud-proxy with AWS you will need to have an Access and Secret key.

Next, ensure you have an SSH key saved on DO. On AWS your SSH key will need to be setup on each region you'd like to use. This is the key that SSH will authentication with. The DO API and cloud-proxy require you to provide the fingerprint of the key you would like to use. You can obtain the fingerprint using `ssh-keygen`. Please note that DO requires fingerprint keys in the MD5 format:
```
$ ssh-keygen -lf ~/.ssh/id_rsa.pub -E MD5
```

If your key requires a passphrase, you will need to use ssh-agent:
```
$ eval `ssh-agent -s`
$ ssh-add ~/.ssh/id_rsa
```

The AWS API only requires the name of the SSH key you previously imported into each region.

Finally, you'll need to create a `secrets.tfvars` file that will contain the API tokens and information for SSH. The file should look like the following:
```
do_token = "YOUR_DO_TOKEN"
do_ssh_fingerprint = "YOUR:SSH:FINGERPRINT"
aws_access_key = "YOUR_ACCESS_KEY"
aws_secret_key = "YOUR_SECRET_KEY"
aws_key_name = "SSH_KEY_NAME"
```

Now you may create some proxies:
```
$ ./cloud-proxy -do -aws -count 15
```

When you are finished using your proxies, use CTRL-C to interrupt the program, cloud-proxy will catch the interrupt and delete the instances.

cloud-proxy will output a proxy list for proxychains and [socksd](https://github.com/eahydra/socks/tree/master/cmd/socksd). proxychains can be configured to iterate over a random proxy for each connection by uncommenting `random_chain`, you should also comment out `string-chain`, which is the default. You will also need to uncomment `chain_len` and set it to `1`.

socksd can be helpful for programs that can accept a socks proxy, but may not work nicely with proxychains. socksd will listen as a socks proxy, and can be configured to use a set of upstream proxies, which it will iterate through in a round-robin manner. Follow the instructions in the README linked above, as it is self explanitory.

### Example Output
```
$ ./cloud-proxy -token <my_token> -key <my_fingerprint>
==> Info: Droplets deployed. Waiting 100 seconds...
==> Info: SSH proxy started on port 55555 on droplet name: cloud-proxy-1 IP: <IP>
==> Info: SSH proxy started on port 55556 on droplet name: cloud-proxy-2 IP: <IP>
==> Info: SSH proxy started on port 55557 on droplet name: cloud-proxy-3 IP: <IP>
==> Info: SSH proxy started on port 55558 on droplet name: cloud-proxy-4 IP: <IP>
==> Info: SSH proxy started on port 55559 on droplet name: cloud-proxy-5 IP: <IP>
==> Info: proxychains config
socks5 127.0.0.1 55555
socks5 127.0.0.1 55556
socks5 127.0.0.1 55557
socks5 127.0.0.1 55558
socks5 127.0.0.1 55559
==> Info: socksd config
"upstreams": [
{"type": "socks5", "address": "127.0.0.1:55555"},
{"type": "socks5", "address": "127.0.0.1:55556"},
{"type": "socks5", "address": "127.0.0.1:55557"},
{"type": "socks5", "address": "127.0.0.1:55558"},
{"type": "socks5", "address": "127.0.0.1:55559"}
]
==> Info: Please CTRL-C to destroy droplets
^C==> Info: Deleted droplet name: cloud-proxy-1
==> Info: Deleted droplet name: cloud-proxy-2
==> Info: Deleted droplet name: cloud-proxy-3
==> Info: Deleted droplet name: cloud-proxy-4
==> Info: Deleted droplet name: cloud-proxy-5
```

