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
		txhash string
		err    error
	)
	val, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, nil)
		return
	}

	fmt.Println("Get Result, len()==", len(val))
	var result ChallengeResult

	err = json.Unmarshal(val, &result)
	if err != nil {
		n.Logs.Chlg("err", fmt.Errorf("UnMarshal result err: %v", err))
		c.JSON(http.StatusOK, nil)
		return
	}

	msg := revert(result)
	var report chain.ChallengeReport
	report.Message = types.Bytes([]byte(msg))
	if len(report.Signature) != len(result.Sig) {
		n.Logs.Chlg("err", fmt.Errorf("report sig length err: %v", len(result.Sig)))
		c.JSON(http.StatusOK, nil)
		return
	}
	for i := 0; i < len(result.Sig); i++ {
		report.Signature[i] = types.U8(result.Sig[i])
	}

	for {
		txhash, err = n.Chn.SubmitChallengeReport(report)
		if err != nil {
			n.Logs.Chlg("err", err)
			time.Sleep(configs.BlockInterval)
			continue
		}
		n.Logs.Chlg("info", fmt.Errorf("SubmitChallengeReport suc: %v", txhash))
		break
	}

	c.JSON(http.StatusOK, nil)
	return
}

func revert(result ChallengeResult) string {
	abf_json, _ := json.Marshal(result.AutonomousBloomFilter)
	ibf_json, _ := json.Marshal(result.IdleBloomFilter)
	sbf_json, _ := json.Marshal(result.ServiceBloomFilter)
	autonomous_file_hashes_json, _ := json.Marshal(result.AutonomousFailedFileHashes)
	idle_file_hashes_json, _ := json.Marshal(result.IdleFailedFileHashes)
	service_failed_file_hashes, _ := json.Marshal(result.ServiceFailedFileHashes)
	chal_json, _ := json.Marshal(result.ChalId)

	message := string(abf_json) + "|" + string(ibf_json) + "|" + string(sbf_json) + "|" + string(autonomous_file_hashes_json) + "|" + string(idle_file_hashes_json) + "|" + string(service_failed_file_hashes) + "|" + string(chal_json)
	return message
}
