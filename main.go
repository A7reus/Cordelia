package main

import (
	"errors"
	"log"
	"os"

	"github.com/A7reus/Cordelia/internal/identity"
)

func main() {
	id, err := identity.Load()
	if errors.Is(err, os.ErrNotExist) {
		id, err = createIdentity()
	}
	if err != nil {
		log.Fatal("cordelia:", err)
	}
	log.Printf("Running as %s [%s]\n", id.Name, id.Fingerprint)
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
