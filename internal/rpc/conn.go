package rpc

import (
	"context"

	"storage-mining/internal/logger"
	. "storage-mining/internal/rpc/proto"

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
			logger.Warn.Sugar().Warnf("RPC service connection read err:%v", err)
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
			logger.Warn.Sugar().Warnf("RPC client service connection read err:%v", err)
			c.closeCh <- struct{}{}
			break
		}

		recv(msg)
	}
}
