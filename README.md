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
- Blacklist domains using a supplied file (txt file of domains to block)
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

The `resolvers.yml` file which you'll see in the archive also gives you the ability to enable/disable DNS resolvers as needed. I've added comments in there which should explain things.

### Blacklists

Blacklist support is also available to block ad domains etc. For that you'll need to head to [Steven Black's repo](https://github.com/StevenBlack/hosts) where you can find multiple blacklists available for download.

Veild is happy working with the hosts file format, so, once you have a blacklist downloaded, simply add: `-b blacklist.txt` to the end of the command above.

I think that just about covers things... for a full set of the arguments that you can pass to veild run: `./veild --help`

## Todo

- Limit size of cache
- Add ability to remap domain requests
