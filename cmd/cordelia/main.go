package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/A7reus/Cordelia/internal/certs"
	"github.com/A7reus/Cordelia/internal/client"
	"github.com/A7reus/Cordelia/internal/config"
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

	cfg, err := config.Load(*dataDir)
	if err != nil {
		log.Printf("config: %v", err)
		cfg = config.Default()
	}
	if *port == defaultPort && cfg.Port != 0 && cfg.Port != defaultPort {
		*port = cfg.Port
	}
	if *outDir == "" && cfg.OutDir != "" {
		*outDir = cfg.OutDir
	}

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
			if len(args) < 2 {
				log.Fatalf("usage: cordelia send-text [host:port] <text>")
			}

			var host, text string
			if len(args) == 2 {
				h, err := client.PickPeer(*port)
				if err != nil {
					log.Fatalf("pick peer: %v", err)
				}
				host = h
				text = args[1]
			} else {
				if _, _, err := net.SplitHostPort(args[1]); err == nil {
					host = args[1]
					text = strings.Join(args[2:], " ")
				} else {
					h, err := client.PickPeer(*port)
					if err != nil {
						log.Fatalf("pick peer: %v", err)
					}
					host = h
					text = strings.Join(args[1:], " ")
				}
			}
			if text == "" {
				log.Fatalf("usage: cordelia send-text [host:port] <text>")
			}

			client.SendText(host, id.Name, text, *port)
			return
		case "send-file":
			if len(args) < 2 {
				log.Fatalf("usage: cordelia send-file [host:port] <file> [file...]")
			}

			var host string
			var files []string
			if len(args) == 2 {
				h, err := client.PickPeer(*port)
				if err != nil {
					log.Fatalf("pick peer: %v", err)
				}
				host = h
				files = args[1:]
			} else {
				if _, _, err := net.SplitHostPort(args[1]); err == nil {
					host = args[1]
					files = args[2:]
				} else {
					h, err := client.PickPeer(*port)
					if err != nil {
						log.Fatalf("pick peer: %v", err)
					}
					host = h
					files = args[1:]
				}
			}

			for _, fp := range files {
				client.SendFile(host, fp, *port)
			}
			return
		case "version":
			fmt.Println(version)
			return
		default:
			log.Fatalf("unknown command %q", args[0])
		}
	}

	cert, err := certs.LoadOrCreate(*dataDir)
	if err != nil {
		log.Fatalf("cert: %v", err)
	}
	certFingerprint, err := certs.Fingerprint(cert)
	if err != nil {
		log.Fatalf("cert fingerprint: %v", err)
	}

	reg := registry.New(config.TTLOrDefault(cfg))
	go reg.SweepEvery(3 * time.Second)
	go discovery.Listen(id, reg)
	go discovery.Announce(id, certFingerprint, *port)

	downloadDir := server.ResolveDownloadDir(*outDir)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/info", server.InfoHandler(id))
	mux.HandleFunc("GET /info", server.InfoHandler(id))
	mux.HandleFunc("GET /v1/peers", server.PeersHandler(reg))
	mux.HandleFunc("GET /peers", server.PeersHandler(reg))
	mux.HandleFunc("POST /v1/message", server.MessageHandler())
	mux.HandleFunc("POST /message", server.MessageHandler())
	mux.HandleFunc("POST /v1/upload", server.UploadHandler(downloadDir))
	mux.HandleFunc("POST /upload", server.UploadHandler(downloadDir))

	addr := fmt.Sprintf(":%d", *port)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		},
	}

	log.Printf("serving API on %s with TLS", addr)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := server.ListenAndServeTLS("", ""); err != nil {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
	log.Println("server stopped")
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
