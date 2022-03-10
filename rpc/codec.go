package rpc

import (
	"io"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
)

type protoCodec struct {
	conn         *websocket.Conn
	closedCh      chan struct{}
}

func (p *protoCodec) closed() <- chan struct{} {
	return p.closedCh
}

func (p *protoCodec) close() {
	select {
	case <-p.closedCh:
	default:
		close(p.closedCh)
		p.conn.Close()
	}
}

func (p *protoCodec) read(v proto.Message) error {
	_, r, err := p.conn.NextReader()
	if err != nil {
		return err
	}

	bs, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	err = proto.Unmarshal(bs, v)
	return err
}

func (p *protoCodec) write(v proto.Message) error {
	w, err := p.conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return err
	}

	bs, _ := proto.Marshal(v)
	_, err = w.Write(bs)
	if err != nil {
		return err
	}

	err = w.Close()
	return err
}

func (p *protoCodec) getConn() *websocket.Conn {
	return p.conn
}

func errorMessage(err error) *RespMsg {
	msg := &RespMsg{
	}
	ec, ok := err.(Error)
	errMsg := &Err{
		Code: defaultErrorCode,
		Msg:  err.Error(),
	}
	if ok {
		errMsg.Code = ec.ErrorCode()
	}
	msg.Body, _ = proto.Marshal(errMsg)
	return msg
}
