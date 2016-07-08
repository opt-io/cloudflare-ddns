# cloudflare-ddns
A command-line cloudflare DDNS updater written in Go, using the CloudFlare GO API library: https://github.com/cloudflare/cloudflare-go

# Usage

You must update and include a config.json file in the working directory.  The last fetched IP is stored in the config file, to limit CloudFlare API activity.

You can exclude the IPv6 or IPv4 fetch URL to force a single protocol.  When fetching an IP, the request is made over the respective protocol.

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

