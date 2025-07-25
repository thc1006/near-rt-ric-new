package a1

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

// Server holds the details for the A1 interface server.
type Server struct {
	engine *gin.Engine
	addr   string
}

// NewServer creates a new A1 server.
func NewServer(addr string) *Server {
	r := gin.Default()

	// Define a simple status route
	r.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "A1 interface is running",
		})
	})

	return &Server{
		engine: r,
		addr:   addr,
	}
}

// Start runs the A1 server.
func (s *Server) Start() error {
	return s.engine.Run(s.addr)
}