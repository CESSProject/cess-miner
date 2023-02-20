package node

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"sync"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
)

type TcpCon struct {
	conn *net.TCPConn

	recv chan *Message
	send chan *Message

	onceStop *sync.Once
	stop     chan struct{}
}

var (
	HEAD_FILE   = []byte("c100")
	HEAD_FILLER = []byte("c101")
)

func NewTcp(conn *net.TCPConn) *TcpCon {
	return &TcpCon{
		conn:     conn,
		recv:     make(chan *Message, configs.TCP_Message_Read_Buffers),
		send:     make(chan *Message, configs.TCP_Message_Send_Buffers),
		onceStop: &sync.Once{},
		stop:     make(chan struct{}),
	}
}

func (t *TcpCon) HandlerLoop() {
	go t.readMsg()
	go t.sendMsg()
}

func (t *TcpCon) sendMsg() {
	sendBuf := readBufPool.Get().([]byte)
	defer func() {
		recover()
		t.Close()
		time.Sleep(time.Second)
		close(t.send)
		readBufPool.Put(sendBuf)
	}()

	copy(sendBuf[:len(HEAD_FILE)], HEAD_FILE)

	for !t.IsClose() {
		select {
		case m := <-t.send:
			data, err := json.Marshal(m)
			if err != nil {
				return
			}

			switch cap(m.Bytes) {
			case configs.TCP_SendBuffer:
				sendBufPool.Put(m.Bytes)
			default:
			}

			binary.BigEndian.PutUint32(sendBuf[len(HEAD_FILE):len(HEAD_FILE)+4], uint32(len(data)))
			copy(sendBuf[len(HEAD_FILE)+4:], data)

			_, err = t.conn.Write(sendBuf[:len(HEAD_FILLER)+4+len(data)])
			if err != nil {
				return
			}
		default:
			time.Sleep(configs.TCP_Message_Interval)
		}
	}
}

func (t *TcpCon) readMsg() {
	var (
		err      error
		n        int
		flag     bool = true
		waittime int64
		header   = make([]byte, 4)
	)
	readBuf := readBufPool.Get().([]byte)
	defer func() {
		recover()
		t.Close()
		close(t.recv)
		readBufPool.Put(readBuf)
	}()

	for !t.IsClose() {
		if !flag {
			// read until we get 4 bytes for the magic
			_, err = io.ReadAtLeast(t.conn, header, 4)
			if err != nil {
				if err != io.EOF {
					return
				} else {
					waittime++
					if waittime >= 10 {
						return
					}
					time.Sleep(time.Second)
					continue
				}
			}

			if !bytes.Equal(header, HEAD_FILLER) && !bytes.Equal(header, HEAD_FILE) {
				return
			}
		}
		flag = false
		// read until we get 4 bytes for the header
		_, err = io.ReadAtLeast(t.conn, header, 4)
		if err != nil {
			if err != io.EOF {
				return
			}
			continue
		}

		// data size
		msgSize := binary.BigEndian.Uint32(header)
		if msgSize > configs.TCP_ReadBuffer {
			return
		}

		// read data
		n, err = io.ReadFull(t.conn, readBuf[:msgSize])
		if err != nil {
			return
		}
		m := &Message{}
		m.Bytes = readBufPool.Get().([]byte)

		err = json.Unmarshal(readBuf[:n], &m)
		if err != nil {
			return
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
		close(t.stop)
	})
	return nil
}

func (t *TcpCon) IsClose() bool {
	select {
	case <-t.stop:
		return true
	default:
		return false
	}
}

var _ = NetConn(&TcpCon{})
