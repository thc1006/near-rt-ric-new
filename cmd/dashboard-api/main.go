/*
SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/oran/near-rt-ric-new/pkg/dashboard"
)

var log = logging.GetLogger("dashboard-api")

func main() {
	var (
		port           = flag.Int("port", 8080, "Port to run the dashboard API server on")
		e2mgrEndpoint  = flag.String("e2mgr-endpoint", "localhost:3800", "E2 Manager gRPC endpoint")
		submgrEndpoint = flag.String("submgr-endpoint", "localhost:3801", "Subscription Manager gRPC endpoint")
		appmgrEndpoint = flag.String("appmgr-endpoint", "localhost:8080", "App Manager REST endpoint")
	)
	flag.Parse()

	// Setup logging
	logging.SetLevel(logging.InfoLevel)
	log.Info("Starting Dashboard API Gateway")

	// Create dashboard server
	config := &dashboard.Config{
		Port:           *port,
		E2MgrEndpoint:  *e2mgrEndpoint,
		SubmgrEndpoint: *submgrEndpoint,
		AppmgrEndpoint: *appmgrEndpoint,
	}

	server, err := dashboard.NewServer(config)
	if err != nil {
		log.Fatalf("Failed to create dashboard server: %v", err)
	}

	// Start server in a goroutine
	go func() {
		log.Infof("Dashboard API server starting on port %d", *port)
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down Dashboard API server...")

	// Give outstanding requests a deadline for completion
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Errorf("Server forced to shutdown: %v", err)
	}

	log.Info("Dashboard API server exited")
}
