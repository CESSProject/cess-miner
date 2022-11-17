package node

import (
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
	sendBufPool = &sync.Pool{
		New: func() any {
			return make([]byte, configs.TCP_SendBuffer)
		},
	}

	readBufPool = &sync.Pool{
		New: func() any {
			return make([]byte, configs.TCP_ReadBuffer)
		},
	}
)

func NewNotifyMsg(fileName string, status Status) *Message {
	m := &Message{}
	m.MsgType = MsgNotify
	m.Bytes = []byte{byte(status)}
	m.FileName = ""
	m.FileHash = ""
	m.FileSize = 0
	m.LastMark = false
	m.FileType = FileType_file
	m.Pubkey = nil
	m.SignMsg = nil
	m.Sign = nil
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
	m := &Message{}
	m.MsgType = MsgEnd
	m.FileName = fileName
	m.FileSize = fsize
	m.FileHash = ""
	m.FileType = FileType_file
	m.LastMark = false
	m.Pubkey = nil
	m.SignMsg = nil
	m.Sign = nil
	m.Bytes = nil
	return m
}

func NewCloseMsg(fileName string, status Status) *Message {
	m := &Message{}
	m.MsgType = MsgClose
	m.Bytes = []byte{byte(status)}
	m.FileName = ""
	m.FileHash = ""
	m.FileSize = 0
	m.LastMark = false
	m.FileType = FileType_file
	m.Pubkey = nil
	m.SignMsg = nil
	m.Sign = nil
	return m
}
