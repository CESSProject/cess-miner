package chain

import (
	"fmt"
	"os"
	"storage-mining/configs"
	"storage-mining/internal/logger"
	"sync"
	"time"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
)

var (
	wlock *sync.Mutex
	r     *gsrpc.SubstrateAPI
)

func Chain_Init() {
	var (
		err error
	)
	r, err = gsrpc.NewSubstrateAPI(configs.Confile.CessChain.ChainAddr)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}
	wlock = new(sync.Mutex)
	go substrateAPIKeepAlive()
}

func substrateAPIKeepAlive() {
	var (
		err     error
		count_r uint8  = 0
		peer    uint64 = 0
	)

	for range time.Tick(time.Second * 25) {
		if count_r <= 1 {
			peer, err = healthchek(r)
			if err != nil || peer == 0 {
				count_r++
			}
		}
		if count_r > 1 {
			count_r = 2
			r, err = gsrpc.NewSubstrateAPI(configs.Confile.CessChain.ChainAddr)
			if err != nil {
				logger.ErrLogger.Sugar().Errorf("%v", err)
			} else {
				count_r = 0
			}
		}
	}
}

func healthchek(a *gsrpc.SubstrateAPI) (uint64, error) {
	defer func() {
		err := recover()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	h, err := a.RPC.System.Health()
	return uint64(h.Peers), err
}

func getSubstrateAPI() *gsrpc.SubstrateAPI {
	wlock.Lock()
	return r
}
func releaseSubstrateAPI() {
	wlock.Unlock()
}
