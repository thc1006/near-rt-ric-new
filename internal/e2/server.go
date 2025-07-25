package e2

import (
	"log"
	"net"
	"github.com/ishidawataru/sctp"
)

// Server holds the details for the E2 interface server.
type Server struct {
	addr string
}

// NewServer creates a new E2 server.
func NewServer(addr string) *Server {
	return &Server{
		addr: addr,
	}
}

// Start runs the E2 server.
func (s *Server) Start() error {
	addr, err := sctp.ResolveSCTPAddr("sctp", s.addr)
	if err != nil {
		return err
	}

	ln, err := sctp.ListenSCTP("sctp", addr)
	if err != nil {
		return err
	}
	defer ln.Close()

	log.Printf("E2 interface is listening on %s", ln.Addr())

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Failed to accept new E2 connection: %v", err)
			continue
		}

		go handleE2Connection(conn)
	}
}

func handleE2Connection(conn net.Conn) {
	defer conn.Close()
	log.Printf("Accepted new E2 connection from %s", conn.RemoteAddr())

	// In a real implementation, we would handle the E2AP messages here.
	// For now, we just log the connection and close it.
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			log.Printf("Failed to read from E2 connection: %v", err)
			return
		}
		log.Printf("Received %d bytes from E2 connection", n)
	}
}