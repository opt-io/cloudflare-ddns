# cloudflare-ddns
A command-line cloudflare DDNS updater written in Go, using the CloudFlare GO API library: https://github.com/cloudflare/cloudflare-go

# Usage

You must update and include a config.json file in the working directory.  If you exclude the IPv6 fetch URL, only IPv4 will be updated.

```
$ cf-ddns
NAME:
   cf-ddns - CloudFlare DDNS client

USAGE:
   cf-ddns [arguments...]

ARGUMENTS:
  -config string
    	config file path (default "config.json")
  -force
    	force update

GLOBAL OPTIONS:
   --help, -h		show help
```

