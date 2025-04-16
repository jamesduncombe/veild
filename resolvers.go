package veild

import (
	"os"

	"gopkg.in/yaml.v2"
)

// defaultResolver is a YML config for DNS resolvers if no resolvers file is given.
const defaultResolver = `
resolvers:
  - address: "9.9.9.9:853"
    hostname: "dns.quad9.net"
  - address: "194.242.2.9:853"
    hostname: "all.dns.mullvad.net"
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

// NewResolvers loads of a list of resolvers from a file.
func NewResolvers(resolversPath string) (*Resolvers, error) {
	resolvers := &Resolvers{}

	var resolversList []byte
	var err error

	if resolversPath == "" {
		resolversList = []byte(defaultResolver)
	} else {
		resolversList, err = os.ReadFile(resolversPath)
		if err != nil {
			return nil, err
		}
	}

	if err := yaml.Unmarshal(resolversList, &resolvers); err != nil {
		return nil, err
	}

	return resolvers, nil
}
