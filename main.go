package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/A7reus/Cordelia/internal/identity"
)

const defaultPort = "47777"

func main() {
	id, err := identity.Load()
	if errors.Is(err, os.ErrNotExist) {
		id, err = createIdentity()
	}
	if err != nil {
		log.Fatal("cordelia:", err)
	}

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "probe":
			if len(os.Args) != 3 {
				log.Fatal("usage: cordelia probe <host:port>")
			}
			probe(os.Args[2])
			return
		default:
			log.Fatalf("unknown command %q", os.Args[1])
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /info", infoHandler(id))

	addr := ":" + defaultPort
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

func createIdentity() (identity.Identity, error) {
	name := "cordelia-device"
	if host, err := os.Hostname(); err == nil && host != "" {
		name = host
	}

	id, err := identity.New(name)
	if err != nil {
		return identity.Identity{}, err
	}
	if err := identity.Save(id); err != nil {
		return identity.Identity{}, err
	}

	log.Println("New identity created")
	return id, nil
}
