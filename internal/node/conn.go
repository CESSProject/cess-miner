package node

type Server interface {
	Start()
}

type Client interface {
	SendFile(fid string, pkey, signmsg, sign []byte) error
}

type NetConn interface {
	HandlerLoop()
	GetMsg() (*Message, bool)
	SendMsg(m *Message)
	Close() error
	IsClose() bool
}

type ConMgr struct {
	conn       NetConn
	fileDir    string
	fileName   string
	sendFiles  []string
	waitNotify chan bool
	stop       chan struct{}
}
