package rpc

import (
	"context"
	"sync"
	"sync/atomic"

	. "github.com/CESSProject/cess-bucket/internal/rpc/proto"
)

type ID uint32

type call struct {
	id ID
	ch chan<- *RespMsg
}

type Client struct {
	conn *ClientConn

	sync.Mutex
	pending   map[ID]call
	id        ID
	closeOnce sync.Once
	closeCh   <-chan struct{}
}

func newClient(codec *websocketCodec) *Client {
	ch := make(chan struct{})
	c := &ClientConn{
		codec:   codec,
		closeCh: ch,
	}
	client := &Client{
		closeCh: ch,
		conn:    c,
		pending: make(map[ID]call),
	}
	client.receive()
	client.dispatch()
	return client
}

func (c *Client) dispatch() {
	go func() {
		for {
			select {
			case <-c.closeCh:
				c.Close()
				return
			}
		}
	}()
}

func (c *Client) receive() {
	go c.conn.readLoop(func(msg RespMsg) {
		c.Lock()
		id := ID(msg.Id)
		ca, exist := c.pending[id]
		if exist {
			delete(c.pending, id)
		}
		c.Unlock()

		if exist {
			ca.ch <- &msg
		}
	})
}

func (c *Client) nextId() ID {
	n := atomic.AddUint32((*uint32)(&c.id), 1)
	return ID(n)
}

func (c *Client) Call(ctx context.Context, msg *ReqMsg) (*RespMsg, error) {
	ch := make(chan *RespMsg)
	ca := call{
		id: c.nextId(),
		ch: ch,
	}
	msg.Id = uint64(ca.id)

	c.Lock()
	c.pending[ca.id] = ca
	c.Unlock()

	err := c.conn.codec.WriteMsg(ctx, msg)
	if err != nil {
		return nil, err
	}

	select {
	case resp := <-ch:
		return resp, nil
	case <-ctx.Done():
		c.Lock()
		delete(c.pending, ca.id)
		c.Unlock()
		return nil, ctx.Err()
	}
}

func (c *Client) Close() {
	c.conn.codec.close()
	c.closeOnce.Do(func() {
		c.Lock()
		defer c.Unlock()
		for id, ca := range c.pending {
			close(ca.ch)
			delete(c.pending, id)
		}
	})
}
