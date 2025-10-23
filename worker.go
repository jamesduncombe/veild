package veild

// Worker represents a worker in the pool.
type Worker struct {
	host       string
	serverName string
	done       chan struct{}
}

// NewWorker creates a new worker.
func NewWorker(host, serverName string) *Worker {
	return &Worker{
		host:       host,
		serverName: serverName,
		done:       make(chan struct{}),
	}
}
