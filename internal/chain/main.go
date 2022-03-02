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
		logger.ErrLogger.Sugar().Errorf("%v", err)
		os.Exit(configs.Exit_Normal)
	}
	wlock = new(sync.Mutex)
	// api.c = make(chan bool, 1)
	// api.c <- true
	//go waitBlock(api.c)
	go substrateAPIKeepAlive()
	// mData, err := GetMinerDataOnChain(
	// 	configs.Confile.MinerData.TransactionPrK,
	// 	configs.ChainModule_Sminer,
	// 	configs.ChainModule_Sminer_MinerItems,
	// )
	// if err != nil {
	// 	fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
	// 	logger.ErrLogger.Sugar().Errorf("%v", err)
	// 	os.Exit(configs.Exit_Normal)
	// }

	// configs.MinerDataPath = fmt.Sprintf("Miner_C%v", mData.Peerid)
	// configs.MinerId_I = uint64(mData.Peerid)
	// configs.MinerId_S = fmt.Sprintf("C%v", mData.Peerid)
	// path := filepath.Join(configs.Confile.MinerData.MountedPath, configs.MinerDataPath)
	// configs.MinerDataPath = path

	// _, err = os.Stat(path)
	// if err != nil {
	// 	err = os.MkdirAll(path, os.ModePerm)
	// 	if err != nil {
	// 		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
	// 		logger.ErrLogger.Sugar().Errorf("%v", err)
	// 		os.Exit(configs.Exit_CreateFolder)
	// 	}
	// }

	// if configs.Confile.MinerData.MountedPath != "/" {
	// 	paths_mount := strings.Split(configs.Confile.MinerData.MountedPath, "/")
	// 	paths_dfs := strings.Split(configs.Confile.FileSystem.DfsInstallPath, "/")
	// 	if len(paths_dfs) < 2 || len(paths_mount) < 2 {
	// 		fmt.Printf("\x1b[%dm[err]\x1b[0m Your file service is not installed on the mount path.\n", 41)
	// 		logger.ErrLogger.Sugar().Errorf("Your file service [%v] is not installed on the mount path [%v].", configs.Confile.FileSystem.DfsInstallPath, configs.Confile.MinerData.MountedPath)
	// 		os.Exit(configs.Exit_CreateFolder)
	// 	}
	// 	if paths_mount[1] != paths_dfs[1] {
	// 		fmt.Printf("\x1b[%dm[err]\x1b[0m Your file service is not installed on the mount path.\n", 41)
	// 		logger.ErrLogger.Sugar().Errorf("Your file service [%v] is not installed on the mount path [%v].", configs.Confile.FileSystem.DfsInstallPath, configs.Confile.MinerData.MountedPath)
	// 		os.Exit(configs.Exit_CreateFolder)
	// 	}
	// }
	// dfscache := filepath.Join(configs.Confile.FileSystem.DfsInstallPath, "files", configs.Cache)
	// _, err = os.Stat(dfscache)
	// if err != nil {
	// 	err = os.MkdirAll(dfscache, os.ModePerm)
	// 	if err != nil {
	// 		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
	// 		logger.ErrLogger.Sugar().Errorf("%v", err)
	// 		os.Exit(configs.Exit_CreateFolder)
	// 	}
	// }
	//fmt.Printf("\x1b[%dm[ok]\x1b[0m Your data is stored in %v\n", 42, path)
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
			//fmt.Println(peer, err)
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

// func waitBlock(ch chan bool) {
// 	for {
// 		ch <- true
// 		time.Sleep(time.Second * 1)
// 	}
// }

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

// func SubstrateAPI_Read() *gsrpc.SubstrateAPI {
// 	return api.r
// }

func getSubstrateAPI() *gsrpc.SubstrateAPI {
	wlock.Lock()
	// for len(api.c) == 0 {
	// 	time.Sleep(time.Millisecond * 200)
	// }
	// <-api.c
	return r
}
func releaseSubstrateAPI() {
	wlock.Unlock()
}

func Chain_Main() {
	if configs.MinerEvent_Exit {
		if configs.MinerId_I == 0 {
			fmt.Printf("\x1b[%dm[note]\x1b[0m Unregistered miners cannot use the logout function\n", 43)
			os.Exit(configs.Exit_Normal)
		}
		//TODO:
		fmt.Printf("\x1b[%dm[note]\x1b[0m The logout function is under development...\n", 43)
		os.Exit(configs.Exit_Normal)
	}
	if configs.MinerEvent_RenewalTokens {
		if configs.MinerId_I == 0 {
			fmt.Printf("\x1b[%dm[note]\x1b[0m Unregistered miners cannot renewal tokens\n", 43)
			os.Exit(configs.Exit_Normal)
		}
		//TODO:
		fmt.Printf("\x1b[%dm[note]\x1b[0m The renewal function is under development...\n", 43)
		os.Exit(configs.Exit_Normal)
	}
}
