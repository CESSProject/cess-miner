/*
   Copyright 2022 CESS (Cumulus Encrypted Storage System) authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

        http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package serve

import (
	"errors"
	"fmt"
	"net"

	"github.com/CESSProject/cess-bucket/configs"
)

// Server
type Server struct {
	// Name of the server
	Name string
	// tcp4 or other
	IPVersion string
	// Service binding port IP
	IP string
	// Service binding port
	Port int
	// message Handler
	msgHandler IMsgHandle
	// Connection Manager
	ConnMgr IConnManager
	// Exit Channel
	exitChan chan struct{}
	// Data packet
	packet IDataPack
}

// NewServer
func NewServer(name, host string, port int) IServer {
	s := &Server{
		Name:       name,
		IPVersion:  "tcp4",
		IP:         host,
		Port:       port,
		msgHandler: NewMsgHandle(),
		ConnMgr:    NewConnManager(),
		exitChan:   nil,
		packet:     Factory().NewPack(DefaultDataPack),
	}
	return s
}

// Start
func (s *Server) Start() {
	fmt.Printf("[START] Server name: %s,listenner at IP: %s, Port %d is starting\n", s.Name, s.IP, s.Port)
	s.exitChan = make(chan struct{})

	// Linster Service
	go func() {
		addr, err := net.ResolveTCPAddr(s.IPVersion, fmt.Sprintf("%s:%d", s.IP, s.Port))
		if err != nil {
			fmt.Println("resolve tcp addr err: ", err)
			return
		}

		listener, err := net.ListenTCP(s.IPVersion, addr)
		if err != nil {
			panic(err)
		}

		fmt.Println("start server  ", s.Name, " suc, now listenning...")

		var cID uint32
		cID = 0

		go func() {
			for {
				// If the maximum connection is exceeded, wait
				if s.ConnMgr.Len() >= int(configs.MAX_TCP_CONNECTION) {
					fmt.Println("Exceeded the maxConnNum:", configs.MAX_TCP_CONNECTION, ", Wait:", AcceptDelay.duration)
					AcceptDelay.Delay()
					continue
				}

				conn, err := listener.AcceptTCP()
				if err != nil {
					// Go 1.16+
					if errors.Is(err, net.ErrClosed) {
						fmt.Println("Listener closed")
						return
					}
					fmt.Println("Accept err ", err)
					AcceptDelay.Delay()
					continue
				}

				AcceptDelay.Reset()

				dealConn := NewConnection(s, conn, cID, s.msgHandler)
				cID++

				go dealConn.Start()
			}
		}()

		select {
		case <-s.exitChan:
			err := listener.Close()
			if err != nil {
				fmt.Println("Listener close err ", err)
			}
		}
	}()
}

// Stop
func (s *Server) Stop() {
	fmt.Println("[STOP] server , name ", s.Name)

	s.ConnMgr.ClearConn()
	s.exitChan <- struct{}{}
	close(s.exitChan)
}

// Serve
func (s *Server) Serve() {
	s.Start()
	select {}
}

// AddRouter registers a routing service method for the current service to handle client connections
func (s *Server) AddRouter(msgID uint32, router IRouter) {
	s.msgHandler.AddRouter(msgID, router)
}

func (s *Server) GetConnMgr() IConnManager {
	return s.ConnMgr
}

func (s *Server) Packet() IDataPack {
	return s.packet
}
