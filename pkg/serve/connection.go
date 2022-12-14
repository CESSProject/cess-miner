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
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
)

// Connection
type Connection struct {
	// hosting server
	TCPServer IServer
	// socket TCP
	Conn *net.TCPConn
	// connection ID, globally unique
	ConnID uint32
	// message processing module
	MsgHandler IMsgHandle
	// context
	ctx context.Context
	// context.CancelFunc
	cancel context.CancelFunc
	// with buffer pipe
	msgBuffChan chan []byte
	// read-write lock
	sync.RWMutex
	// whether to close
	isClosed bool
}

// NewConnection
func NewConnection(server IServer, conn *net.TCPConn, connID uint32, msgHandler IMsgHandle) *Connection {
	// init Conn
	c := &Connection{
		TCPServer:   server,
		Conn:        conn,
		ConnID:      connID,
		isClosed:    false,
		MsgHandler:  msgHandler,
		msgBuffChan: make(chan []byte, configs.TCP_Message_Read_Buffers),
	}

	c.TCPServer.GetConnMgr().Add(c)
	return c
}

// StartWriter is the Goroutine used to write messages back to the client
func (c *Connection) StartWriter() {
	fmt.Println("[Writer Goroutine is running]")
	defer fmt.Println(c.RemoteAddr().String(), "[conn Writer exit!]")

	for {
		select {
		case data, ok := <-c.msgBuffChan:
			if ok {
				if _, err := c.Conn.Write(data); err != nil {
					fmt.Println("Send Buff Data error:, ", err, " Conn Writer exit")
					return
				}
			} else {
				fmt.Println("msgBuffChan is Closed")
				break
			}
		case <-c.ctx.Done():
			return
		}
	}
}

// StartReader is the Goroutine used to read client messages
func (c *Connection) StartReader() {
	fmt.Println("[Reader Goroutine is running]")
	defer fmt.Println(c.RemoteAddr().String(), "[conn Reader exit!]")
	defer c.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			headData := make([]byte, c.TCPServer.Packet().GetHeadLen())
			if _, err := io.ReadFull(c.Conn, headData); err != nil {
				fmt.Println("read msg head error ", err)
				return
			}

			msg, err := c.TCPServer.Packet().Unpack(headData)
			if err != nil {
				fmt.Println("unpack error ", err)
				return
			}

			var data []byte
			if msg.GetDataLen() > 0 {
				data = make([]byte, msg.GetDataLen())
				if _, err := io.ReadFull(c.Conn, data); err != nil {
					fmt.Println("read msg data error ", err)
					return
				}
			}
			msg.SetData(data)

			req := Request{
				conn:  c,
				msg:   msg,
				index: 0,
			}

			go c.MsgHandler.DoMsgHandler(c.cancel, &req)
		}
	}
}

// Start starts the connection and lets it work
func (c *Connection) Start() {
	c.ctx, c.cancel = context.WithCancel(context.Background())

	go c.StartReader()
	go c.StartWriter()

	select {
	case <-c.ctx.Done():
		c.finalizer()
		return
	}
}

// Stop Stop the connection and end the connection state
func (c *Connection) Stop() {
	c.cancel()
}

// GetTCPConnection Get the original socket TCPConn from the connection
func (c *Connection) GetTCPConnection() *net.TCPConn {
	return c.Conn
}

// GetConnID Get the connection ID
func (c *Connection) GetConnID() uint32 {
	return c.ConnID
}

// RemoteAddr Get remote client address information
func (c *Connection) RemoteAddr() net.Addr {
	return c.Conn.RemoteAddr()
}

// SendMsg sends the message data to the remote client
func (c *Connection) SendMsg(msgID uint32, data []byte) error {
	c.RLock()
	defer c.RUnlock()
	if c.isClosed == true {
		return errors.New("connection closed when send msg")
	}

	dp := c.TCPServer.Packet()
	msg, err := dp.Pack(NewMsgPackage(msgID, data))
	if err != nil {
		fmt.Println("Pack error msg ID = ", msgID)
		return errors.New("Pack error msg ")
	}

	_, err = c.Conn.Write(msg)
	return err
}

// SendBuffMsg sends the message data to the buffer
func (c *Connection) SendBuffMsg(msgID uint32, data []byte) error {
	c.RLock()
	defer c.RUnlock()
	idleTimeout := time.NewTimer(5 * time.Millisecond)
	defer idleTimeout.Stop()

	if c.isClosed == true {
		return errors.New("Connection closed when send buff msg")
	}

	dp := c.TCPServer.Packet()
	msg, err := dp.Pack(NewMsgPackage(msgID, data))
	if err != nil {
		fmt.Println("Pack error msg ID = ", msgID)
		return errors.New("Pack error msg ")
	}

	select {
	case <-idleTimeout.C:
		return errors.New("send buff msg timeout")
	case c.msgBuffChan <- msg:
		return nil
	}
}

// Context returns a user-defined context
func (c *Connection) Context() context.Context {
	return c.ctx
}

// Finalizer does the final cleaning work
func (c *Connection) finalizer() {
	c.Lock()
	defer c.Unlock()

	if c.isClosed == true {
		return
	}

	fmt.Println("Conn Stop()...ConnID = ", c.ConnID)

	_ = c.Conn.Close()
	c.TCPServer.GetConnMgr().Remove(c)
	close(c.msgBuffChan)
	c.isClosed = true
}
