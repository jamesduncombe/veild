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
	resolvers chan *Resolver
	reconnect chan *Resolver
	requests  chan *Request
	log       *slog.Logger
}

// NewPool creates a new connection pool.
func NewPool(logger *slog.Logger, workerQueueSize int) *Pool {
	return &Pool{
		resolvers: make(chan *Resolver, workerQueueSize),
		reconnect: make(chan *Resolver, reconnectionQueueSize),
		requests:  make(chan *Request, requestQueueSize),
		log:       logger.With("module", "pool"),
	}
}

// Stats prints out connection stats every x seconds.
func (p *Pool) Stats() {
	for {
		p.log.Info("Stats", "requests", len(p.requests), "reconnecting", len(p.reconnect), "workers", len(p.resolvers))
		time.Sleep(statsFrequency)
	}
}

// ConnectionManagement management handles reconnects.
func (p *Pool) ConnectionManagement() {
	for resolver := range p.reconnect {
		p.log.Debug("Reconnecting", "host", resolver.resolver.Address)

		// Let's see how many are reconnecting and how many workers we have.
		p.log.Debug("Stats", "requests", len(p.requests), "reconnecting", len(p.reconnect), "workers", len(p.resolvers))

		rd := TLSResolverDialer{}
		p.AddResolver(resolver.resolver, rd)
	}
}

// AddResolver adds a new worker to the pool.
func (p *Pool) AddResolver(resolver ResolverEntry, rd ResolverDialer) {
	go p.worker(resolver, rd)
}

// worker creates a new underlying connection and assigns it a ResponseCache.
func (p *Pool) worker(re ResolverEntry, rd ResolverDialer) {

	// Each resolver has it's own ResponseCache.
	responseCache := NewResponseCache(p.log)

	// Start a new connection.
	// TODO: Return an error?
	resolver, err := NewResolver(responseCache, re, rd, p.log)
	if err != nil {
		p.log.Warn("Failed to add a new connection", "host", re.Address, "err", err)
		return
	}

	// Put the worker into the pool.
	p.resolvers <- resolver

	// Enter the loop for the worker.
	for {
		select {
		case <-resolver.closeCh:
			p.log.Debug("Resolver gone")
			resolver.doneCh <- struct{}{}
			return
		case req := <-p.requests:
			p.log.Debug("Pulled request from worker, pushing to upstream",
				"host", re.Address, "resolver_requests", len(resolver.writeCh))
			resolver.writeCh <- req
		}
	}
}

// Dispatch handles dispatching requests to the underlying workers.
func (p *Pool) Dispatch() {
	for {
		p.log.Debug("Waiting for outgoing requests...")

		// Pull a request off.
		request := <-p.requests

		p.log.Debug("Workers and requests in pool", "workers", len(p.resolvers), "requests", len(p.requests))

		// Pull a worker off.
		resolver := <-p.resolvers

		p.log.Debug("Worker picked up", "worker", resolver.resolver.Hostname)

		select {
		case <-resolver.doneCh:
			// Worker is down, reconnect and re-queue request.
			p.reconnect <- resolver
			p.requests <- request

			p.log.Debug("Worker down", "worker", resolver.resolver.Hostname)

		default:
			// Else, put the worker back into the pool and
			// stick the request back on the requests queue.
			p.log.Debug("Worker still alive, forwarding request", "worker", resolver.resolver.Hostname)

			p.resolvers <- resolver
			p.requests <- request

			p.log.Debug("Worker returned to pool", "worker", resolver.resolver.Hostname)
		}

	}
}
