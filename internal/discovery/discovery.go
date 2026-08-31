package discovery

import (
	"encoding/json"
	"log"
	"net"
	"time"

	"github.com/A7reus/Cordelia/internal/identity"
	"github.com/A7reus/Cordelia/internal/registry"
)

const (
	groupAddress = "239.255.77.77:47777"
	interval     = 3 * time.Second
	maxDatagram  = 1024
)

type Announcement struct {
	Name            string `json:"name"`
	Fingerprint     string `json:"fingerprint"`
	CertFingerprint string `json:"cert_fingerprint"`
	TCPPort         int    `json:"tcp_port"`
}

func Announce(id identity.Identity, certFingerprint string, tcpPort int) {
	group, err := net.ResolveUDPAddr("udp4", groupAddress)
	if err != nil {
		log.Fatal("discovery:", err)
	}

	conn, err := net.DialUDP("udp4", nil, group)
	if err != nil {
		log.Fatal("discovery:", err)
	}
	defer conn.Close()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	send := func() {
		data, err := json.Marshal(Announcement{
			Name:            id.Name,
			Fingerprint:     id.Fingerprint,
			CertFingerprint: certFingerprint,
			TCPPort:         tcpPort,
		})
		if err != nil {
			log.Printf("discovery: send: %v", err)
		}
		if _, err := conn.Write(data); err != nil {
			log.Printf("discovery: send: %v", err)
		}
	}

	send()
	for range ticker.C {
		send()
	}
}

func Listen(self identity.Identity, reg *registry.Registry) {
	group, err := net.ResolveUDPAddr("udp4", groupAddress)
	if err != nil {
		log.Fatal("discovery:", err)
	}

	conn, err := net.ListenMulticastUDP("udp4", nil, group)
	if err != nil {
		log.Fatal("discovery:", err)
	}
	defer conn.Close()

	buf := make([]byte, maxDatagram)
	for {
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("discovery: read: %v", err)
			continue
		}

		var ann Announcement
		if err := json.Unmarshal(buf[:n], &ann); err != nil {
			continue
		}
		if ann.Fingerprint == self.Fingerprint {
			continue
		}

		reg.Update(ann.Name, ann.Fingerprint, ann.CertFingerprint, src.IP.String(), ann.TCPPort)
	}
}
