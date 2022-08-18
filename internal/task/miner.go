package task

import (
	"cess-bucket/internal/chain"
	"cess-bucket/internal/pattern"
	"log"

	. "cess-bucket/internal/logger"
	"os"
	"time"
)

func task_self_judgment(ch chan bool) {
	defer func() {
		err := recover()
		if err != nil {
			Pnc.Sugar().Errorf("[panic]: %v", err)
		}
		ch <- true
	}()
	Out.Info(">>>>> Start task_self_judgment <<<<<")
	var failcount uint8
	minfo, err := chain.GetMinerInfo(nil)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	pattern.SetMinerState(string(minfo.State))
	for {
		minfo, err := chain.GetMinerInfo(nil)
		if err != nil {
			if err.Error() == chain.ERR_Empty {
				failcount++
			}
		} else {
			failcount = 0
			pattern.SetMinerState(string(minfo.State))
		}
		if failcount >= 10 {
			os.Exit(1)
		}
		time.Sleep(time.Minute * 5)
	}
}
