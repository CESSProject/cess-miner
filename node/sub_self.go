/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

func (node *Node) task_self_judgment(ch chan bool) {
	defer func() {
		err := recover()
		if err != nil {
			//Pnc.Sugar().Errorf("[panic]: %v", err)
		}
		ch <- true
	}()
	// Out.Info(">>>>> Start task_self_judgment <<<<<")
	// var failcount uint8
	// var count uint8
	// var clearMemNum uint8
	// minfo, err := chain.GetMinerInfo(nil)
	// if err != nil {
	// 	log.Println(err)
	// 	os.Exit(1)
	// }
	// pattern.SetMinerState(string(minfo.State))

	// for {
	// 	time.Sleep(time.Minute)
	// 	runtime.GC()
	// 	count++
	// 	clearMemNum++
	// 	if count >= 5 {
	// 		count = 0
	// 		minfo, err := chain.GetMinerInfo(nil)
	// 		if err != nil {
	// 			if err.Error() == chain.ERR_Empty {
	// 				failcount++
	// 			}
	// 		} else {
	// 			failcount = 0
	// 			pattern.SetMinerState(string(minfo.State))
	// 		}
	// 		if failcount >= 10 {
	// 			os.Exit(1)
	// 		}
	// 	}
	// 	if clearMemNum >= 200 {
	// 		clearMemNum = 0
	// 		tools.ClearMemBuf()
	// 	}
	// }
}
