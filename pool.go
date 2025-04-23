package veild

import (
	"log"
	"os"
	"time"
)

// Worker represents a worker in the pool.
type Worker struct {
	host       string
	serverName string
	requests   chan Request
	done       chan struct{}
	closed     bool
}

// Pool represents a new connection pool.
type Pool struct {
	workers   chan Worker
	reconnect chan Worker
	requests  chan Request
	log       *log.Logger
}

// NewPool creates a new connection pool.
// TODO: Worker channels should probably be scoped to the size of the resolvers?
func NewPool() *Pool {
	return &Pool{
		workers:   make(chan Worker, 10),
		reconnect: make(chan Worker, 10),
		requests:  make(chan Request, 10),
		log:       log.New(os.Stdout, "[pool] ", log.LstdFlags|log.Lmsgprefix),
	}
}

// NewWorker adds a new worker to the Pool.
func (p *Pool) NewWorker(host, serverName string) {
	w := Worker{
		host:       host,
		serverName: serverName,
		requests:   make(chan Request),
		done:       make(chan struct{}),
		closed:     false,
	}
	p.workers <- w
	go p.worker(w)
}

// Stats prints out connection stats every x seconds.
func (p *Pool) Stats() {
	for {
		p.log.Printf("[stats] Requests: %d, Reconnecting: %d, Workers: %d\n", len(p.requests), len(p.reconnect), len(p.workers))
		time.Sleep(10 * time.Second)
	}
}

// ConnectionManagement management handles reconnects.
func (p *Pool) ConnectionManagement() {
	for reconnect := range p.reconnect {
		p.log.Printf("Reconnecting %s\n", reconnect.host)
		p.NewWorker(reconnect.host, reconnect.serverName)
	}
}

// worker creates a new underlying pconn and assigns it a ResponseCache.
func (p *Pool) worker(w Worker) {

	// Each pconn has it's own ResponseCache.
	responseCache := NewResponseCache()

	// Start a new connection.
	pconn, err := NewPConn(responseCache, w.host, w.serverName)
	if err != nil {
		p.log.Printf("Failed to add a new connection to %s\n", w.host)
		return
	}

	// Enter the loop for the worker.
	for {
		select {
		case <-pconn.closeCh:
			p.log.Println("PConn gone")
			w.done <- struct{}{}
			return
		case req := <-w.requests:
			pconn.writeCh <- req
		}
	}
}

// Dispatch handles dispatching requests to the underlying workers.
func (p *Pool) Dispatch() {
	for {
		select {
		// Pull a packet off
		case request := <-p.requests:
			select {
			// Grab a worker.
			case worker := <-p.workers:
				p.log.Printf("Worker: %s\n", worker.host)
				select {
				// If the worker is done, then requeue the packet and also raise a reconnection attempt.
				case <-worker.done:
					p.log.Printf("Worker down: %s\n", worker.host)
					p.reconnect <- worker
					p.requests <- request
					// Else, write the packet to the workers queue.
					// stick the worker back on the stack.
				default:
					worker.requests <- request
					p.workers <- worker
				}
			// We're out of luck.
			default:
				p.requests <- request
				p.log.Println("No workers left")
				time.Sleep(2 * time.Second)
			}
		// Every 5 seconds or there abouts check each worker in turn
		// we want to keep connections as hot as possible.
		case <-time.After(5 * time.Second):
			worker := <-p.workers
			select {
			case <-worker.done:
				p.log.Printf("Worker %s gone, reconnecting.", worker.host)
				p.reconnect <- worker
			default:
				p.workers <- worker
			}
		}
	}
}
