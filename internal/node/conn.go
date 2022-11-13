package node

type Server interface {
	Start()
}

type Client interface {
	SendFile(fid string, pkey, signmsg, sign []byte) error
	//RecvFiller(pkey, signmsg, sign []byte) error
}

type NetConn interface {
	HandlerLoop()
	GetMsg() (*Message, bool)
	SendMsg(m *Message)
	GetRemoteAddr() string
	Close() error
	IsClose() bool
}

type ConMgr struct {
	conn       NetConn
	fileDir    string
	fileName   string
	sendFiles  []string
	fillerId   string
	waitNotify chan bool
	stop       chan struct{}
}
