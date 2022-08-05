package chain

import (
	"sync"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
)

var (
	l   *sync.Mutex
	api *gsrpc.SubstrateAPI
)

func init() {
	l = new(sync.Mutex)
}

func GetRpcClient_Safe(rpcaddr string) (*gsrpc.SubstrateAPI, error) {
	var err error
	l.Lock()
	if api == nil {
		api, err = gsrpc.NewSubstrateAPI(rpcaddr)
		return api, err
	}
	id, err := healthchek(api)
	if id == 0 || err != nil {
		api, err = gsrpc.NewSubstrateAPI(rpcaddr)
		if err != nil {
			return nil, err
		}
	}
	return api, nil
}

func Free() {
	l.Unlock()
}

func NewRpcClient(rpcaddr string) (*gsrpc.SubstrateAPI, error) {
	return gsrpc.NewSubstrateAPI(rpcaddr)
}

// func substrateAPIKeepAlive() {
// 	var (
// 		err     error
// 		count_r uint8  = 0
// 		peer    uint64 = 0
// 	)

// 	for range time.Tick(time.Second * 25) {
// 		if count_r <= 1 {
// 			peer, err = healthchek(r)
// 			if err != nil || peer == 0 {
// 				count_r++
// 			}
// 		}
// 		if count_r > 1 {
// 			count_r = 2
// 			r, err = gsrpc.NewSubstrateAPI(C.RpcAddr)
// 			if err != nil {
// 				Err.Sugar().Errorf("%v", err)
// 			} else {
// 				count_r = 0
// 			}
// 		}
// 	}
// }

func healthchek(a *gsrpc.SubstrateAPI) (uint64, error) {
	defer func() {
		recover()
	}()
	h, err := a.RPC.System.Health()
	return uint64(h.Peers), err
}

// func getSubstrateAPI() *gsrpc.SubstrateAPI {
// 	wlock.Lock()
// 	return r
// }
// func releaseSubstrateAPI() {
// 	wlock.Unlock()
// }
