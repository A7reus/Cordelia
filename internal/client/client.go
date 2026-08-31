package client

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/A7reus/Cordelia/internal/identity"
	"github.com/A7reus/Cordelia/internal/registry"
)

func insecureClient(timeout time.Duration) http.Client {
	return http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
					if len(rawCerts) == 0 {
						return nil
					}
					cert, err := x509.ParseCertificate(rawCerts[0])
					if err != nil {
						return nil
					}

					sum := sha256.Sum256(cert.Raw)
					log.Printf("server cert fingerprint %s", hex.EncodeToString(sum[:]))
					return nil
				},
			},
		},
	}
}

func Probe(addr string) {
	client := insecureClient(3 * time.Second)

	res, err := client.Get(fmt.Sprintf("https://%s/info", addr))
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

func FetchPeers(port int) {
	client := insecureClient(3 * time.Second)

	res, err := client.Get(fmt.Sprintf("https://localhost:%d/peers", port))
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
	}
	fmt.Printf("%d peer(s)\n", len(peers))
}

func SendText(addr, from, text string) {
	client := insecureClient(3 * time.Second)

	payload, err := json.Marshal(map[string]string{"from": from, "text": text})
	if err != nil {
		log.Fatalf("send-text: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("https://%s/message", addr), bytes.NewBuffer(payload))
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

type progressReader struct {
	r       io.Reader
	total   int64
	sent    int64
	lastPct int
	name    string
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.sent += int64(n)
	if p.total > 0 {
		pct := int(p.sent * 100 / p.total)
		if pct != p.lastPct && (pct%10 == 0 || p.sent == p.total) {
			log.Printf("sending %s: %d/%d bytes (%d%%)", p.name, p.sent, p.total, pct)
			p.lastPct = pct
		}
	}
	return n, err
}

func SendFile(addr, filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("send-file: %v", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		log.Fatalf("send-file: %v", err)
	}
	if info.IsDir() {
		log.Fatalf("send-file: %s is a directory", filePath)
	}

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)
	progress := &progressReader{r: file, total: info.Size(), name: filepath.Base(filePath)}

	go func() {
		defer pw.Close()
		defer writer.Close()

		part, err := writer.CreateFormFile("file", filepath.Base(filePath))
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		if _, err := io.Copy(part, progress); err != nil {
			pw.CloseWithError(err)
			return
		}
	}()

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("https://%s/upload", addr), pr)
	if err != nil {
		log.Fatalf("send-file: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := insecureClient(30 * time.Second)
	res, err := client.Do(req)
	if err != nil {
		log.Fatalf("send-file %s: %v", addr, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(res.Body)
		log.Fatalf("send-file %s: got %s: %s", addr, res.Status, body)
	}

	log.Printf("sent %s (%d bytes) to %s", filepath.Base(filePath), info.Size(), addr)
}
