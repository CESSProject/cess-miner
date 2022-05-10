package chain

import (
	"encoding/json"
	"math/big"
	"strconv"

	"cess-bucket/configs"
	. "cess-bucket/internal/logger"
	"cess-bucket/tools"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

// miner register
func RegisterToChain(transactionPrK, revenuePuK, ipAddr, TransactionName string, pledgeTokens uint64, puk []byte) (bool, error) {
	var (
		err         error
		accountInfo types.AccountInfo
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()

	keyring, err := signature.KeyringPairFromSecret(transactionPrK, 0)
	if err != nil {
		return false, errors.Wrap(err, "KeyringPairFromSecret err")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return false, errors.Wrap(err, "GetMetadataLatest err")
	}

	incomeAccount, err := types.NewMultiAddressFromHexAccountID(revenuePuK)
	if err != nil {
		return false, errors.Wrap(err, "NewMultiAddressFromHexAccountID err")
	}

	pTokens := strconv.FormatUint(pledgeTokens, 10)
	pTokens += configs.TokenAccuracy
	realTokens, ok := new(big.Int).SetString(pTokens, 10)
	if !ok {
		return false, errors.New("big.Int.SetString err")
	}
	tokens := types.NewUCompact(realTokens)

	c, err := types.NewCall(meta, TransactionName, incomeAccount, types.Bytes([]byte(ipAddr)), tokens, types.Bytes(puk))
	if err != nil {
		return false, errors.Wrap(err, "NewCall err")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return false, errors.Wrap(err, "NewExtrinsic err")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return false, errors.Wrap(err, "GetBlockHash err")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return false, errors.Wrap(err, "GetRuntimeVersionLatest err")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return false, errors.Wrap(err, "CreateStorageKey System  Account err")
	}

	keye, err := types.CreateStorageKey(meta, "System", "Events", nil)
	if err != nil {
		return false, errors.Wrap(err, "CreateStorageKey System Events err")
	}

	ok, err = api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return false, errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return false, errors.New("GetStorageLatest return value is empty")
	}

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	// Sign the transaction
	err = ext.Sign(keyring, o)
	if err != nil {
		return false, errors.Wrap(err, "Sign err")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return false, errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()
	var head *types.Header
	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := MyEventRecords{}
				head, _ = api.RPC.Chain.GetHeader(status.AsInBlock)
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					if head != nil {
						return false, errors.Wrapf(err, "[%v]", head.Number)
					} else {
						return false, err
					}
				}
				err = types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
				if err != nil {
					if head != nil {
						Out.Sugar().Infof("[%v]Decode event err:%v", head.Number, err)
					} else {
						Out.Sugar().Infof("Decode event err:%v", err)
					}
				}
				if events.Sminer_Registered != nil {
					for i := 0; i < len(events.Sminer_Registered); i++ {
						if events.Sminer_Registered[i].Acc == types.NewAccountID(keyring.PublicKey) {
							return true, nil
						}
					}
					if head != nil {
						return false, errors.Errorf("[%v]events.Sminer_Registered data err", head.Number)
					} else {
						return false, errors.New("events.Sminer_Registered data err")
					}
				}
				if head != nil {
					return false, errors.Errorf("[%v]events.Sminer_Registered not found", head.Number)
				} else {
					return false, errors.New("events.Sminer_Registered not found")
				}
			}
		case err = <-sub.Err():
			return false, err
		case <-timeout:
			return false, errors.New("SubmitAndWatchExtrinsic timeout")
		}
	}
}

