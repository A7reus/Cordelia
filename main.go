package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/A7reus/Cordelia/internal/discovery"
	"github.com/A7reus/Cordelia/internal/identity"
	"github.com/A7reus/Cordelia/internal/registry"
)

const defaultPort = 47777
const maxMessageSize = 64 * 1024

var (
	port    = flag.Int("port", defaultPort, "TCP API port")
	dataDir = flag.String("data-dir", "", "config directory override (for testing)")
)

type Message struct {
	From string `json:"from"`
	Text string `json:"text"`
}

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
		case "send-text":
			if len(args) < 3 {
				log.Fatalf("usage: cordelia send-text <host:port> <text>")
			}
			sendText(args[1], id, strings.Join(args[2:], " "))
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
	mux.HandleFunc("POST /message", messageHandler())

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

func messageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxMessageSize)

		var msg Message
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
				http.Error(w, "message too long", http.StatusRequestEntityTooLarge)
				return
			}

			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		if msg.Text == "" {
			http.Error(w, "empty text", http.StatusBadRequest)
			return
		}

		log.Printf("message from %s: %s", msg.From, msg.Text)
		w.WriteHeader(http.StatusNoContent)
	}
}

func sendText(addr string, self identity.Identity, text string) {
	client := http.Client{Timeout: 3 * time.Second}

	payload, err := json.Marshal(Message{From: self.Name, Text: text})
	if err != nil {
		log.Fatalf("send-text: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s/message", addr), bytes.NewBuffer(payload))
	if err != nil {
		log.Fatalf("send-text: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		log.Fatalf("send-text %s: %v", addr, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(res.Body)
		log.Fatalf("send-text %s: got %s: %s", addr, res.Status, body)
	}

	log.Printf("delivered to %s", addr)
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
		fmt.Printf("%d peers(s)\n", len(peers))
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
