package serve

import (
	"sync"
	"time"
)

type blacklistMiner struct {
	Lock *sync.Mutex
	List map[uint64]int64
}

var BlackMiners *blacklistMiner

func init() {
	BlackMiners = &blacklistMiner{
		Lock: new(sync.Mutex),
		List: make(map[uint64]int64, 100),
	}
}

func (b *blacklistMiner) Add(peerid uint64) {
	b.Lock.Lock()
	b.List[peerid] = time.Now().Unix()
	b.Lock.Unlock()
}

func (b *blacklistMiner) Delete(peerid uint64) {
	b.Lock.Lock()
	delete(b.List, peerid)
	b.Lock.Unlock()
}

func (b *blacklistMiner) IsExist(peerid uint64) bool {
	b.Lock.Lock()
	defer b.Lock.Unlock()
	v, ok := b.List[peerid]
	if !ok {
		return false
	}
	if time.Since(time.Unix(v, 0)).Hours() > 3 {
		delete(b.List, peerid)
		return false
	}
	return true
}
