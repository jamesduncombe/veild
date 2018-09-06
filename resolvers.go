package veild

import (
	"io/ioutil"

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

// NewResolvers loads of a list of resolvers from a file.
func NewResolvers(resolversPath string) (*Resolvers, error) {
	resolvers := &Resolvers{}

	var resolversList []byte
	var err error

	if resolversPath == "" {
		resolversList = []byte(defaultResolver)
	} else {
		resolversList, err = ioutil.ReadFile(resolversPath)
		if err != nil {
			return nil, err
		}
	}

	if err := yaml.Unmarshal(resolversList, &resolvers); err != nil {
		return nil, err
	}

	return resolvers, nil
}
