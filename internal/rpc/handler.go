package rpc

import (
	"context"
	. "storage-mining/internal/rpc/proto"

	"github.com/golang/protobuf/proto"
)

type handleWrapper func(id uint64, body []byte) *RespMsg
type Handler func(body []byte) (proto.Message, error)

func (s *Server) handle(msg *ReqMsg, c *SrvConn) {
	s.processMsg(func(ctx context.Context) {
		answer := s.callMethod(msg)
		c.codec.WriteMsg(ctx, answer)
	})
}

func (s *Server) processMsg(fn func(ctx context.Context)) {
	s.handleWg.Add(1)
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer s.handleWg.Done()
		defer cancel()
		fn(ctx)
	}()
}

func (s *Server) callMethod(msg *ReqMsg) *RespMsg {
	handler := s.router.lookup(msg.Service, msg.Method)
	if handler == nil {
		err := &methodNotFoundError{msg.Service + "." + msg.Method}
		answer := errorMessage(err)
		answer.Id = msg.Id
		return answer
	}
	answer := handler(msg.Id, msg.Body)
	return answer
}
