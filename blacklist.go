package veild

import (
	"bufio"
	"os"
	"regexp"
	"sync"
)

// Blacklist represents a blacklist.
type Blacklist struct {
	mu   sync.Mutex
	list map[string]struct{}
}

// NewBlacklist creates a new Blacklist from a given hosts file.
func NewBlacklist(blacklistPath string) (*Blacklist, error) {

	// Parse and load the blacklist.
	blacklist, err := parseBlacklist(blacklistPath)
	if err != nil {
		return nil, err
	}

	return &Blacklist{
		list: blacklist,
	}, nil
}

// Exists returns a boolean as to whether this entry was found or not in the list.
func (b *Blacklist) Exists(item string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.list[item]; ok {
		return true
	}
	return false
}

// parseBlacklist handles parsing of a hosts file.
func parseBlacklist(blacklistPath string) (map[string]struct{}, error) {

	blacklistFile, err := os.Open(blacklistPath)
	if err != nil {
		return nil, err
	}
	defer blacklistFile.Close()

	blacklist := make(map[string]struct{})
	pattern := regexp.MustCompile(`^[^#].+\s+([A-Za-z\-0-9\.]+)$`)
	scanner := bufio.NewScanner(blacklistFile)

	for scanner.Scan() {
		text := scanner.Text()
		match := pattern.FindStringSubmatch(text)
		if len(match) > 1 {
			blacklist[match[1]] = struct{}{}
		}

	}

	return blacklist, nil
}