//
func IntentSubmitToChain(identifyAccountPhrase, TransactionName string, segsizetype, segtype uint8, peerid uint64, unsealedcid [][]byte, shardhash []byte) (uint64, uint32, error) {
	var (
		err         error
		ok          bool
		accountInfo types.AccountInfo
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	keyring, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return 0, 0, errors.Wrap(err, "KeyringPairFromSecret err")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return 0, 0, errors.Wrap(err, "GetMetadataLatest err")
	}
	var uncid []types.Bytes = make([]types.Bytes, len(unsealedcid))
	for i := 0; i < len(unsealedcid); i++ {
		uncid[i] = make(types.Bytes, 0)
		uncid[i] = append(uncid[i], unsealedcid[i]...)
	}
	c, err := types.NewCall(meta, TransactionName, types.U8(segsizetype), types.U8(segtype), types.U64(peerid), uncid, types.Bytes(shardhash))
	if err != nil {
		return 0, 0, errors.Wrap(err, "NewCall err")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return 0, 0, errors.Wrap(err, "NewExtrinsic err")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return 0, 0, errors.Wrap(err, "GetBlockHash err")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return 0, 0, errors.Wrap(err, "GetRuntimeVersionLatest err")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return 0, 0, errors.Wrap(err, "CreateStorageKey err")
	}

	keye, err := types.CreateStorageKey(meta, "System", "Events", nil)
	if err != nil {
		return 0, 0, errors.Wrap(err, "CreateStorageKey System Events err")
	}

	ok, err = api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return 0, 0, errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return 0, 0, errors.New("GetStorageLatest return value is empty")
	}

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	// Sign the transaction
	err = ext.Sign(keyring, o)
	if err != nil {
		return 0, 0, errors.Wrap(err, "Sign err")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return 0, 0, errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()
	var head *types.Header
	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := MyEventRecords{}
				head, _ = api.RPC.Chain.GetHeader(status.AsInBlock)
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					if head != nil {
						return 0, 0, errors.Wrapf(err, "[%v]", head.Number)
					} else {
						return 0, 0, err
					}
				}

				err = types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
				if err != nil {
					if head != nil {
						Out.Sugar().Infof("[%v]Decode event err:%v", head.Number, err)
					} else {
						Out.Sugar().Infof("Decode event err:%v", err)
					}
				}
				if events.SegmentBook_ParamSet != nil {
					for i := 0; i < len(events.SegmentBook_ParamSet); i++ {
						if events.SegmentBook_ParamSet[i].PeerId == types.U64(configs.MinerId_I) {
							return uint64(events.SegmentBook_ParamSet[i].SegmentId), uint32(events.SegmentBook_ParamSet[i].Random), nil
						}
					}
					if head != nil {
						return 0, 0, errors.Errorf("[%v]events.SegmentBook_ParamSet data err", head.Number)
					} else {
						return 0, 0, errors.New("events.SegmentBook_ParamSet data err")
					}
				}
				if head != nil {
					return 0, 0, errors.Errorf("[%v]events.SegmentBook_ParamSet not found", head.Number)
				} else {
					return 0, 0, errors.New("events.SegmentBook_ParamSet not found")
				}
			}
		case err = <-sub.Err():
			return 0, 0, err
		case <-timeout:
			return 0, 0, errors.New("SubmitAndWatchExtrinsic timeout")
		}
	}
}

