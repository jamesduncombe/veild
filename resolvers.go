package veild

import (
	"gopkg.in/yaml.v2"
)

// defaultResolver is a YML config for DNS resolvers if no resolvers file is given.
const defaultResolver = `
resolvers:
  - address: "1.1.1.1:853"
    hostname: "cloudflare-dns.com"
  
  - address: "1.0.0.1:853"
    hostname: "cloudflare-dns.com"
`

// Resolver implements a resolver.
type Resolver struct {
	Address  string
	Hostname string
	Hash     string
	Pin      string
}

// Resolvers implements a list of resolvers.
type Resolvers struct {
	Resolvers []Resolver
}

// LoadResolvers loads of a list of resolvers from a YAML file.
func LoadResolvers(d []byte) (Resolvers, error) {
	l := Resolvers{}
	if err := yaml.Unmarshal(d, &l); err != nil {
		return Resolvers{}, err
	}
	return l, nil
}
