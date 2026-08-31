package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/A7reus/Cordelia/internal/certs"
	"github.com/A7reus/Cordelia/internal/client"
	"github.com/A7reus/Cordelia/internal/discovery"
	"github.com/A7reus/Cordelia/internal/identity"
	"github.com/A7reus/Cordelia/internal/registry"
	"github.com/A7reus/Cordelia/internal/server"
)

const defaultPort = 47777

var version = "dev"

var (
	port    = flag.Int("port", defaultPort, "TCP API port")
	dataDir = flag.String("data-dir", "", "config directory override (for testing)")
	outDir  = flag.String("out", "", "download directory (default ~/Downloads/cordelia)")
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
			client.Probe(args[1])
			return
		case "peers":
			client.FetchPeers(*port)
			return
		case "send-text":
			if len(args) < 3 {
				log.Fatalf("usage: cordelia send-text <host:port> <text>")
			}
			client.SendText(args[1], id.Name, strings.Join(args[2:], " "))
			return
		case "send-file":
			if len(args) < 3 {
				log.Fatalf("usage: cordelia send-file <host:port> <file> [file...]")
			}
			for _, fp := range args[2:] {
				client.SendFile(args[1], fp)
			}
			return
		case "version":
			fmt.Println(version)
			return
		default:
			log.Fatalf("unknown command %q", args[0])
		}
	}

	reg := registry.New(10 * time.Second)
	go reg.SweepEvery(3 * time.Second)
	go discovery.Listen(id, reg)
	go discovery.Announce(id, *port)

	downloadDir := server.ResolveDownloadDir(*outDir)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /info", server.InfoHandler(id))
	mux.HandleFunc("GET /peers", server.PeersHandler(reg))
	mux.HandleFunc("POST /message", server.MessageHandler())
	mux.HandleFunc("POST /upload", server.UploadHandler(downloadDir))

	cert, err := certs.LoadOrCreate(*dataDir)
	if err != nil {
		log.Fatalf("cert: %v", err)
	}

	addr := fmt.Sprintf(":%d", *port)
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		},
	}

	log.Printf("serving API on %s with TLS", addr)
	if err := srv.ListenAndServeTLS("", ""); err != nil {
		log.Fatal(err)
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
