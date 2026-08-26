package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/A7reus/Cordelia/internal/discovery"
	"github.com/A7reus/Cordelia/internal/identity"
	"github.com/A7reus/Cordelia/internal/registry"
)

const defaultPort = 47777

var (
	port    = flag.Int("port", defaultPort, "TCP API port")
	dataDir = flag.String("data-dir", "", "config directory override (for testing)")
)

func main() {
	flag.Parse()

	id, err := identity.Load(*dataDir)
	if errors.Is(err, os.ErrNotExist) {
		id, err = createIdentity(*dataDir)
	}
	if err != nil {
		log.Fatal("cordelia:", err)
	}

	if args := flag.Args(); len(args) > 0 {
		switch args[0] {
		case "probe":
			if len(args) != 2 {
				log.Fatal("usage: cordelia probe <host:port>")
			}
			probe(args[1])
			return
		case "peers":
			fetchPeers(*port)
			return
		default:
			log.Fatalf("unknown command %q", args[0])
		}
	}

	reg := registry.New(10 * time.Second)
	go reg.SweepEvery(3 * time.Second)
	go discovery.Listen(id, reg)
	go discovery.Announce(id, *port)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /info", infoHandler(id))
	mux.HandleFunc("GET /peers", peersHandler(reg))

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Serving API on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func probe(addr string) {
	client := http.Client{Timeout: 3 * time.Second}

	res, err := client.Get(fmt.Sprintf("http://%s/info", addr))
	if err != nil {
		log.Fatalf("probe %s: %v", addr, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Fatalf("probe %s: got status %s", addr, res.Status)
	}

	var peer identity.Identity
	if err := json.NewDecoder(res.Body).Decode(&peer); err != nil {
		log.Fatalf("probe %s: bad response: %v", addr, err)
	}

	log.Printf("Found peer %s [%s]", peer.Name, peer.Fingerprint)
}

func infoHandler(id identity.Identity) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(id)
	}
}

func peersHandler(reg *registry.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(reg.Snapshot())
	}
}

func fetchPeers(port int) {
	client := http.Client{Timeout: 3 * time.Second}

	res, err := client.Get(fmt.Sprintf("http://localhost:%d/peers", port))
	if err != nil {
		log.Fatalf("peers: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Fatalf("peers: got status %s", res.Status)
	}

	var peers []registry.Peer
	if err := json.NewDecoder(res.Body).Decode(&peers); err != nil {
		log.Fatalf("peers: bad response: %v", err)
	}

	sort.Slice(peers, func(i, j int) bool {
		return peers[i].Name < peers[j].Name
	})

	for _, peer := range peers {
		fmt.Printf("%-24s %-34s %15s:%d\n", peer.Name, peer.Fingerprint, peer.Addr, peer.TCPPort)
		fmt.Printf("%d peer(s)\n", len(peers))
	}
}

func createIdentity(dataDir string) (identity.Identity, error) {
	name := "cordelia-device"
	if host, err := os.Hostname(); err == nil && host != "" {
		name = host
	}

	id, err := identity.New(name)
	if err != nil {
		return identity.Identity{}, err
	}
	if err := identity.Save(id, dataDir); err != nil {
		return identity.Identity{}, err
	}

	log.Println("New identity created")
	return id, nil
}
