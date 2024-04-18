/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/logger"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	"github.com/CESSProject/cess-go-sdk/core/sdk"
	sutils "github.com/CESSProject/cess-go-sdk/utils"
	"github.com/CESSProject/p2p-go/core"
	"github.com/CESSProject/p2p-go/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type DataDir struct {
	DbDir           string
	LogDir          string
	SpaceDir        string
	PoisDir         string
	AccDir          string
	RandomDir       string
	PeersFile       string
	Podr2PubkeyFile string
}

const (
	Active = iota
	Calculate
	Missing
	Recovery
)

const (
	// Record the fid of stored files
	Cach_prefix_File = "file:"
	// Record the block of reported tags
	Cach_prefix_Tag = "tag:"

	Cach_prefix_MyLost      = "mylost:"
	Cach_prefix_recovery    = "recovery:"
	Cach_prefix_TargetMiner = "targetminer:"
	Cach_prefix_ParseBlock  = "parseblocks"
)

func SyncTeeInfo(cli sdk.SDK, l logger.Logger, peernode *core.PeerNode, teeRecord *TeeRecord, ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			l.Pnc(utils.RecoverError(err))
		}
	}()
	var dialOptions []grpc.DialOption
	var chainPublickey = make([]byte, pattern.WorkerPublicKeyLen)
	teelist, err := cli.QueryAllTeeWorkerMap()
	if err != nil {
		l.Log("err", err.Error())
	} else {
		for i := 0; i < len(teelist); i++ {
			l.Log("info", fmt.Sprintf("check tee: %s", hex.EncodeToString([]byte(string(teelist[i].Pubkey[:])))))
			endpoint, err := cli.QueryTeeWorkEndpoint(teelist[i].Pubkey)
			if err != nil {
				l.Log("err", err.Error())
				continue
			}
			endpoint = ProcessTeeEndpoint(endpoint)

			if !strings.Contains(endpoint, "443") {
				dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
			} else {
				dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(configs.GetCert())}
			}

			// verify identity public key
			identityPubkeyResponse, err := peernode.GetIdentityPubkey(endpoint,
				&pb.Request{
					StorageMinerAccountId: cli.GetSignatureAccPulickey(),
				},
				time.Duration(time.Minute),
				dialOptions,
				nil,
			)
			if err != nil {
				l.Log("err", err.Error())
				continue
			}
			//n.Log("info", fmt.Sprintf("get identityPubkeyResponse: %v", identityPubkeyResponse.Pubkey))
			if len(identityPubkeyResponse.Pubkey) != pattern.WorkerPublicKeyLen {
				teeRecord.DeleteTee(string(teelist[i].Pubkey[:]))
				l.Log("err", fmt.Sprintf("identityPubkeyResponse.Pubkey length err: %d", len(identityPubkeyResponse.Pubkey)))
				continue
			}

			for j := 0; j < pattern.WorkerPublicKeyLen; j++ {
				chainPublickey[j] = byte(teelist[i].Pubkey[j])
			}
			if !sutils.CompareSlice(identityPubkeyResponse.Pubkey, chainPublickey) {
				teeRecord.DeleteTee(string(teelist[i].Pubkey[:]))
				l.Log("err", fmt.Sprintf("identityPubkeyResponse.Pubkey: %s", hex.EncodeToString(identityPubkeyResponse.Pubkey)))
				l.Log("err", "identityPubkeyResponse.Pubkey err: not qual to chain")
				continue
			}

			l.Log("info", fmt.Sprintf("Save a tee: %s  %d", endpoint, teelist[i].Role))
			err = teeRecord.SaveTee(string(teelist[i].Pubkey[:]), endpoint, uint8(teelist[i].Role))
			if err != nil {
				l.Log("err", err.Error())
			}
		}
	}
}

func (n *Node) WatchMem() {
	memSt := &runtime.MemStats{}
	tikProgram := time.NewTicker(time.Second * 3)
	defer tikProgram.Stop()

	for range tikProgram.C {
		runtime.ReadMemStats(memSt)
		if memSt.HeapSys >= pattern.SIZE_1GiB*8 {
			n.Log("err", fmt.Sprintf("Mem heigh: %d", memSt.HeapSys))
			os.Exit(1)
		}
	}
}

func ProcessTeeEndpoint(endPoint string) string {
	var teeEndPoint string
	if strings.HasPrefix(endPoint, "http://") {
		teeEndPoint = strings.TrimPrefix(endPoint, "http://")
		teeEndPoint = strings.TrimSuffix(teeEndPoint, "/")
		if !strings.Contains(teeEndPoint, ":") {
			teeEndPoint = teeEndPoint + ":80"
		}
	} else if strings.HasPrefix(endPoint, "https://") {
		teeEndPoint = strings.TrimPrefix(endPoint, "https://")
		teeEndPoint = strings.TrimSuffix(teeEndPoint, "/")
		if !strings.Contains(teeEndPoint, ":") {
			teeEndPoint = teeEndPoint + ":443"
		}
	} else {
		if !strings.Contains(endPoint, ":") {
			teeEndPoint = endPoint + ":80"
		} else {
			teeEndPoint = endPoint
		}
	}
	return teeEndPoint
}

func GetFragmentFromOss(fid string, signAcc string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s%s", configs.DefaultDeossAddr, fid), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Account", signAcc)
	req.Header.Set("Operation", "download")

	client := &http.Client{}
	client.Transport = utils.GlobalTransport
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed")
	}
	data, err := io.ReadAll(resp.Body)
	return data, err
}
