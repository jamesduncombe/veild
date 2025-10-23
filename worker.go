package veild

// Worker represents a worker in the pool.
type Worker struct {
	host       string
	serverName string
	done       chan struct{}
}

// NewWorker adds a new worker to the Pool.
func (p *Pool) NewWorker(host, serverName string) *Worker {
	return &Worker{
		host:       host,
		serverName: serverName,
		done:       make(chan struct{}),
	}
}
