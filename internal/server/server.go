package server

import (
	"errors"
	"fmt"
	"log"
	"net/http"
)

type Server struct {
	Addr string
	Handler http.Handler
}

func NewServer(addr string, handler http.Handler) *Server {
	return &Server{
		Addr: addr,
		Handler: handler,
	}
}

func (s *Server) Run() {
	log.Println(fmt.Sprintf("Server running on http://localhost%s", s.Addr))
	if err := http.ListenAndServe(s.Addr, s.Handler); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}
}
