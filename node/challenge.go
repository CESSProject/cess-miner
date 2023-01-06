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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/chain"
	"github.com/CESSProject/cess-bucket/pkg/pbc"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/pkg/errors"
)

// The task_challenge is used to complete the challenge.
// It keeps running as a subtask.
func (n *Node) task_challenge(ch chan<- bool) {
	defer func() {
		if err := recover(); err != nil {
			n.Logs.Pnc(utils.RecoverError(err))
		}
		ch <- true
	}()
	n.Logs.Chlg("info", fmt.Errorf(">>>>> Start task_challenge <<<<<"))
	time.Sleep(configs.BlockInterval)
	for {
		minerInfo, err := n.Chn.GetMinerInfo(n.Chn.GetPublicKey())
		if string(minerInfo.State) != chain.MINER_STATE_POSITIVE {
			n.Logs.Chlg("err", fmt.Errorf("miner state: %v", string(minerInfo.State)))
			time.Sleep(time.Minute)
			continue
		}

		challenge, err := n.Chn.GetChallenges()
		if err != nil {
			if err.Error() != chain.ERR_Empty {
				n.Logs.Chlg("err", err)
			}
			time.Sleep(time.Minute)
			continue
		}

		if challenge.Start <= 0 || challenge.Deadline <= 0 {
			time.Sleep(time.Minute)
			continue
		}

		n.Logs.Chlg("info", fmt.Errorf("challenge time: %v ~ %v", challenge.Start, challenge.Deadline))

		//1. start sgx chal time and get QElement
		qElement, err := GetChallengeReq(configs.ChallengeBlocks, n.Cfile.GetSgxPortNum(), configs.URL_GetChal, configs.URL_GetReport_Callback, n.Cfile.GetServiceAddr(), challenge.Random)
		if err != nil {
			n.Logs.Chlg("err", err)
			continue
		}
		//2. challange all file
		n.challengeFiller(challenge, qElement)
		n.challengeService(challenge, qElement)
		n.challengeAutonomous(challenge, qElement)
	}
}

func (n *Node) challengeFiller(challenge chain.NetworkSnapshot, qElement []pbc.QElement) {
	fillers, _ := utils.WorkFiles(n.FillerDir)
	for i := 0; i < len(fillers); i++ {
		if len(filepath.Base(fillers[i])) == len(chain.FileHash{}) {
			val, err := n.Cach.Get([]byte(Cach_Blockheight + filepath.Base(fillers[i])))
			if err != nil {
				continue
			}
			recordBlock := utils.BytesToInt64(val)
			if recordBlock > int64(challenge.Start) {
				continue
			}
			matrix, _, s, _ := pbc.SplitV2(fillers[i], configs.BlockSize, configs.SegmentSize)
			ftag, err := os.ReadFile(fillers[i] + ".tag")
			if err != nil {
				n.Logs.Chlg("err", errors.Wrapf(err, "[%v] [%d/%d]", filepath.Base(fillers[i]), challenge.Start, recordBlock))
				continue
			}
			var tag chain.Result
			err = json.Unmarshal(ftag, &tag)
			if err != nil {
				n.Logs.Chlg("err", err)
				continue
			}
			var sigmas = make([][]byte, len(tag.Sigmas))
			for j := 0; j < len(tag.Sigmas); j++ {
				sigmas[j], _ = hex.DecodeString(tag.Sigmas[j])
			}
			sigma, miu := pbc.GenProof(sigmas, qElement, matrix, s, configs.SegmentSize)
			var proof MinerProof
			proof.Sigma = sigma
			proof.Miu = miu
			proof.Tag = tag.Tag
			data, err := json.Marshal(&proof)
			if err != nil {
				n.Logs.Chlg("err", err)
				continue
			}
			err = GetProofResultReq(n.Cfile.GetSgxPortNum(), configs.URL_GetProofResult, configs.URL_GetProofResult_Callback, n.Cfile.GetServiceAddr(), challenge.Random, Proof_Idle, data)
			if err != nil {
				n.Logs.Chlg("err", err)
				continue
			}

			var proofResult ChalResponse
			timeout := time.NewTicker(configs.TimeOut_WaitProofResult)
			defer timeout.Stop()
			select {
			case <-timeout.C:
				n.Logs.Space("err", fmt.Errorf("Wait Proof Result timeout"))
			case proofResult = <-Ch_ProofResult:
				if proofResult.Status.StatusCode != configs.SgxReportSuc {
					n.Logs.Space("err", fmt.Errorf("Recv Proof Result status code: %v", tag.Status.StatusCode))
				}
			}
		}
	}
}

func (n *Node) challengeService(challenge chain.NetworkSnapshot, qElement []pbc.QElement) {

}

func (n *Node) challengeAutonomous(challenge chain.NetworkSnapshot, qElement []pbc.QElement) {

}
