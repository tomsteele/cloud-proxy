# cloud-proxy
cloud-proxy creates multiple DO droplets and then starts local socks proxies using SSH. After exiting, the droplets are deleted.

### Warning
This tool will deploy as many droplets as you desire, and will make a best effort to delete them after use. However, you are ultimately going to pay the bill for these droplets, and it is up to you, and you alone to ensure they actually get deleted.

### Install
Download a compiled release [here](https://github.com/tomsteele/cloud-proxy/releases/latest). You can now execute without any dependencies. Currently the only supported and tested OS is Linux:
```
$ ./cloud-proxy
```
### Usage
```
Usage of cloud-proxy:
  -count int
    	Amount of droplets to deploy (default 5)
  -force
    	Bypass built-in protections that prevent you from deploying more than 50 droplets
  -key string
    	SSH key fingerprint
  -name string
    	Droplet name prefix (default "cloud-proxy")
  -region string
    	Region to deploy droplets to (default "nyc3")
  -start-tcp int
    	TCP port to start first proxy on and increment from (default 55555)
  -token string
    	DO API key
  -v	Print version and exit
```

### Getting Started
To use cloud-proxy you will need to have a DO API token, you can get one [here](https://cloud.digitalocean.com/settings/api/tokens). Next, ensure you have an SSH key saved on DO. This is the key that SSH will authentication with. The DO API and cloud-proxy require you to provide the fingerprint of the key you would like to use. You can obtain the fingerprint using `ssh-keygen`:
```
$ ssh-keygen -lf ~/.ssh/id_rsa.pub
```

If your key requires a passphrase, you will need to use ssh-agent:
```
$ eval `ssh-agent -s`
$ ssh-add ~/.ssh/id_rsa
```

Now you may create some proxies:
```
$ cloud-proxy -count 2 -token <api-token> -key <fingerprint>
```

When you are finished using your proxies, use CTRL-C to interrupt the program, cloud-proxy will catch the interrupt and delete the droplets.

cloud-proxy will output a proxy list for proxychains and [socksd](https://github.com/eahydra/socks/tree/master/cmd/socksd). proxychains can be configured to iterate over a random proxy for each connection by uncommenting `random_chain`, you should also comment out `string-chain`, which is the default. You will also need to uncomment `chain_len` and set it to `1`.

socksd can be helpful for programs that can accept a socks proxy, but may not work nicely with proxychains. socksd will listen as a socks proxy, and can be configured to use a set of upstream proxies, which it will iterate through in a round-robin manner. Follow the instructions in the README linked above, as it is self explanitory.

### Example Output
```
$ ./cloud-proxy -token <my_token> -key <my_fingerprint>
==> Info: Droplets deployed. Waiting 100 seconds...
==> Info: SSH proxy started on port 55555 on droplet name cloud-proxy-1
==> Info: SSH proxy started on port 55556 on droplet name cloud-proxy-2
==> Info: SSH proxy started on port 55557 on droplet name cloud-proxy-3
==> Info: SSH proxy started on port 55558 on droplet name cloud-proxy-4
==> Info: SSH proxy started on port 55559 on droplet name cloud-proxy-5
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
^C==> Info: Deleted droplet name: cloud-proxy-1
==> Info: Deleted droplet name: cloud-proxy-2
==> Info: Deleted droplet name: cloud-proxy-3
==> Info: Deleted droplet name: cloud-proxy-4
==> Info: Deleted droplet name: cloud-proxy-5
```

