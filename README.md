# veild

[![Build Status](https://github.com/jamesduncombe/veild/actions/workflows/ci.yml/badge.svg)](https://github.com/jamesduncombe/veild/actions) [![Go Report Card](https://goreportcard.com/badge/github.com/jamesduncombe/veild)](https://goreportcard.com/report/github.com/jamesduncombe/veild) [![godoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/jamesduncombe/veild)

Stub resolver for routing DNS queries over TLS (DNS-over-TLS).

Thanks to the following sites/RFCs:

- [https://tools.ietf.org/html/rfc1035](https://tools.ietf.org/html/rfc1035)
- [https://tools.ietf.org/html/rfc7858](https://tools.ietf.org/html/rfc7858)
- [https://dnsprivacy.org/wiki/display/DP/DNS+Privacy+Test+Servers](https://dnsprivacy.org/wiki/display/DP/DNS+Privacy+Test+Servers)

## Features

- Roundrobin of requests over each DNS server
- Caches responses and adhers to TTLs
- Blocklist domains using a supplied file (txt file of domains to block)
- Ability to define a list of resolvers in a YAML file

## Install

[Head on over to the latest releases page](https://github.com/jamesduncombe/veild/releases) to pick up your release of choice :)

## Usage

The quickest and easiest way to get started, assuming you've extracted the archive and are in the directory:

```sh
sudo ./veild
```

This will start `veild` with caching on and a resolvers set to [Quad9's](https://www.quad9.net/) 9.9.9.9 and [Mullvad's](https://mullvad.net/en/help/dns-over-https-and-dns-over-tls) 194.242.2.9 servers.

Why do I need sudo?! Well, by default veild listens on port `53` (UDP) which is within the privileged ports range... more on that [here](https://www.w3.org/Daemon/User/Installation/PrivilegedPorts.html).

Hopefully you should see it startup with output similar to the following:

```sh
$ sudo ./veild
2025/04/06 16:59:03 Starting Veil
2025/04/06 16:59:03 [main] Outbound port set to 853
2025/04/06 16:59:03 [main] Listening on 127.0.0.1:53 (UDP)
```

If you do... good stuff!

Time to set your resolver to your nice, new, fresh super secure™ resolver.

- [Linux instructions](https://www.techrepublic.com/article/how-to-set-dns-nameservers-in-ubuntu-server-18-04/)
- [MacOS instructions](http://osxdaily.com/2015/12/05/change-dns-server-settings-mac-os-x/)
- [Windows instructions](https://www.lifewire.com/how-to-change-dns-servers-in-windows-2626242)

When your OS is set to use veild you should start to see some activity in the console.

### Resolvers

Out of the box `veild` is set to use the following resolvers config:

```yaml
resolvers:
  - address: "9.9.9.9:853"
    hostname: "dns.quad9.net"
  - address: "194.242.2.9:853"
    hostname: "all.dns.mullvad.net"
```

To add your own, create a file called `resolvers.yml` (for example) and add other resolvers. Here are a few other options to get you started (uncomment them and restart `veild` as needed):

```yaml
resolvers:

  # Cloudflare DNS servers
  # - address: "1.1.1.1:853"
  # hostname: "cloudflare-dns.com"

  # - address: "1.0.0.1:853"
  #   hostname: "cloudflare-dns.com"

  # Google DNS servers
  #- address: "8.8.8.8:853"
  #  hostname: "dns.google"

  # Quad9 DNS servers
  # See: https://www.quad9.net/faq/#Does_Quad9_support_DNS_over_TLS
  # Secure IP: 9.9.9.9 Provides: Security blocklist, DNSSEC, No EDNS Client-Subnet sent.
  - address: "9.9.9.9:853"
    hostname: "dns.quad9.net"

  # Mullvad DNS servers
  # See: https://mullvad.net/en/help/dns-over-https-and-dns-over-tls
  - address: "194.242.2.9:853"
    hostname: "all.dns.mullvad.net"

  # Unsecured IP: 9.9.9.10 Provides: No security blocklist, no DNSSEC, sends EDNS Client-Subnet.
  # - address: "9.9.9.10:853"
  #   hostname: "dns.quad9.net"
```

### Blocklists

Support is also available to block ad domains etc. Head to https://github.com/hagezi/dns-blocklists where you can find multiple blocklists available for download.

As a headstart, try the "Multi Normal" (all round protection list) here: https://github.com/hagezi/dns-blocklists/tree/main?tab=readme-ov-file#normal. Look for the `Hosts` format and download from there.

`veild` is happy working with the hosts file format, so, once you have a blocklist downloaded, simply add: `-b blocklist.txt` to the end of the command above.

I think that just about covers things... for a full set of the arguments that you can pass to veild run: `./veild -help`

## Todo

- Limit size of cache
- Add ability to remap domain requests
