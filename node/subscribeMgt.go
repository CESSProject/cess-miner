/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package node

import (
	"fmt"
	"strconv"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/event"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	sutils "github.com/CESSProject/cess-go-sdk/core/utils"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
)

func (n *Node) subscribeMgt(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	n.Subscribe("info", ">>>>> Start subscribeMgt task")

	var err error
	var startBlock uint32
	var peerid string
	var stakingAcc string
	var b []byte
	var head types.Header
	var storageNode pattern.MinerInfo
	var events = event.EventRecords{}

	b, err = n.Get([]byte(Cach_prefix_ParseBlock))
	if err == nil {
		block, err := strconv.Atoi(string(b))
		if err == nil {
			startBlock = uint32(block)
		}
	}

	for {
		startBlock, err = n.parsingOldBlocks(startBlock)
		if err == nil {
			break
		}
		n.Subscribe("err", err.Error())
		time.Sleep(time.Minute)
	}
	for {
		if n.GetChainState() {
			sub, err := n.GetSubstrateAPI().RPC.Chain.SubscribeNewHeads()
			if err != nil {
				n.Subscribe("err", fmt.Sprintf("[SubscribeNewHeads] %v", err.Error()))
				time.Sleep(pattern.BlockInterval)
				continue
			}
			defer sub.Unsubscribe()
			for {
				head = <-sub.Chan()
				blockhash, err := n.GetSubstrateAPI().RPC.Chain.GetBlockHash(uint64(head.Number))
				if err != nil {
					n.Subscribe("err", fmt.Sprintf("[GetBlockHash] %v", err.Error()))
					continue
				}

				h, err := n.GetSubstrateAPI().RPC.State.GetStorageRaw(n.GetKeyEvents(), blockhash)
				if err != nil {
					n.Subscribe("err", fmt.Sprintf("[GetStorageRaw] %v", err.Error()))
					continue
				}

				err = types.EventRecordsRaw(*h).DecodeEventRecords(n.GetMetadata(), &events)
				if err != nil {
					n.Subscribe("err", fmt.Sprintf("[DecodeEventRecords] %v", err.Error()))
					continue
				}

				// Corresponding processing according to different events
				for _, v := range events.Sminer_Registered {
					storageNode, err = n.QueryStorageMiner(v.Acc[:])
					if err != nil {
						n.Subscribe("err", fmt.Sprintf("[QueryStorageMiner] %v", err.Error()))
						continue
					}
					stakingAcc, _ = sutils.EncodePublicKeyAsCessAccount(v.Acc[:])
					peerid = base58.Encode([]byte(string(storageNode.PeerId[:])))
					n.SaveStoragePeer(peerid, stakingAcc)
					configs.Tip(fmt.Sprintf("Record a storage node: %s", peerid))
					n.Subscribe("info", fmt.Sprintf("Record a storage node: %s", peerid))
				}

				for _, v := range events.FileBank_StorageCompleted {
					n.Put([]byte(Cach_prefix_File+string(v.FileHash[:])), []byte(fmt.Sprintf("%d", head.Number)))
					n.Subscribe("info", fmt.Sprintf("Record a file: %s", string(v.FileHash[:])))
				}

				for _, v := range events.TeeWorker_RegistrationTeeWorker {
					peerid = base58.Encode([]byte(string(v.PeerId[:])))
					n.SaveTeePeer(peerid, 0)
					configs.Tip(fmt.Sprintf("Record a tee node: %s", peerid))
				}
				n.Put([]byte(Cach_prefix_ParseBlock), []byte(fmt.Sprintf("%d", head.Number)))
				n.Subscribe("info", fmt.Sprintf("Parse block: %d", head.Number))
			}
		}
		time.Sleep(time.Millisecond * 20)
	}
}

func (n *Node) parsingOldBlocks(block uint32) (uint32, error) {
	var err error
	var peerid string
	var stakingAcc string
	var blockheight uint32
	var startBlock uint32 = block
	var storageNode pattern.MinerInfo
	var blockhash types.Hash
	var h *types.StorageDataRaw
	var events = event.EventRecords{}
	for {
		blockheight, err = n.QueryBlockHeight("")
		if err != nil {
			return startBlock, errors.Wrapf(err, "[QueryBlockHeight]")
		}
		if startBlock >= blockheight {
			return startBlock, nil
		}
		for i := startBlock; i <= blockheight; i++ {
			blockhash, err = n.GetSubstrateAPI().RPC.Chain.GetBlockHash(uint64(i))
			if err != nil {
				return startBlock, errors.Wrapf(err, "[GetBlockHash]")
			}

			h, err = n.GetSubstrateAPI().RPC.State.GetStorageRaw(n.GetKeyEvents(), blockhash)
			if err != nil {
				return startBlock, errors.Wrapf(err, "[GetStorageRaw]")
			}

			err = types.EventRecordsRaw(*h).DecodeEventRecords(n.GetMetadata(), &events)
			if err != nil {
				return startBlock, errors.Wrapf(err, "[DecodeEventRecords]")
			}

			// Corresponding processing according to different events
			for _, v := range events.Sminer_Registered {
				storageNode, err = n.QueryStorageMiner(v.Acc[:])
				if err != nil {
					n.Subscribe("err", fmt.Sprintf("[QueryStorageMiner] %v", err.Error()))
					continue
				}
				stakingAcc, _ = sutils.EncodePublicKeyAsCessAccount(v.Acc[:])
				peerid = base58.Encode([]byte(string(storageNode.PeerId[:])))
				n.SaveStoragePeer(peerid, stakingAcc)
				configs.Tip(fmt.Sprintf("Record a storage node: %s", peerid))
				n.Subscribe("info", fmt.Sprintf("Record a storage node: %s", peerid))
			}

			for _, v := range events.FileBank_StorageCompleted {
				n.Put([]byte(Cach_prefix_File+string(v.FileHash[:])), []byte(fmt.Sprintf("%d", i)))
				n.Subscribe("info", fmt.Sprintf("Record a file: %s", string(v.FileHash[:])))
			}

			for _, v := range events.TeeWorker_RegistrationTeeWorker {
				peerid = base58.Encode([]byte(string(v.PeerId[:])))
				n.SaveTeePeer(peerid, 0)
				configs.Tip(fmt.Sprintf("Record a tee node: %s", peerid))
			}

			n.Put([]byte(Cach_prefix_ParseBlock), []byte(fmt.Sprintf("%d", i)))
			n.Subscribe("info", fmt.Sprintf("Parse block: %d", i))
			startBlock = i
		}
		startBlock = blockheight
	}
}
