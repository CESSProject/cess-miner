package node

import (
	"log"
	"runtime"

	"github.com/CESSProject/cess-bucket/internal/chain"
	"github.com/CESSProject/cess-bucket/internal/pattern"

	"os"
	"time"

	. "github.com/CESSProject/cess-bucket/internal/logger"
)

func (node *Node) task_self_judgment(ch chan bool) {
	defer func() {
		err := recover()
		if err != nil {
			Pnc.Sugar().Errorf("[panic]: %v", err)
		}
		ch <- true
	}()
	Out.Info(">>>>> Start task_self_judgment <<<<<")
	var failcount uint8
	var count uint8
	minfo, err := chain.GetMinerInfo(nil)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	pattern.SetMinerState(string(minfo.State))

	for {
		time.Sleep(time.Minute)
		runtime.GC()
		count++
		if count >= 5 {
			count = 0
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
		}
	}
}