//
func IntentSubmitPostToChain(identifyAccountPhrase, TransactionName string, segmentid uint64, segsizetype, segtype uint8) (uint32, error) {
	var (
		ok          bool
		accountInfo types.AccountInfo
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	keyring, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return 0, errors.Wrap(err, "KeyringPairFromSecret err")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return 0, errors.Wrap(err, "GetMetadataLatest err")
	}

	c, err := types.NewCall(meta, TransactionName, types.U64(segmentid), types.U8(segsizetype), types.U8(segtype))
	if err != nil {
		return 0, errors.Wrap(err, "NewCall err")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return 0, errors.Wrap(err, "NewExtrinsic err")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return 0, errors.Wrap(err, "GetBlockHash err")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return 0, errors.Wrap(err, "GetRuntimeVersionLatest err")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return 0, errors.Wrap(err, "CreateStorageKey err")
	}

	keye, err := types.CreateStorageKey(meta, "System", "Events", nil)
	if err != nil {
		return 0, errors.Wrap(err, "CreateStorageKey System Events err")
	}

	ok, err = api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return 0, errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return 0, errors.New("GetStorageLatest return value is empty")
	}

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	// Sign the transaction
	err = ext.Sign(keyring, o)
	if err != nil {
		return 0, errors.Wrap(err, "Sign err")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return 0, errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()

	var head *types.Header
	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := MyEventRecords{}
				head, _ = api.RPC.Chain.GetHeader(status.AsInBlock)
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					if head != nil {
						return 0, errors.Wrapf(err, "[%v]", head.Number)
					} else {
						return 0, err
					}
				}
				err = types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
				if err != nil {
					if head != nil {
						Out.Sugar().Infof("[%v]Decode event err:%v", head.Number, err)
					} else {
						Out.Sugar().Infof("Decode event err:%v", err)
					}
				}
				if events.SegmentBook_ParamSet != nil {
					for i := 0; i < len(events.SegmentBook_ParamSet); i++ {
						if events.SegmentBook_ParamSet[i].PeerId == types.U64(configs.MinerId_I) && events.SegmentBook_ParamSet[i].SegmentId == types.U64(segmentid) {
							return uint32(events.SegmentBook_ParamSet[i].Random), nil
						}
					}
					if head != nil {
						return 0, errors.Errorf("[%v]events.SegmentBook_ParamSet data err", head.Number)
					} else {
						return 0, errors.New("events.SegmentBook_ParamSet data err")
					}
				}
				if head != nil {
					return 0, errors.Errorf("[%v]events.SegmentBook_ParamSet not found", head.Number)
				} else {
					return 0, errors.New("events.SegmentBook_ParamSet not found")
				}
			}
		case err = <-sub.Err():
			return 0, errors.Wrap(err, "sub.Err")
		case <-timeout:
			return 0, errors.New("SubmitAndWatchExtrinsic timeout")
		}
	}
}

