package pattern

import (
	"sync"
	"time"
)

type Blacklist struct {
	list map[string]int64
	l    *sync.Mutex
}

var blacklist *Blacklist

func init() {
	blacklist = new(Blacklist)
	blacklist.l = new(sync.Mutex)
	blacklist.list = make(map[string]int64, 10)
}

func AddToBlacklist(key string) {
	blacklist.l.Lock()
	blacklist.list[key] = time.Now().Unix()
	blacklist.l.Unlock()
}

func IsInBlacklist(key string) bool {
	blacklist.l.Lock()
	_, ok := blacklist.list[key]
	blacklist.l.Unlock()
	return ok
}

func DeleteExpiredBlacklist() {
	blacklist.l.Lock()
	for k, v := range blacklist.list {
		if time.Since(time.Unix(v, 0)).Minutes() > 60 {
			delete(blacklist.list, k)
		}
	}
	blacklist.l.Unlock()
}
