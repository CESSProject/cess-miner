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
	"io"
	"net/http"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/chain"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/gin-gonic/gin"
)

func (n *Node) GetResult(c *gin.Context) {
	var (
		err    error
		txhash string
		result ChallengeResult
		msg    ChallengeSignMessage
	)
	val, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, nil)
		return
	}

	err = json.Unmarshal(val, &result)
	if err != nil {
		n.Logs.Chlg("err", fmt.Errorf("UnMarshal result err: %v", err))
		c.JSON(http.StatusBadRequest, nil)
		return
	}
	c.JSON(http.StatusOK, nil)

	msg.AutonomousBloomFilter = result.AutonomousBloomFilter
	msg.AutonomousFailedFileHashes = result.AutonomousFailedFileHashes
	msg.ChalId = hex.EncodeToString(result.ChalId)
	msg.IdleBloomFilter = result.IdleBloomFilter
	msg.IdleFailedFileHashes = result.IdleFailedFileHashes
	msg.Pkey = hex.EncodeToString(result.Pkey)
	msg.ServiceBloomFilter = result.ServiceBloomFilter
	msg.ServiceFailedFileHashes = result.ServiceFailedFileHashes

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		n.Logs.Chlg("err", fmt.Errorf("Marshal msg err: %v", err))
		return
	}

	var report chain.ChallengeReport
	report.Message = msgBytes
	if len(report.Signature) != len(result.Sig) {
		n.Logs.Chlg("err", fmt.Errorf("report sig length err: %v", len(result.Sig)))
		return
	}

	for i := 0; i < len(result.Sig); i++ {
		report.Signature[i] = types.U8(result.Sig[i])
	}

	tryCount := 0
	for {
		txhash, err = n.Chn.SubmitChallengeReport(report)
		if err != nil {
			n.Logs.Chlg("err", err)
			tryCount++
			if tryCount > 5 {
				n.Logs.Chlg("err", fmt.Errorf("SubmitChallengeReport failed"))
				return
			}
			time.Sleep(configs.BlockInterval * 2)
			continue
		}
		n.Logs.Chlg("info", fmt.Errorf("SubmitChallengeReport suc: %v", txhash))
		break
	}
}
