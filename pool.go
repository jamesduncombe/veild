package veild

import (
	"log/slog"
	"time"
)

const (
	reconnectionQueueSize = 10
	requestQueueSize      = 100
)

const statsFrequency = 10 * time.Second

// Pool represents a new connection pool.
type Pool struct {
	workers   chan *Worker
	reconnect chan *Worker
	requests  chan *Request
	log       *slog.Logger
}

// NewPool creates a new connection pool.
func NewPool(logger *slog.Logger, workerQueueSize int) *Pool {
	return &Pool{
		workers:   make(chan *Worker, workerQueueSize),
		reconnect: make(chan *Worker, reconnectionQueueSize),
		requests:  make(chan *Request, requestQueueSize),
		log:       logger.With("module", "pool"),
	}
}

// Stats prints out connection stats every x seconds.
func (p *Pool) Stats() {
	for {
		p.log.Info("Stats", "requests", len(p.requests), "reconnecting", len(p.reconnect), "workers", len(p.workers))
		time.Sleep(statsFrequency)
	}
}

// ConnectionManagement management handles reconnects.
func (p *Pool) ConnectionManagement() {
	for reconnect := range p.reconnect {
		p.log.Info("Reconnecting", "host", reconnect.host)

		// Let's see how many are reconnecting and how many workers we have.
		p.log.Info("Stats", "requests", len(p.requests), "reconnecting", len(p.reconnect), "workers", len(p.workers))

		w := NewWorker(reconnect.host, reconnect.serverName)
		p.AddWorker(w)
	}
}

// AddWorker adds a new worker to the pool.
func (p *Pool) AddWorker(w *Worker) {
	go p.worker(w)
}

// worker creates a new underlying pconn and assigns it a ResponseCache.
func (p *Pool) worker(worker *Worker) {

	// Each pconn has it's own ResponseCache.
	responseCache := NewResponseCache(p.log)

	// Start a new connection.
	// TODO: Return an error?
	pconn, err := NewPConn(responseCache, worker, p.log)
	if err != nil {
		p.log.Warn("Failed to add a new connection", "host", worker.host)
		return
	}

	// Put the worker into the pool.
	p.workers <- worker

	// Enter the loop for the worker.
	for {
		select {
		case <-pconn.closeCh:
			p.log.Info("PConn gone")
			worker.done <- struct{}{}
			return
		case req := <-p.requests:
			p.log.Debug("Pulled request from worker, pushing to upstream",
				"host",
				pconn.host,
				"pconn_requests",
				len(pconn.writeCh))
			pconn.writeCh <- req
		}
	}
}

// Dispatch handles dispatching requests to the underlying workers.
func (p *Pool) Dispatch() {
	for {
		p.log.Debug("Waiting for outgoing requests...")

		// Pull a request off.
		request := <-p.requests

		p.log.Debug("Workers and requests in pool", "workers", len(p.workers), "requests", len(p.requests))

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
			// Else, put the worker back into the pool and
			// stick the request back on the requests queue.
			p.log.Debug("Worker still alive, forwarding request", "worker", worker.serverName)

			p.workers <- worker
			p.requests <- request

			p.log.Debug("Worker returned to pool", "worker", worker.serverName)
		}

	}
}
