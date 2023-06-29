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

	"github.com/CESSProject/cess-bucket/pkg/cache"
	"github.com/CESSProject/cess-bucket/pkg/utils"
	"github.com/CESSProject/cess-go-sdk/core/event"
	"github.com/CESSProject/cess-go-sdk/core/pattern"
	sutils "github.com/CESSProject/cess-go-sdk/core/utils"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
)

func (n *Node) parseBlockMgt(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Pnc(utils.RecoverError(err))
		}
	}()

	n.Parseblock("info", ">>>>> Start parseBlockMgt <<<<<")

	var err error
	var recordErr string

	tick := time.NewTicker(time.Minute)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			if n.GetChainState() {
				err = n.pBlock()
				if err != nil {
					if recordErr != err.Error() {
						n.Parseblock("err", err.Error())
						recordErr = err.Error()
					}
				}
			} else {
				if recordErr != pattern.ERR_RPC_CONNECTION.Error() {
					n.Parseblock("err", pattern.ERR_RPC_CONNECTION.Error())
					recordErr = pattern.ERR_RPC_CONNECTION.Error()
				}
			}
		}
	}
}

func (n *Node) pBlock() error {
	var err error
	var pdblock uint32
	var latestBlockHeight uint32
	var parsedBlock int
	var b []byte

	latestBlockHeight, err = n.QueryBlockHeight("")
	if err != nil {
		return errors.Wrapf(err, "[QueryBlockHeight]")
	}

	b, err = n.Get([]byte(Cach_prefix_ParseBlock))
	if err != nil {
		if err == cache.NotFound {
			_, err = n.parseOldBlocks(0, 100)
			if err != nil {
				return errors.Wrapf(err, "[parseOldBlocks]")
			}
		}
		return errors.Wrapf(err, "[cache.Get]")
	}

	parsedBlock, err = strconv.Atoi(string(b))
	if err != nil {
		return errors.Wrapf(err, "[strconv.Atoi]")
	}

	pdblock, err = n.parseOldBlocks(uint32(parsedBlock+1), latestBlockHeight)
	n.Put([]byte(Cach_prefix_ParseBlock), []byte(fmt.Sprintf("%d", pdblock)))
	if err != nil {
		return errors.Wrapf(err, "[parseOldBlocks]")
	}

	return nil
}

func (n *Node) parseOldBlocks(startBlock, endBlock uint32) (uint32, error) {
	var err error
	var parsedBlock uint32
	var peerid string
	var stakingAcc string
	var storageNode pattern.MinerInfo
	var blockhash types.Hash
	var h *types.StorageDataRaw

	if startBlock > endBlock {
		return startBlock, errors.New("startBlock cannot be larger than endBlock")
	}

	parsedBlock = startBlock
	for i := startBlock; i <= endBlock; i++ {
		blockhash, err = n.GetSubstrateAPI().RPC.Chain.GetBlockHash(uint64(i))
		if err != nil {
			return parsedBlock, errors.Wrapf(err, "[GetBlockHash]")
		}

		h, err = n.GetSubstrateAPI().RPC.State.GetStorageRaw(n.GetKeyEvents(), blockhash)
		if err != nil {
			return parsedBlock, errors.Wrapf(err, "[GetStorageRaw]")
		}

		var events = event.EventRecords{}
		err = types.EventRecordsRaw(*h).DecodeEventRecords(n.GetMetadata(), &events)
		if err != nil {
			return parsedBlock, errors.Wrapf(err, "[DecodeEventRecords]")
		}

		// Corresponding processing according to different events
		for _, v := range events.Sminer_Registered {
			storageNode, err = n.QueryStorageMiner(v.Acc[:])
			if err != nil {
				n.Parseblock("err", fmt.Sprintf("[QueryStorageMiner] %v", err.Error()))
				continue
			}
			stakingAcc, _ = sutils.EncodePublicKeyAsCessAccount(v.Acc[:])
			peerid = base58.Encode([]byte(string(storageNode.PeerId[:])))
			n.SaveStoragePeer(peerid, stakingAcc)
			n.Parseblock("info", fmt.Sprintf("Record a storage node: %s", peerid))
		}

		for _, v := range events.FileBank_StorageCompleted {
			n.Put([]byte(Cach_prefix_File+string(v.FileHash[:])), []byte(fmt.Sprintf("%d", i)))
			n.Parseblock("info", fmt.Sprintf("Record a file: %s", string(v.FileHash[:])))
		}

		for _, v := range events.TeeWorker_RegistrationTeeWorker {
			peerid = base58.Encode([]byte(string(v.PeerId[:])))
			n.SaveTeePeer(peerid, 0)
			n.Parseblock("info", fmt.Sprintf("Record a tee node: %s", peerid))
		}

		n.Put([]byte(Cach_prefix_ParseBlock), []byte(fmt.Sprintf("%d", i)))
		n.Parseblock("info", fmt.Sprintf("parsed block: %d", i))
		parsedBlock = i
	}
	return parsedBlock, nil
}
