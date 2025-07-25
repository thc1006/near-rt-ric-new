package o1

import "log"

// Server holds the details for the O1 interface server.
type Server struct {
	addr string
}

// NewServer creates a new O1 server.
func NewServer(addr string) *Server {
	return &Server{
		addr: addr,
	}
}

// Start runs the O1 server.
func (s *Server) Start() error {
	log.Printf("O1 interface is listening on %s", s.addr)
	// In a real implementation, we would start a NETCONF server here.
	// For now, we just block forever.
	select {}
}