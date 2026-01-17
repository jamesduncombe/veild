package veild

import (
	"bufio"
	"log/slog"
	"os"
	"regexp"
	"sync"
)

// Blocklist represents a DNS blocklist.
type Blocklist struct {
	mu   sync.Mutex
	list map[string]struct{}
	log  *slog.Logger
}

// NewBlocklist creates a new Blocklist from a given hosts file.
func NewBlocklist(blocklistPath string, logger *slog.Logger) (*Blocklist, error) {

	// Parse and load the blocklist.
	blocklist, err := parseBlocklist(blocklistPath)
	if err != nil {
		return nil, err
	}

	return &Blocklist{
		list: blocklist,
		log:  logger.With("module", "blocklist"),
	}, nil
}

// Exists returns a boolean as to whether this entry was found or not in the list.
func (b *Blocklist) Exists(item string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.list[item]; ok {
		return true
	}
	return false
}

// parseBlocklist handles parsing of a hosts file.
func parseBlocklist(blocklistPath string) (map[string]struct{}, error) {

	blocklistFile, err := os.Open(blocklistPath)
	if err != nil {
		return nil, err
	}
	defer blocklistFile.Close()

	blocklist := make(map[string]struct{})
	pattern := regexp.MustCompile(`^[^#].+\s+([A-Za-z\-0-9\.]+)$`)
	scanner := bufio.NewScanner(blocklistFile)

	for scanner.Scan() {
		text := scanner.Text()
		match := pattern.FindStringSubmatch(text)
		if len(match) > 1 {
			blocklist[match[1]] = struct{}{}
		}

	}

	return blocklist, nil
}