// Submit To Vpa or Vpb
func SegmentSubmitToVpaOrVpb(identifyAccountPhrase, TransactionName string, peerid, segmentid uint64, proofs, cid []byte) (bool, error) {
	var (
		err         error
		ok          bool
		accountInfo types.AccountInfo
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	keyring, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return false, errors.Wrapf(err, "KeyringPairFromSecret err [%v]", TransactionName)
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return false, errors.Wrapf(err, "GetMetadataLatest err [%v]", TransactionName)
	}

	c, err := types.NewCall(meta, TransactionName, types.U64(peerid), types.U64(segmentid), types.Bytes(proofs), types.Bytes(cid))
	if err != nil {
		return false, errors.Wrapf(err, "NewCall err [%v]", TransactionName)
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return false, errors.Wrap(err, "NewExtrinsic err")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return false, errors.Wrap(err, "GetBlockHash err")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return false, errors.Wrap(err, "GetRuntimeVersionLatest err")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return false, errors.Wrap(err, "CreateStorageKey err")
	}

	keye, err := types.CreateStorageKey(meta, "System", "Events", nil)
	if err != nil {
		return false, errors.Wrap(err, "CreateStorageKey System Events err")
	}

	ok, err = api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return false, errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return false, errors.New("GetStorageLatest return value is empty")
	}

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	// Sign the transaction
	err = ext.Sign(keyring, o)
	if err != nil {
		return false, errors.Wrap(err, "Sign err")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return false, errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()
	var head *types.Header
	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := MyEventRecords{}
				head, _ = api.RPC.Chain.GetHeader(status.AsInBlock)
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					if head != nil {
						return false, errors.Wrapf(err, "[%v]", head.Number)
					} else {
						return false, err
					}
				}
				err = types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
				if err != nil {
					if head != nil {
						Out.Sugar().Infof("[%v]Decode event err:%v", head.Number, err)
					} else {
						Out.Sugar().Infof("Decode event err:%v", err)
					}
				}
				switch TransactionName {
				case configs.ChainTx_SegmentBook_SubmitToVpa:
					if events.SegmentBook_VPASubmitted != nil {
						for i := 0; i < len(events.SegmentBook_VPASubmitted); i++ {
							if events.SegmentBook_VPASubmitted[i].PeerId == types.U64(configs.MinerId_I) && events.SegmentBook_VPASubmitted[i].SegmentId == types.U64(segmentid) {
								return true, nil
							}
						}
						if head != nil {
							return false, errors.Errorf("[%v]events.SegmentBook_VPASubmitted data err", head.Number)
						} else {
							return false, errors.New("events.SegmentBook_VPASubmitted data err")
						}
					}
					if head != nil {
						return false, errors.Errorf("[%v]events.SegmentBook_VPASubmitted not found", head.Number)
					} else {
						return false, errors.New("events.SegmentBook_VPASubmitted not found")
					}
				case configs.ChainTx_SegmentBook_SubmitToVpb:
					if events.SegmentBook_VPBSubmitted != nil {
						for i := 0; i < len(events.SegmentBook_VPBSubmitted); i++ {
							if events.SegmentBook_VPBSubmitted[i].PeerId == types.U64(configs.MinerId_I) && events.SegmentBook_VPBSubmitted[i].SegmentId == types.U64(segmentid) {
								return true, nil
							}
						}
						if head != nil {
							return false, errors.Errorf("[%v]events.SegmentBook_VPBSubmitted data err", head.Number)
						} else {
							return false, errors.New("events.SegmentBook_VPBSubmitted data err")
						}
					}
					if head != nil {
						return false, errors.Errorf("[%v]events.SegmentBook_VPBSubmitted not found", head.Number)
					} else {
						return false, errors.New("events.SegmentBook_VPBSubmitted not found")
					}
				}
				if head != nil {
					return false, errors.Errorf("[%v]events.ChainTx_SegmentBook_SubmitToVpa/b not found", head.Number)
				} else {
					return false, errors.New("events.ChainTx_SegmentBook_SubmitToVpa/b not found")
				}
			}
		case err = <-sub.Err():
			return false, err
		case <-timeout:
			return false, errors.New("SubmitAndWatchExtrinsic timeout")
		}
	}
}

// Submit To Vpc
func SegmentSubmitToVpc(identifyAccountPhrase, TransactionName string, peerid, segmentid uint64, proofs [][]byte, sealcid []types.Bytes, fid types.Bytes) (bool, error) {
	var (
		err         error
		ok          bool
		accountInfo types.AccountInfo
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	keyring, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return false, errors.Wrap(err, "KeyringPairFromSecret err")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return false, errors.Wrap(err, "GetMetadataLatest err")
	}

	var fileVpc []types.Bytes = make([]types.Bytes, len(proofs))
	for i := 0; i < len(proofs); i++ {
		fileVpc[i] = make(types.Bytes, 0)
		fileVpc[i] = append(fileVpc[i], proofs[i]...)
	}

	c, err := types.NewCall(meta, TransactionName, types.U64(peerid), types.U64(segmentid), fileVpc, sealcid, fid)
	if err != nil {
		return false, errors.Wrap(err, "NewCall err")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return false, errors.Wrap(err, "NewExtrinsic err")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return false, errors.Wrap(err, "GetBlockHash err")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return false, errors.Wrap(err, "GetRuntimeVersionLatest err")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return false, errors.Wrap(err, "CreateStorageKey err")
	}

	keye, err := types.CreateStorageKey(meta, "System", "Events", nil)
	if err != nil {
		return false, errors.Wrap(err, "CreateStorageKey System Events err")
	}

	ok, err = api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return false, errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return false, errors.New("GetStorageLatest return value is empty")
	}

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	// Sign the transaction
	err = ext.Sign(keyring, o)
	if err != nil {
		return false, errors.Wrap(err, "Sign err")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return false, errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()
	var head *types.Header
	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := MyEventRecords{}
				head, _ = api.RPC.Chain.GetHeader(status.AsInBlock)
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					if head != nil {
						return false, errors.Wrapf(err, "[%v]", head.Number)
					} else {
						return false, err
					}
				}
				err = types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
				if err != nil {
					if head != nil {
						Out.Sugar().Infof("[%v]Decode event err:%v", head.Number, err)
					} else {
						Out.Sugar().Infof("Decode event err:%v", err)
					}
				}
				if events.SegmentBook_VPCSubmitted != nil {
					for i := 0; i < len(events.SegmentBook_VPCSubmitted); i++ {
						if events.SegmentBook_VPCSubmitted[i].PeerId == types.U64(configs.MinerId_I) && events.SegmentBook_VPCSubmitted[i].SegmentId == types.U64(segmentid) {
							return true, nil
						}
					}
					if head != nil {
						return false, errors.Errorf("[%v]events.SegmentBook_VPCSubmitted data err", head.Number)
					} else {
						return false, errors.New("events.SegmentBook_VPCSubmitted data err")
					}
				}
				if head != nil {
					return false, errors.Errorf("[%v]Not found events.SegmentBook_VPCSubmitted", head.Number)
				} else {
					return false, errors.New("Not found events.SegmentBook_VPCSubmitted")
				}
			}
		case err = <-sub.Err():
			return false, err
		case <-timeout:
			return false, errors.New("SubmitAndWatchExtrinsic timeout")
		}
	}
}

