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

package serve

import (
	"errors"
	"net"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/chain"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/go-keyring"
)

func VerifySign(pkey, signmsg, sign []byte) (bool, error) {
	if len(signmsg) == 0 || len(sign) < 64 {
		return false, errors.New("Wrong signature")
	}

	ss58, err := utils.EncodePublicKeyAsSubstrateAccount(pkey)
	if err != nil {
		return false, err
	}

	verkr, _ := keyring.FromURI(ss58, keyring.NetSubstrate{})

	var sign_array [64]byte
	for i := 0; i < 64; i++ {
		sign_array[i] = sign[i]
	}

	// Verify signature
	return verkr.Verify(verkr.SigningContext(signmsg), sign_array), nil
}

func GetFileState(c chain.IChain, fileHash string) (string, error) {
	var try_count uint8
	for try_count <= 3 {
		fmeta, err := c.GetFileMetaInfo(fileHash)
		if err != nil {
			try_count++
			if try_count >= 3 {
				return "", err
			}
			time.Sleep(time.Second * 3)
			continue
		}
		return string(fmeta.State), nil
	}
	return "", errors.New("GetFileMetaInfo failed")
}

func dialTcpServer(address string) (*net.TCPConn, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return nil, err
	}
	dialer := net.Dialer{Timeout: configs.Tcp_Dial_Timeout}
	netCon, err := dialer.Dial("tcp", tcpAddr.String())
	if err != nil {
		return nil, err
	}
	conTcp, ok := netCon.(*net.TCPConn)
	if !ok {
		netCon.Close()
		return nil, errors.New("network conversion failed")
	}
	return conTcp, nil
}
