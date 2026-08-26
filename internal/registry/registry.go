package registry

import (
	"log"
	"sync"
	"time"
)

type Peer struct {
	Name        string    `json:"name"`
	Fingerprint string    `json:"fingerprint"`
	Addr        string    `json:"addr"`
	TCPPort     int       `json:"tcp_port"`
	LastSeen    time.Time `json:"last_seen"`
}

type Registry struct {
	mu    sync.Mutex
	peers map[string]Peer
	ttl   time.Duration
}

func New(ttl time.Duration) *Registry {
	return &Registry{
		peers: make(map[string]Peer),
		ttl:   ttl,
	}
}

func (r *Registry) Update(name, fingerprint, addr string, tcpPort int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, known := r.peers[fingerprint]
	r.peers[fingerprint] = Peer{
		Name:        name,
		Fingerprint: fingerprint,
		Addr:        addr,
		TCPPort:     tcpPort,
		LastSeen:    time.Now(),
	}
	if !known {
		log.Printf("discovered %s [%s] at %s", name, fingerprint, addr)
	}
}

func (r *Registry) Sweep() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for fingerprint, peer := range r.peers {
		if time.Since(peer.LastSeen) > r.ttl {
			delete(r.peers, fingerprint)
			log.Printf("expired %s [%s]", peer.Name, fingerprint)
		}
	}
}

func (r *Registry) SweepEvery(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		r.Sweep()
	}
}

func (r *Registry) Snapshot() []Peer {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]Peer, 0, len(r.peers))
	for _, peer := range r.peers {
		out = append(out, peer)
	}

	return out
}
