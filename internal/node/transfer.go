package node

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"runtime"
	"sync"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
)

type TcpCon struct {
	conn *net.TCPConn

	recv chan *Message
	send chan *Message

	onceStop *sync.Once
	lock     *sync.RWMutex
	stop     *bool
}

var (
	MAGIC_BYTES = []byte("cess")
	EmErr       = fmt.Errorf("dont have msg")
)

func NewTcp(conn *net.TCPConn) *TcpCon {
	return &TcpCon{
		conn:     conn,
		recv:     make(chan *Message, configs.TCP_Message_Read_Buffers),
		send:     make(chan *Message, configs.TCP_Message_Send_Buffers),
		onceStop: &sync.Once{},
		lock:     new(sync.RWMutex),
		stop:     new(bool),
	}
}

func (t *TcpCon) HandlerLoop() {
	go t.readMsg()
	go t.sendMsg()
}

func (t *TcpCon) sendMsg() {
	defer func() {
		t.Close()
		time.Sleep(time.Second)
		close(t.send)
	}()
	for !t.IsClose() {
		select {
		case m := <-t.send:
			data, err := json.Marshal(m)
			if err != nil {
				return
			}

			head := make([]byte, len(MAGIC_BYTES)+4+len(data), len(MAGIC_BYTES)+4+len(data))
			copy(head[:len(MAGIC_BYTES)], MAGIC_BYTES)
			binary.BigEndian.PutUint32(head[len(MAGIC_BYTES):len(MAGIC_BYTES)+4], uint32(len(data)))
			copy(head[len(MAGIC_BYTES)+4:], data)

			_, err = t.conn.Write(head)
			if err != nil {
				return
			}
		default:
			time.Sleep(configs.TCP_Message_Interval)

		}
	}
}

func (t *TcpCon) readMsg() {
	defer func() {
		t.Close()
		close(t.recv)
	}()
	var (
		err    error
		n      int
		header = make([]byte, 4)
		buf    = make([]byte, configs.TCP_ReadBuffer)
	)

	for !t.IsClose() {
		// read until we get 4 bytes for the magic
		_, err = io.ReadFull(t.conn, header)
		if err != nil {
			if err != io.EOF {
				runtime.Goexit()
			}
			continue
		}

		if !bytes.Equal(header, MAGIC_BYTES) {
			runtime.Goexit()
		}

		// read until we get 4 bytes for the header
		_, err = io.ReadFull(t.conn, header)
		if err != nil {
			runtime.Goexit()
		}

		// // data size
		msgSize := binary.BigEndian.Uint32(header)

		n, err = io.ReadFull(t.conn, buf[:msgSize])
		if err != nil {
			err = fmt.Errorf("initial read error: %v \n", err)
			return
		}

		m := &Message{}
		err = json.Unmarshal(buf[:n], &m)
		if err != nil {
			runtime.Goexit()
		}

		t.recv <- m
	}
}

func (t *TcpCon) GetMsg() (*Message, bool) {
	timer := time.NewTimer(configs.TCP_Time_WaitNotification)
	defer timer.Stop()
	select {
	case m, ok := <-t.recv:
		return m, ok
	case <-timer.C:
		return nil, true
	}
}

func (t *TcpCon) SendMsg(m *Message) {
	t.send <- m
}

func (t *TcpCon) GetRemoteAddr() string {
	return t.conn.RemoteAddr().String()
}

func (t *TcpCon) Close() error {
	t.onceStop.Do(func() {
		t.conn.Close()
		t.lock.Lock()
		*t.stop = true
		t.lock.Unlock()
	})
	return nil
}

func (t *TcpCon) IsClose() bool {
	var ok bool
	t.lock.RLock()
	ok = *t.stop
	t.lock.RUnlock()
	return ok
}

var _ = NetConn(&TcpCon{})
