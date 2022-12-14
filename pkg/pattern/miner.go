package pattern

import (
	"sync"
)

const (
	M_Pending  = "pending"
	M_Positive = "positive"
	M_Frozen   = "frozen"
	M_Exit     = "exit"
)

type Miner struct {
	Acc        []byte
	SignAddr   string
	IncomeAddr string
	State      string
	RecentSche string
	l          *sync.Mutex
}

var m *Miner

func init() {
	m = new(Miner)
	m.State = M_Pending
	m.l = new(sync.Mutex)
}

func GetMiner() *Miner {
	return m
}

func SetMinerState(st string) {
	m.l.Lock()
	m.State = st
	m.l.Unlock()
}

func GetMinerState() string {
	m.l.Lock()
	defer m.l.Unlock()
	return m.State
}

func GetMinerAcc() []byte {
	m.l.Lock()
	defer m.l.Unlock()
	return m.Acc
}

func SetMinerAcc(acc []byte) {
	m.l.Lock()
	m.Acc = acc
	m.l.Unlock()
}

func GetMinerSignAddr() string {
	m.l.Lock()
	defer m.l.Unlock()
	return m.SignAddr
}

func SetMinerSignAddr(addr string) {
	m.l.Lock()
	m.SignAddr = addr
	m.l.Unlock()
}

func GetMinerIncomeAddr() string {
	m.l.Lock()
	defer m.l.Unlock()
	return m.IncomeAddr
}

func SetMinerIncomeAddr(addr string) {
	m.l.Lock()
	m.IncomeAddr = addr
	m.l.Unlock()
}

func GetMinerRecentSche() string {
	m.l.Lock()
	defer m.l.Unlock()
	return m.RecentSche
}

func SetMinerRecentSche(sche string) {
	m.l.Lock()
	m.RecentSche = sche
	m.l.Unlock()
}
