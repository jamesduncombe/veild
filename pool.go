package veild

import (
	"log"
	"sync"
	"time"
)

// Worker represents a worker in the pool.
type Worker struct {
	host       string
	serverName string
	requests   chan Packet
	done       chan struct{}
	closed     bool
}

// Pool is a new connection pool.
type Pool struct {
	mu        sync.RWMutex
	workers   chan Worker
	reconnect chan Worker
	packets   chan Packet
}

// NewPool creates a new connection pool.
func NewPool() *Pool {
	return &Pool{
		workers:   make(chan Worker, 10),
		reconnect: make(chan Worker, 10),
		packets:   make(chan Packet, 10),
	}
}

// NewWorker adds a new worker to the Pool.
func (p *Pool) NewWorker(host, serverName string) {
	w := Worker{
		host:       host,
		serverName: serverName,
		requests:   make(chan Packet),
		done:       make(chan struct{}),
		closed:     false,
	}
	p.workers <- w
	go p.worker(w)
}

// Stats prints out connection stats every x seconds.
func (p *Pool) Stats() {
	for {
		log.Printf("[pool] [stats] Packets: %d, Reconnections: %d, Workers: %d\n", len(p.packets), len(p.reconnect), len(p.workers))
		time.Sleep(10 * time.Second)
	}
}

// ConnectionManagement management handles reconnects.
func (p *Pool) ConnectionManagement() {
	for {
		select {
		case reconnect := <-p.reconnect:
			log.Printf("[pool] Reconnecting %s\n", reconnect.host)
			p.NewWorker(reconnect.host, reconnect.serverName)
		}
	}
}

// worker creates a new underlying pconn and assigns it a ResponseCache.
func (p *Pool) worker(w Worker) {

	// Each pconn has it's own ResponseCache.
	responseCache := &ResponseCache{
		responses: make(map[uint16]Packet),
	}

	// Start a new connection.
	pconn, err := NewPConn(p, responseCache, w.host, w.serverName)
	if err != nil {
		log.Printf("[pool] Failed to add a new connection to %s\n", w.host)
		return
	}

	// Enter the loop for the worker.
	for {
		select {
		case <-pconn.closech:
			log.Println("[pool] PConn gone")
			w.done <- struct{}{}
			return
		case req := <-w.requests:
			pconn.writech <- req
		}
	}
}

// Dispatch handles dispatching requests to the underlying workers.
func (p *Pool) Dispatch() {
	for {
		select {
		// Pull a packet off
		case packet := <-p.packets:
			select {
			// Grab a worker.
			case worker := <-p.workers:
				log.Printf("[pool] Worker: %s\n", worker.host)
				select {
				// If the worker is done, then requeue the packet and also raise a reconnection attempt.
				case <-worker.done:
					log.Printf("[pool] Worker down: %s\n", worker.host)
					p.reconnect <- worker
					p.packets <- packet
					// Else, write the packet to the workers queue.
					// stick the worker back on the stack.
				default:
					worker.requests <- packet
					p.workers <- worker
				}
			// We're bang out of luck :()
			default:
				p.packets <- packet
				log.Println("[pool] No workers left")
				time.Sleep(2 * time.Second)
			}
		// Every 5 seconds or there abouts check each worker in turn
		// we want to keep connections as hot as possible.
		case <-time.After(5 * time.Second):
			worker := <-p.workers
			select {
			case <-worker.done:
				log.Printf("[pool] Worker %s gone, reconnecting.", worker.host)
				p.reconnect <- worker
			default:
				p.workers <- worker
			}
		}
	}
}
