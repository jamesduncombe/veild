package veild

import (
	"bufio"
	"io"
	"os"
	"regexp"
	"sync"
)

// Blacklist represents a blacklist.
type Blacklist struct {
	mu   sync.Mutex
	list map[string]string
}

// NewBlacklist creates a new Blacklist from a given hosts file.
func NewBlacklist(blacklistPath string) (*Blacklist, error) {

	// Init the blacklist.
	file, err := os.Open(blacklistPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	list := ParseBlacklist(file)

	return &Blacklist{
		list: list,
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

// ParseBlacklist handles parsing of a hosts file.
func ParseBlacklist(file io.Reader) map[string]string {

	list := make(map[string]string)
	pattern := regexp.MustCompile(`^[^#].+\s+([A-Za-z\-0-9\.]+)$`)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		text := scanner.Text()

		// Match on the regex.
		match := pattern.FindStringSubmatch(text)
		if len(match) > 1 {
			list[match[1]] = match[1]
		}

	}

	return list
}
