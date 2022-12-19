package serve

// import (
// 	"sync"
// 	"time"
// )

// type globalTokens struct {
// 	lock   *sync.RWMutex
// 	tokens map[string]int64
// }

// var Tokens *globalTokens

// func init() {
// 	Tokens = &globalTokens{
// 		lock:   new(sync.RWMutex),
// 		tokens: make(map[string]int64, 10),
// 	}
// }

// func (t *globalTokens) Add(key string) {
// 	Tokens.lock.Lock()
// 	Tokens.tokens[key] = time.Now().Unix()
// 	Tokens.lock.Unlock()
// }

// func (t *globalTokens) IsExit(key string) bool {
// 	Tokens.lock.RLock()
// 	defer Tokens.lock.RUnlock()
// 	_, ok := Tokens.tokens[key]
// 	return ok
// }

// func (t *globalTokens) Update(key string) bool {
// 	Tokens.lock.RLock()
// 	defer Tokens.lock.RUnlock()
// 	_, ok := Tokens.tokens[key]
// 	if ok {
// 		Tokens.tokens[key] = time.Now().Unix()
// 	}
// 	return ok
// }

// func AutoExpirationDelete() {
// 	tick := time.NewTicker(time.Minute * 5)
// 	for {
// 		select {
// 		case <-tick.C:
// 			Tokens.lock.Lock()
// 			for k, v := range Tokens.tokens {
// 				if time.Since(time.Unix(v, 0)).Minutes() > 10 {
// 					delete(Tokens.tokens, k)
// 				}
// 			}
// 			Tokens.lock.Unlock()
// 		}
// 	}
// }
