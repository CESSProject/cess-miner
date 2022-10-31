package node

import (
	"encoding/json"
	"sync"
)

type MsgType byte

const (
	MsgInvalid MsgType = iota
	MsgHead
	MsgFile
	MsgEnd
	MsgNotify
	MsgClose
	MsgRecvHead
	MsgRecvFile
	MsgFillerHead
	MsgFiller
	MsgFillerEnd
)

type Status byte

const (
	Status_Ok Status = iota
	Status_Err
)

type Message struct {
	MsgType  MsgType `json:"msg_type"`
	FileName string  `json:"file_name"`
	FileHash string  `json:"file_hash"`
	FileSize uint64  `json:"file_size"`
	LastMark bool    `json:"last_mark"`
	Pubkey   []byte  `json:"pub_key"`
	SignMsg  []byte  `json:"sign_msg"`
	Sign     []byte  `json:"sign"`
	Bytes    []byte  `json:"bytes"`
}

type Notify struct {
	Status byte
}

var (
	msgPool = sync.Pool{
		New: func() any {
			return &Message{}
		},
	}

	BytesPool = sync.Pool{
		New: func() any {
			return make([]byte, 40*1024)
		},
	}
)

func (m *Message) GC() {
	if m.MsgType == MsgFile {
		BytesPool.Put(m.Bytes[:cap(m.Bytes)])
	}
	m.reset()
	msgPool.Put(m)
}

func (m *Message) reset() {
	m.MsgType = MsgInvalid
	m.FileName = ""
	m.FileHash = ""
	m.FileSize = 0
	m.LastMark = false
	m.Pubkey = nil
	m.SignMsg = nil
	m.Sign = nil
	m.Bytes = nil
}

func (m *Message) String() string {
	bytes, _ := json.Marshal(m)
	return string(bytes)
}

// Decode will convert from bytes
func Decode(b []byte) (m *Message, err error) {
	m = msgPool.Get().(*Message)
	err = json.Unmarshal(b, &m)
	return
}

func NewNotifyMsg(fileName string, status Status) *Message {
	m := msgPool.Get().(*Message)
	m.MsgType = MsgNotify
	m.Bytes = []byte{byte(status)}
	m.Bytes = append(m.Bytes, []byte(fileName)...)
	m.FileName = ""
	m.FileHash = ""
	m.Pubkey = nil
	m.SignMsg = nil
	m.Sign = nil
	return m
}

func NewHeadMsg(fileName string, fid string, lastmark bool, pkey, signmsg, sign []byte) *Message {
	m := msgPool.Get().(*Message)
	m.MsgType = MsgHead
	m.FileName = fileName
	m.FileHash = fid
	m.LastMark = lastmark
	m.Pubkey = pkey
	m.SignMsg = signmsg
	m.Sign = sign
	return m
}

func NewFillerHeadMsg(pkey, signmsg, sign []byte) *Message {
	m := msgPool.Get().(*Message)
	m.MsgType = MsgFillerHead
	m.Pubkey = pkey
	m.SignMsg = signmsg
	m.Sign = sign
	return m
}

func NewFillerMsg(fillerId string) *Message {
	m := msgPool.Get().(*Message)
	m.MsgType = MsgFiller
	m.FileName = fillerId
	m.Bytes = nil
	m.Pubkey = nil
	m.Sign = nil
	m.SignMsg = nil
	return m
}

func NewFileMsg(fileName string, buf []byte) *Message {
	m := msgPool.Get().(*Message)
	m.MsgType = MsgFile
	m.FileName = fileName
	m.Bytes = buf
	return m
}

func NewEndMsg(fileName string, fsize uint64) *Message {
	m := msgPool.Get().(*Message)
	m.MsgType = MsgEnd
	m.FileName = fileName
	m.FileSize = fsize
	return m
}

func NewCloseMsg(fileName string, status Status) *Message {
	m := msgPool.Get().(*Message)
	m.MsgType = MsgClose
	m.Bytes = []byte{byte(status)}
	m.FileName = fileName
	return m
}
