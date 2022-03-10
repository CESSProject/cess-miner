package rpc

import (
	"context"

	"storage-mining/log"

	"github.com/golang/protobuf/proto"
)

type SrvConn struct {
	srv   *Server
	codec *websocketCodec
}

func (c *SrvConn) readLoop() {
	defer c.codec.close()
	for {
		msg := ReqMsg{}
		err := c.codec.read(&msg)
		if _, ok := err.(*proto.ParseError); ok {
			c.codec.WriteMsg(context.Background(), errorMessage(&parseError{err.Error()}))
			continue
		}

		if err != nil {
			log.Debug("server RPC connection read error ", err)
			c.codec.Close()
			break
		}

		c.srv.handle(&msg, c)
	}
}

type ClientConn struct {
	codec   *websocketCodec
	closeCh chan<- struct{}
}

func (c *ClientConn) readLoop(recv func(msg RespMsg)) {
	for {
		msg := RespMsg{}
		err := c.codec.read(&msg)
		if _, ok := err.(*proto.ParseError); ok {
			continue
		}

		if err != nil {
			log.Debug("client RPC connection read error ", err)
			c.closeCh <- struct{}{}
			break
		}

		recv(msg)
	}
}