// Submit To Vpd
func SegmentSubmitToVpd(identifyAccountPhrase, TransactionName string, peerid, segmentid uint64, proofs [][]byte, sealcid []types.Bytes, fid types.Bytes) (bool, error) {
	var (
		err         error
		ok          bool
		accountInfo types.AccountInfo
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	keyring, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return false, errors.Wrap(err, "KeyringPairFromSecret err")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return false, errors.Wrap(err, "GetMetadataLatest err")
	}

	var fileVpd []types.Bytes = make([]types.Bytes, len(proofs))
	for i := 0; i < len(proofs); i++ {
		fileVpd[i] = make(types.Bytes, 0)
		fileVpd[i] = append(fileVpd[i], proofs[i]...)
	}
	c, err := types.NewCall(meta, TransactionName, types.U64(peerid), types.U64(segmentid), fileVpd, sealcid, fid)
	if err != nil {
		return false, errors.Wrap(err, "NewCall err")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return false, errors.Wrap(err, "NewExtrinsic err")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return false, errors.Wrap(err, "GetBlockHash err")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return false, errors.Wrap(err, "GetRuntimeVersionLatest err")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return false, errors.Wrap(err, "CreateStorageKey err")
	}

	keye, err := types.CreateStorageKey(meta, "System", "Events", nil)
	if err != nil {
		return false, errors.Wrap(err, "CreateStorageKey System Events err")
	}

	ok, err = api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return false, errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return false, errors.New("GetStorageLatest return value is empty")
	}

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	// Sign the transaction
	err = ext.Sign(keyring, o)
	if err != nil {
		return false, errors.Wrap(err, "Sign err")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return false, errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()
	var head *types.Header
	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := MyEventRecords{}
				head, _ = api.RPC.Chain.GetHeader(status.AsInBlock)
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					if head != nil {
						return false, errors.Wrapf(err, "[%v]", head.Number)
					} else {
						return false, err
					}
				}
				err = types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
				if err != nil {
					if head != nil {
						Out.Sugar().Infof("[%v]Decode event err:%v", head.Number, err)
					} else {
						Out.Sugar().Infof("Decode event err:%v", err)
					}
				}
				if events.SegmentBook_VPDSubmitted != nil {
					for i := 0; i < len(events.SegmentBook_VPDSubmitted); i++ {
						if events.SegmentBook_VPDSubmitted[i].PeerId == types.U64(configs.MinerId_I) && events.SegmentBook_VPDSubmitted[i].SegmentId == types.U64(segmentid) {
							if head != nil {
								Out.Sugar().Infof("[%v]SegmentBook_VPDSubmitted suc", head.Number)
							}
							return true, nil
						}
					}
					if head != nil {
						return false, errors.Errorf("[%v]events.SegmentBook_VPDSubmitted data err", head.Number)
					} else {
						return false, errors.New("events.SegmentBook_VPDSubmitted data err")
					}
				}
				if head != nil {
					return false, errors.Errorf("[%v]events.SegmentBook_VPDSubmitted not found", head.Number)
				} else {
					return false, errors.New("events.SegmentBook_VPDSubmitted not found")
				}
			}
		case err = <-sub.Err():
			return false, err
		case <-timeout:
			return false, errors.New("SubmitAndWatchExtrinsic timeout")
		}
	}
}

