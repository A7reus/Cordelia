package registry

import (
	"testing"
	"time"
)

func TestRegistryUpdateAndSnapshot(t *testing.T) {
	r := New(10 * time.Second)
	r.Update("alice", "fp1", "cert1", "192.168.1.2", 47777)
	r.Update("bob", "fp2", "cert2", "192.168.1.3", 47778)

	peers := r.Snapshot()
	if len(peers) != 2 {
		t.Fatalf("expected 2 peers, got %d", len(peers))
	}

	found := make(map[string]bool)
	for _, p := range peers {
		found[p.Fingerprint] = true
	}
	if !found["fp1"] || !found["fp2"] {
		t.Fatalf("missing fingerprints in snapshot")
	}
}

func TestRegistrySweep(t *testing.T) {
	r := New(50 * time.Millisecond)
	r.Update("alice", "fp1", "cert1", "192.168.1.2", 47777)

	time.Sleep(60 * time.Millisecond)
	r.Sweep()
	if len(r.Snapshot()) != 0 {
		t.Fatalf("expected 0 peers after sweep, got %d", len(r.Snapshot()))
	}

	r.Update("bob", "fp2", "cert2", "192.168.1.3", 47778)
	r.Update("alice", "fp1", "cert1", "192.168.1.2", 47777)
	time.Sleep(20 * time.Millisecond)
	r.Sweep()
	if len(r.Snapshot()) != 2 {
		t.Fatalf("expected 2 peers before ttl, got %d", len(r.Snapshot()))
	}
}

func TestRegistryConcurrent(t *testing.T) {
	r := New(10 * time.Second)
	done := make(chan struct{})
	go func() {
		for range 100 {
			r.Update("alice", "fp1", "cert1", "192.168.1.2", 47777)
		}
		close(done)
	}()
	for range 100 {
		_ = r.Snapshot()
		r.Sweep()
	}
	<-done
}
