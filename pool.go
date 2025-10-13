package veild

import (
	"log/slog"
	"os"
	"time"
)

const (
	workerQueueSize       = 10
	reconnectionQueueSize = 10
	requestQueueSize      = 10
)

const statsFrequency = 10 * time.Second

// Worker represents a worker in the pool.
type Worker struct {
	host       string
	serverName string
	requests   chan *Request
	done       chan struct{}
}

// Pool represents a new connection pool.
type Pool struct {
	workers   chan *Worker
	reconnect chan *Worker
	requests  chan *Request
	log       *slog.Logger
}

// NewPool creates a new connection pool.
// TODO: Worker channels should probably be scoped to the size of the resolvers?
func NewPool() *Pool {
	return &Pool{
		workers:   make(chan *Worker, workerQueueSize),
		reconnect: make(chan *Worker, reconnectionQueueSize),
		requests:  make(chan *Request, requestQueueSize),
		log:       slog.New(slog.NewTextHandler(os.Stdout, nil)).With("module", "pool"),
	}
}

// NewWorker adds a new worker to the Pool.
func (p *Pool) NewWorker(host, serverName string) *Worker {
	return &Worker{
		host:       host,
		serverName: serverName,
		requests:   make(chan *Request),
		done:       make(chan struct{}),
	}
}

// Stats prints out connection stats every x seconds.
func (p *Pool) Stats() {
	for {
		p.log.Info("[stats] Requests: %d, Reconnecting: %d, Workers: %d", len(p.requests), len(p.reconnect), len(p.workers))
		time.Sleep(statsFrequency)
	}
}

// ConnectionManagement management handles reconnects.
func (p *Pool) ConnectionManagement() {
	for reconnect := range p.reconnect {
		p.log.Info("Reconnecting", "host", reconnect.host)

		// Let's see how many are reconnecting and how many workers we have.
		p.log.Info("Stats...", "reconnecting", len(p.reconnect), "workers", len(p.workers))

		w := p.NewWorker(reconnect.host, reconnect.serverName)
		p.AddWorker(w)
	}
}

func (p *Pool) AddWorker(w *Worker) {
	p.workers <- w
	go p.worker(w)
}

// worker creates a new underlying pconn and assigns it a ResponseCache.
func (p *Pool) worker(worker *Worker) {

	// Each pconn has it's own ResponseCache.
	responseCache := NewResponseCache()

	// Start a new connection.
	// TODO: Return an error?
	pconn, err := NewPConn(responseCache, worker)
	if err != nil {
		p.log.Warn("Failed to add a new connection", "host", worker.host)
		return
	}

	// Enter the loop for the worker.
	for {
		select {
		case <-pconn.closeCh:
			p.log.Info("PConn gone")
			worker.done <- struct{}{}
			return
		case req := <-worker.requests:
			pconn.writeCh <- req
		}
	}
}

// Dispatch handles dispatching requests to the underlying workers.
func (p *Pool) Dispatch() {
	for {
		p.log.Debug("Waiting for requests...")

		// Pull a request off.
		request := <-p.requests

		p.log.Debug("Workers in pool", "workers", len(p.workers))
		p.log.Debug("Requests in pool", "requests", len(p.requests))

		// Pull a worker off.
		worker := <-p.workers

		p.log.Debug("Worker picked up", "worker", worker.serverName)

		select {
		case <-worker.done:
			// Worker is down, reconnect and re-queue request.
			p.reconnect <- worker
			p.requests <- request

			p.log.Debug("Worker down", "worker", worker.serverName)

		default:
			// Else, write the packet to the workers queue.
			// stick the worker back on the stack.
			p.log.Debug("Worker still alive, forwarding request", "worker", worker.serverName)

			worker.requests <- request
			p.workers <- worker

			p.log.Debug("Worker returned to pool", "worker", worker.serverName)
		}

	}
}