//
func Increase(identifyAccountPhrase, TransactionName string, tokens *big.Int) (bool, error) {
	var (
		err         error
		ok          bool
		accountInfo types.AccountInfo
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	keyring, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return false, errors.Wrap(err, "KeyringPairFromSecret err")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return false, errors.Wrap(err, "GetMetadataLatest err")
	}

	c, err := types.NewCall(meta, TransactionName, types.NewUCompact(tokens))
	if err != nil {
		return false, errors.Wrap(err, "NewCall err")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return false, errors.Wrap(err, "NewExtrinsic err")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return false, errors.Wrap(err, "GetBlockHash err")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return false, errors.Wrap(err, "GetRuntimeVersionLatest err")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return false, errors.Wrap(err, "CreateStorageKey err")
	}

	keye, err := types.CreateStorageKey(meta, "System", "Events", nil)
	if err != nil {
		return false, errors.Wrap(err, "CreateStorageKey System Events err")
	}

	ok, err = api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return false, errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return false, errors.New("GetStorageLatest return value is empty")
	}

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	// Sign the transaction
	err = ext.Sign(keyring, o)
	if err != nil {
		return false, errors.Wrap(err, "Sign err")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return false, errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()

	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := MyEventRecords{}
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					return false, err
				}
				err = types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
				if err != nil {
					Out.Sugar().Infof("Decode event err:%v", err)
				}
				if events.Sminer_IncreaseCollateral != nil {
					for i := 0; i < len(events.Sminer_IncreaseCollateral); i++ {
						if events.Sminer_IncreaseCollateral[i].Acc == types.NewAccountID(keyring.PublicKey) {
							return true, nil
						}
					}
					return false, errors.New("events.Sminer_IncreaseCollateral data err")
				}
				return false, errors.New("events.Sminer_IncreaseCollateral not found")
			}
		case err = <-sub.Err():
			return false, err
		case <-timeout:
			return false, errors.New("SubmitAndWatchExtrinsic timeout")
		}
	}
}

//
func ExitMining(identifyAccountPhrase, TransactionName string) (bool, error) {
	var (
		err         error
		ok          bool
		accountInfo types.AccountInfo
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	keyring, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return false, errors.Wrap(err, "KeyringPairFromSecret err")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return false, errors.Wrap(err, "GetMetadataLatest err")
	}

	c, err := types.NewCall(meta, TransactionName)
	if err != nil {
		return false, errors.Wrap(err, "NewCall err")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return false, errors.Wrap(err, "NewExtrinsic err")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return false, errors.Wrap(err, "GetBlockHash err")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return false, errors.Wrap(err, "GetRuntimeVersionLatest err")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return false, errors.Wrap(err, "CreateStorageKey err")
	}

	keye, err := types.CreateStorageKey(meta, "System", "Events", nil)
	if err != nil {
		return false, errors.Wrap(err, "CreateStorageKey System Events err")
	}

	ok, err = api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return false, errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return false, errors.New("GetStorageLatest return value is empty")
	}

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	// Sign the transaction
	err = ext.Sign(keyring, o)
	if err != nil {
		return false, errors.Wrap(err, "Sign err")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return false, errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()

	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := MyEventRecords{}
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					return false, err
				}
				err = types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
				if err != nil {
					Out.Sugar().Infof("Decode event err:%v", err)
				}
				if events.Sminer_MinerExit != nil {
					for i := 0; i < len(events.Sminer_MinerExit); i++ {
						if events.Sminer_MinerExit[i].Acc == types.NewAccountID(keyring.PublicKey) {
							return true, nil
						}
					}
					return false, errors.New("events.Sminer_MinerExit data err")
				}
				return false, errors.New("events.Sminer_MinerExit not found")
			}
		case err = <-sub.Err():
			return false, err
		case <-timeout:
			return false, errors.New("SubmitAndWatchExtrinsic timeout")
		}
	}
}

