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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/chain"
	"github.com/CESSProject/cess-bucket/pkg/pbc"
)

func GetReportReq(callbackRouter, callbackIp string, callbackPort int, callUrl string) error {
	callbackurl := fmt.Sprintf("http://%v:%d%v", callbackIp, callbackPort, callbackRouter)
	param := map[string]string{
		"callback_url": callbackurl,
	}
	data, err := json.Marshal(param)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, callUrl, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json;charset=UTF-8")

	cli := http.Client{
		Transport: configs.GlobalTransport,
	}

	_, err = cli.Do(req)
	if err != nil {
		return err
	}

	return nil
}

func GetFillFileReq(fpath string, fsize int, callUrl string) error {
	param := struct {
		File_path string `json:"file_path"`
		Data_len  int    `json:"data_len"`
	}{
		File_path: configs.SgxMappingPath + fpath,
		Data_len:  fsize,
	}
	data, err := json.Marshal(param)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, callUrl, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json;charset=UTF-8")

	cli := http.Client{
		Transport: configs.GlobalTransport,
	}

	_, err = cli.Do(req)
	if err != nil {
		return err
	}

	return nil
}

func GetTagReq(fpath string, blocksize, segmentSize, callbackPort int, callUrl, callbackRouter, callbackIp string) error {
	callbackurl := fmt.Sprintf("http://%v:%d%v", callbackIp, callbackPort, callbackRouter)
	param := struct {
		File_path    string `json:"file_path"`
		Block_size   int    `json:"block_size"`
		Callback_url string `json:"callback_url"`
		Segment_size int    `json:"segment_size"`
	}{
		File_path:    configs.SgxMappingPath + fpath,
		Block_size:   blocksize,
		Callback_url: callbackurl,
		Segment_size: segmentSize,
	}
	data, err := json.Marshal(param)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, callUrl, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json;charset=UTF-8")

	cli := http.Client{
		Transport: configs.GlobalTransport,
	}

	_, err = cli.Do(req)
	if err != nil {
		return err
	}

	return nil
}

func GetChallengeReq(bloacks, callbackPort int, callUrl, callbackRouter, callbackIp string, random chain.Random) ([]pbc.QElement, error) {
	callbackurl := fmt.Sprintf("http://%v:%d%v", callbackIp, callbackPort, callbackRouter)
	randomBytes := make([]byte, len(random))
	for i := 0; i < len(random); i++ {
		randomBytes[i] = byte(random[i])
	}
	param := struct {
		N_blocks     int    `json:"n_blocks"`
		Callback_url string `json:"callback_url"`
		Proof_id     []byte `json:"proof_id"`
	}{
		N_blocks:     bloacks,
		Callback_url: callbackurl,
		Proof_id:     randomBytes,
	}
	data, err := json.Marshal(param)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, callUrl, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json;charset=UTF-8")

	cli := http.Client{
		Transport: configs.GlobalTransport,
	}

	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	val, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	var chalResult ChalResponse
	err = json.Unmarshal(val, &chalResult)
	if err != nil {
		return nil, err
	}

	return chalResult.QElement, nil
}

func GetProofResultReq(callbackPort int, callUrl, callbackRouter, callbackIp string, random chain.Random, proofType uint, proofData []byte) error {
	callbackurl := fmt.Sprintf("http://%v:%d%v", callbackIp, callbackPort, callbackRouter)
	randomBytes := make([]byte, len(random))
	for i := 0; i < len(random); i++ {
		randomBytes[i] = byte(random[i])
	}
	param := struct {
		ProofId     []byte `json:"proof_id"`
		ProofJson   []byte `json:"proof_json"`
		CallbackUrl string `json:"callback_url"`
		VerifyType  uint   `json:"verify_type"`
	}{
		ProofId:     randomBytes,
		ProofJson:   proofData,
		CallbackUrl: callbackurl,
		VerifyType:  proofType,
	}
	data, err := json.Marshal(param)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, callUrl, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json;charset=UTF-8")

	cli := http.Client{
		Transport: configs.GlobalTransport,
	}

	_, err = cli.Do(req)
	if err != nil {
		return err
	}

	return nil
}
