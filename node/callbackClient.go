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
	"net/http"

	"github.com/CESSProject/cess-bucket/configs"
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
		Transport: globalTransport,
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
		Transport: globalTransport,
	}

	_, err = cli.Do(req)
	if err != nil {
		return err
	}

	return nil
}

func GetTagReq(fpath string, blocksize, callbackPort int, callUrl, callbackRouter, callbackIp string) error {
	callbackurl := fmt.Sprintf("http://%v:%d%v", callbackIp, callbackPort, callbackRouter)
	param := struct {
		File_path    string `json:"file_path"`
		Block_size   int    `json:"block_size"`
		Callback_url string `json:"callback_url"`
	}{
		File_path:    configs.SgxMappingPath + fpath,
		Block_size:   blocksize,
		Callback_url: callbackurl,
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
		Transport: globalTransport,
	}

	_, err = cli.Do(req)
	if err != nil {
		return err
	}

	return nil
}
