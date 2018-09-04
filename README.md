# VeilD

Stub resolver for routing DNS queries over TLS (DNS-over-TLS).

## Features

- Roundrobin of requests over each DNS server
- Caches responses and adhers to TTLs
- Ability to blacklist domains using a supplied file (txt file of domains to block)
- Ability to define a list of resolvers in a YAML file

## Todo

- Limit size of cache