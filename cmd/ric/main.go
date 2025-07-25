package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/oran/near-rt-ric-new/internal/a1"
	"github.com/oran/near-rt-ric-new/internal/e2"
	"github.com/oran/near-rt-ric-new/internal/o1"
)

func main() {
	log.Println("Starting O-RAN Near-RT RIC...")

	// Initialize and start the A1 interface server
	a1Server := a1.NewServer("0.0.0.0:8080")
	go func() {
		log.Println("A1 interface is listening on :8080")
		if err := a1Server.Start(); err != nil {
			log.Fatalf("Failed to start A1 server: %v", err)
		}
	}()

	// Initialize and start the E2 interface server
	e2Server := e2.NewServer("0.0.0.0:38484")
	go func() {
		log.Println("E2 interface is listening on :38484")
		if err := e2Server.Start(); err != nil {
			log.Fatalf("Failed to start E2 server: %v", err)
		}
	}()

	// Initialize and start the O1 interface server
	o1Server := o1.NewServer("0.0.0.0:830")
	go func() {
		log.Println("O1 interface is listening on :830")
		if err := o1Server.Start(); err != nil {
			log.Fatalf("Failed to start O1 server: %v", err)
		}
	}()

	log.Println("O-RAN Near-RT RIC is running. Press Ctrl+C to exit.")

	// Wait for a termination signal to gracefully shut down.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down O-RAN Near-RT RIC...")
}