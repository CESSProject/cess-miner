package node

import (
	"encoding/json"
	"sync"

	"github.com/CESSProject/cess-bucket/configs"
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
)

const (
	FileType_file   uint8 = 1
	FileType_filler uint8 = 2
)

type Status byte

const (
	Status_Ok Status = iota
	Status_Err
)

type Message struct {
	Pubkey   []byte  `json:"pubkey"`
	SignMsg  []byte  `json:"signmsg"`
	Sign     []byte  `json:"sign"`
	Bytes    []byte  `json:"bytes"`
	FileName string  `json:"filename"`
	FileHash string  `json:"filehash"`
	FileSize uint64  `json:"filesize"`
	MsgType  MsgType `json:"msgtype"`
	LastMark bool    `json:"lastmark"`
	FileType uint8   `json:"filetype"`
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

	sendBufPool = sync.Pool{
		New: func() any {
			return make([]byte, configs.TCP_SendBuffer)
		},
	}

	readBufPool = sync.Pool{
		New: func() any {
			return make([]byte, configs.TCP_ReadBuffer)
		},
	}
)

func (m *Message) reset() {
	m.MsgType = MsgInvalid
	m.FileName = ""
	m.FileHash = ""
	m.FileSize = 0
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

func NewHeadMsg(fileName string, fid string, pkey, signmsg, sign []byte) *Message {
	m := msgPool.Get().(*Message)
	m.MsgType = MsgHead
	m.FileName = fileName
	m.FileHash = fid
	m.Pubkey = pkey
	m.SignMsg = signmsg
	m.Sign = sign
	return m
}

func NewFileMsg(fileName string, num int, buf []byte) *Message {
	m := &Message{}
	m.MsgType = MsgFile
	m.FileType = FileType_file
	m.FileName = fileName
	m.FileHash = ""
	m.FileSize = uint64(num)
	m.LastMark = false
	m.Pubkey = nil
	m.SignMsg = nil
	m.Sign = nil
	m.Bytes = sendBufPool.Get().([]byte)
	copy(m.Bytes, buf)
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
