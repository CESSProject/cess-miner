package rpc

import (
	"sync"

	mapset "github.com/deckarep/golang-set"
)

type Server struct {
	running    int32
	conns     mapset.Set

	handleWg   sync.WaitGroup
	router     *serviceRouter
}

func NewServer() *Server {
	s := &Server{
		conns: mapset.NewSet(),
		router: newServiceRouter(),
	}
	return s
}

func (s *Server) serve(codec *websocketCodec) {
	c := &SrvConn{
		srv: s,
		codec: codec,
	}
	// Add the conn to the set so it can be closed by Stop.
	s.conns.Add(c)
	defer s.conns.Remove(c)

	go c.readLoop()
}

func (s *Server) Register(name string, service interface{}) error {
	return s.router.registerName( name, service)
}

func (s *Server) Close() {
	s.conns.Each(func(c interface{}) bool {
		c.(websocketCodec).close()
		return true
	})
	s.handleWg.Wait()
}