//
func Withdraw(identifyAccountPhrase, TransactionName string) (bool, error) {
	var (
		err         error
		ok          bool
		accountInfo types.AccountInfo
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	keyring, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return false, errors.Wrap(err, "KeyringPairFromSecret err")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return false, errors.Wrap(err, "GetMetadataLatest err")
	}

	c, err := types.NewCall(meta, TransactionName)
	if err != nil {
		return false, errors.Wrap(err, "NewCall err")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return false, errors.Wrap(err, "NewExtrinsic err")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return false, errors.Wrap(err, "GetBlockHash err")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return false, errors.Wrap(err, "GetRuntimeVersionLatest err")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return false, errors.Wrap(err, "CreateStorageKey err")
	}

	keye, err := types.CreateStorageKey(meta, "System", "Events", nil)
	if err != nil {
		return false, errors.Wrap(err, "CreateStorageKey System Events err")
	}

	ok, err = api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return false, errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return false, errors.New("GetStorageLatest return value is empty")
	}

	o := types.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accountInfo.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: rv.TransactionVersion,
	}

	// Sign the transaction
	err = ext.Sign(keyring, o)
	if err != nil {
		return false, errors.Wrap(err, "Sign err")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return false, errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()

	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := MyEventRecords{}
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					return false, err
				}
				err = types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
				if err != nil {
					Out.Sugar().Infof("Decode event err:%v", err)
				}
				if events.Sminer_MinerClaim != nil {
					for i := 0; i < len(events.Sminer_MinerClaim); i++ {
						if events.Sminer_MinerClaim[i].Acc == types.NewAccountID(keyring.PublicKey) {
							return true, nil
						}
					}
					return false, errors.New("events.Sminer_MinerClaim data err")
				}
				return false, errors.New("events.Sminer_MinerClaim not found")
			}
		case err = <-sub.Err():
			return false, err
		case <-timeout:
			return false, errors.New("SubmitAndWatchExtrinsic timeout")
		}
	}
}

type faucet struct {
	Ans    answer `json:"Result"`
	Status string `json:"Status"`
}
type answer struct {
	Err       string `json:"Err"`
	AsInBlock bool   `json:"AsInBlock"`
}

func ObtainFromFaucet(faucetaddr, pbk string) error {
	var ob = struct {
		Address string `json:"Address"`
	}{
		pbk,
	}
	var res faucet
	resp, err := tools.Post(faucetaddr, ob)
	if err != nil {
		return err
	}
	err = json.Unmarshal(resp, &res)
	if err != nil {
		return err
	}
	if res.Ans.Err != "" {
		return err
	}

	if res.Ans.AsInBlock {
		return nil
	} else {
		return errors.New("The address has been picked up today, please come back after 1 day.")
	}
}

//
func GetAddressFromPrk(prk string) (string, error) {
	keyring, err := signature.KeyringPairFromSecret(prk, 0)
	if err != nil {
		return "", errors.Wrap(err, "[KeyringPairFromSecret]")
	}
	addr, err := tools.Encode(keyring.PublicKey, tools.SubstratePrefix)
	if err != nil {
		return "", errors.Wrap(err, "[Encode]")
	}
	return addr, nil
}
