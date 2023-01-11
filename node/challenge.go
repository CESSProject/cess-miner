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
	"strings"
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
	var (
		exist          bool
		chalKey        string
		chal           ChalResponse
		fillers        []string
		files          []string
		autonomousFils []string
	)
	n.Logs.Chlg("info", fmt.Errorf(">>>>> Start task_challenge <<<<<"))
	time.Sleep(configs.BlockInterval)
	for {
		minerInfo, err := n.Chn.GetMinerInfo(n.Chn.GetPublicKey())
		if string(minerInfo.State) != chain.MINER_STATE_POSITIVE && string(minerInfo.State) != chain.MINER_STATE_FROZEN {
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

		chalKey = fmt.Sprintf("%s%d", Chal_Blockheight, challenge.Start)
		exist, _ = n.Cach.Has([]byte(chalKey))
		if exist {
			time.Sleep(time.Minute)
			continue
		}

		n.Logs.Chlg("info", fmt.Errorf("challenge time: %v ~ %v", challenge.Start, challenge.Deadline))

		fillers = n.GetChallengefiles(int64(challenge.Start), n.FillerDir)
		files = n.GetChallengefiles(int64(challenge.Start), n.FileDir)
		autonomousFils = n.GetChallengeAutonomousFiles(int64(challenge.Start))
		if len(fillers) > 0 || len(files) > 0 || len(autonomousFils) > 0 {
			// 1.start sgx chal time and get QElement
			LockChallengeLock()
			err = GetChallengeReq(configs.ChallengeBlocks, n.Cfile.GetSgxPortNum(), configs.URL_GetChal, configs.URL_GetChal_Callback, n.Cfile.GetServiceAddr(), challenge.Random)
			if err != nil {
				ReleaseChallengeLock()
				n.Logs.Chlg("err", err)
				time.Sleep(configs.BlockInterval)
				continue
			}

			timeout := time.NewTicker(configs.TimeOut_WaitChallenge)
			defer timeout.Stop()
			select {
			case <-timeout.C:
				n.Logs.Chlg("err", fmt.Errorf("Wait challenge timeout"))
			case chal = <-Ch_Challenge:
			}

			if chal.Status.StatusCode != configs.SgxReportSuc {
				n.Logs.Chlg("err", fmt.Errorf("Recv challenge status code: %v", chal.Status.StatusCode))
				ReleaseChallengeLock()
				continue
			}
			n.Logs.Chlg("info", fmt.Errorf("Get challenge suc"))

			//2. challange all file
			n.challengeFiller(challenge, chal.QElement, fillers)
			n.challengeService(challenge, chal.QElement, files)
			n.challengeAutonomous(challenge, chal.QElement, autonomousFils)
			ReleaseChallengeLock()
			n.Cach.Put([]byte(chalKey), nil)
		} else {
			n.Logs.Chlg("info", fmt.Errorf("There is no file for this challenge: %v ~ %v", challenge.Start, challenge.Deadline))
			nowHeight, err := n.Chn.GetBlockHeight()
			if err != nil {
				time.Sleep(time.Minute)
			} else {
				if challenge.Deadline > nowHeight {
					time.Sleep(time.Second * time.Duration(3*(challenge.Deadline-nowHeight)))
				}
			}
			continue
		}
	}
}

func (n *Node) challengeFiller(challenge chain.NetworkSnapshot, qElement []pbc.QElement, fillers []string) {
	for i := 0; i < len(fillers); i++ {
		ftag, err := os.ReadFile(fillers[i] + ".tag")
		if err != nil {
			n.Logs.Chlg("err", errors.Wrapf(err, "[%v] [%d/%d]", filepath.Base(fillers[i]), challenge.Start, challenge.Deadline))
			continue
		}
		n.Logs.Chlg("info", fmt.Errorf("%v", fillers[i]))
		matrix, _, s, _ := pbc.SplitV2(fillers[i], configs.BlockSize, configs.SegmentSize)
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
		err = GetProofResultReq(configs.URL_GetProofResult, challenge.Random, Proof_Idle, data)
		if err != nil {
			n.Logs.Chlg("err", err)
		}
	}
}

func (n *Node) challengeService(challenge chain.NetworkSnapshot, qElement []pbc.QElement, files []string) {
	for i := 0; i < len(files); i++ {
		ftag, err := os.ReadFile(files[i] + ".tag")
		if err != nil {
			n.Logs.Chlg("err", errors.Wrapf(err, "[%v] [%d/%d]", filepath.Base(files[i]), challenge.Start, challenge.Deadline))
			continue
		}
		n.Logs.Chlg("info", fmt.Errorf("%v", files[i]))
		matrix, _, s, _ := pbc.SplitV2(files[i], configs.BlockSize, configs.SegmentSize)
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
		err = GetProofResultReq(configs.URL_GetProofResult, challenge.Random, Proof_Service, data)
		if err != nil {
			n.Logs.Chlg("err", err)
		}
	}
}

func (n *Node) challengeAutonomous(challenge chain.NetworkSnapshot, qElement []pbc.QElement, autonomousFils []string) {
	for i := 0; i < len(autonomousFils); i++ {
		ftag, err := os.ReadFile(autonomousFils[i] + ".tag")
		if err != nil {
			n.Logs.Chlg("err", errors.Wrapf(err, "[%v] [%d/%d]", filepath.Base(autonomousFils[i]), challenge.Start, challenge.Deadline))
			continue
		}
		n.Logs.Chlg("info", fmt.Errorf("%v", autonomousFils[i]))
		matrix, _, s, _ := pbc.SplitV2(autonomousFils[i], configs.BlockSize, configs.SegmentSize)
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
		err = GetProofResultReq(configs.URL_GetProofResult, challenge.Random, Proof_Autonomous, data)
		if err != nil {
			n.Logs.Chlg("err", err)
		}
	}
}

func (n *Node) GetChallengefiles(start int64, dir string) []string {
	var (
		recordBlock int64
		chalFillers = make([]string, 0)
	)
	files, err := utils.WorkFiles(dir)
	if err != nil {
		n.Logs.Space("err", err)
		return chalFillers
	}
	for i := 0; i < len(files); i++ {
		if len(filepath.Base(files[i])) == len(chain.FileHash{}) {
			val, err := n.Cach.Get([]byte(Cach_Blockheight + filepath.Base(files[i])))
			if err != nil {
				continue
			}
			recordBlock = utils.BytesToInt64(val)
			if recordBlock > start {
				continue
			}
			chalFillers = append(chalFillers, files[i])
		}
	}
	return chalFillers
}

func (n *Node) GetChallengeAutonomousFiles(start int64) []string {
	var (
		recordBlock         int64
		err                 error
		fname               string
		chalAutonomousFiles = make([]string, 0)
	)
	files, err := utils.WorkFiles(n.Cfile.GetAutonomousRegion())
	if err != nil {
		n.Logs.Space("err", err)
		return chalAutonomousFiles
	}

	for i := 0; i < len(files); i++ {
		fname = filepath.Base(files[i])
		if strings.Contains(fname, ".") {
			fname = strings.TrimSuffix(fname, filepath.Ext(fname))
		}
		val, err := n.Cach.Get([]byte(Cach_Blockheight + fname))
		if err != nil {
			continue
		}

		recordBlock = utils.BytesToInt64(val)
		if recordBlock > start {
			continue
		}
		chalAutonomousFiles = append(chalAutonomousFiles, files[i])
	}

	return chalAutonomousFiles
}
