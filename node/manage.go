/*
   Copyright 2022 CESS (Cumulus Encrypted Storage System) authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

        http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package node

import (
	"fmt"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/chain"
	"github.com/CESSProject/cess-bucket/pkg/utils"
)

// The task_HandlingChallenges task will automatically help you complete file challenges.
// Apart from human influence, it ensures that you submit your certificates in a timely manner.
// It keeps running as a subtask.
func (n *Node) task_manage(ch chan bool) {
	var (
		txhash string
	)
	defer func() {
		if err := recover(); err != nil {
			n.Logs.Pnc(utils.RecoverError(err))
		}
		ch <- true
	}()
	n.Logs.Chlg("info", fmt.Errorf(">>>>> Start task_manage <<<<<"))
	time.Sleep(configs.BlockInterval)
	for {
		minerInfo, err := n.Chn.GetMinerInfo(n.Chn.GetPublicKey())
		if string(minerInfo.State) != chain.MINER_STATE_POSITIVE {
			time.Sleep(time.Minute)
			continue
		}

		challenge, err := n.Chn.GetChallenges()
		if err != nil {
			n.Logs.Chlg("err", err)
		}

		if challenge.Start <= 0 {
			time.Sleep(time.Minute)
			continue
		}

		n.Logs.Chlg("info", fmt.Errorf("challenge height: %v", challenge.Start))

		//Call sgx to generate sign
		var msg []byte
		var sign chain.Signature

		// proof up chain
		for {
			txhash, err = n.Chn.SubmitProofs(msg, sign)
			if txhash == "" {
				n.Logs.Chlg("err", fmt.Errorf("SubmitProofs fail: %v", err))
			} else {
				n.Logs.Chlg("info", fmt.Errorf("SubmitProofs suc: %v", txhash))
				break
			}
			time.Sleep(configs.BlockInterval)
		}
	}
}
