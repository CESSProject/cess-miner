package chain

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	"cess-bucket/configs"
	. "cess-bucket/internal/logger"
	"cess-bucket/tools"
	"time"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

// miner register
func RegisterBucketToChain(signaturePrk, imcodeAcc, ipAddr string, pledgeTokens uint64, authPuk []byte) (string, int, error) {
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

	keyring, err := signature.KeyringPairFromSecret(signaturePrk, 0)
	if err != nil {
		return "", configs.Code_400, errors.Wrap(err, "[KeyringPairFromSecret]")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}
	var pre []byte
	if configs.NewTestAddr {
		pre = tools.ChainCessTestPrefix
	} else {
		pre = tools.SubstratePrefix
	}
	b, err := tools.DecodeToPub(imcodeAcc, pre)
	if err != nil {
		return "", configs.Code_400, errors.Wrap(err, "[DecodeToPub]")
	}

	pTokens := strconv.FormatUint(pledgeTokens, 10)
	pTokens += configs.TokenAccuracy
	realTokens, ok := new(big.Int).SetString(pTokens, 10)
	if !ok {
		return "", configs.Code_500, errors.New("[big.Int.SetString]")
	}

	c, err := types.NewCall(meta, ChainTx_Sminer_Register, types.NewAccountID(b), types.Bytes([]byte(ipAddr)), types.NewU128(*realTokens), types.Bytes(authPuk))
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "[NewCall]")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "[NewExtrinsic]")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "[GetBlockHash]")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "[GetRuntimeVersionLatest]")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "[CreateStorageKey System Account]")
	}

	keye, err := types.CreateStorageKey(meta, "System", "Events", nil)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "[CreateStorageKey System Events]")
	}

	ok, err = api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return "", configs.Code_500, errors.New("[GetStorageLatest return value is empty]")
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
		return "", configs.Code_500, errors.Wrap(err, "[Sign]")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return "", configs.Code_500, errors.Wrap(err, "[SubmitAndWatchExtrinsic]")
	}
	defer sub.Unsubscribe()
	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := MyEventRecords{}
				txhash := fmt.Sprintf("%#x", status.AsInBlock)
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					return txhash, configs.Code_600, err
				}
				types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
				if events.Sminer_Registered != nil {
					for i := 0; i < len(events.Sminer_Registered); i++ {
						if events.Sminer_Registered[i].Acc == types.NewAccountID(keyring.PublicKey) {
							return txhash, configs.Code_200, nil
						}
					}
					return txhash, configs.Code_600, errors.Errorf("events.Sminer_Registered data err")
				}
				return txhash, configs.Code_600, errors.Errorf("events.Sminer_Registered not found")
			}
		case err = <-sub.Err():
			return "", configs.Code_500, err
		case <-timeout:
			return "", configs.Code_500, errors.New("Timeout")
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
				case ChainTx_SegmentBook_SubmitToVpa:
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
				case ChainTx_SegmentBook_SubmitToVpb:
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
func GetAddressFromPrk(prk string, prefix []byte) (string, error) {
	keyring, err := signature.KeyringPairFromSecret(prk, 0)
	if err != nil {
		return "", errors.Wrap(err, "[KeyringPairFromSecret]")
	}
	var pre []byte
	if configs.NewTestAddr {
		pre = tools.ChainCessTestPrefix
	} else {
		pre = tools.SubstratePrefix
	}
	addr, err := tools.Encode(keyring.PublicKey, pre)
	if err != nil {
		return "", errors.Wrap(err, "[Encode]")
	}
	return addr, nil
}

//
func PutProofToChain(signaturePrk string, id uint64, fid, sigma []byte, mu [][]byte) (int, error) {
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

	keyring, err := signature.KeyringPairFromSecret(signaturePrk, 0)
	if err != nil {
		return configs.Code_400, errors.Wrap(err, "[KeyringPairFromSecret]")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	// b, err := types.EncodeToBytes(id)
	// if err != nil {
	// 	return configs.Code_400, errors.Wrap(err, "[EncodeToBytes]")
	// }

	var mus []types.Bytes = make([]types.Bytes, len(mu))
	for i := 0; i < len(mu); i++ {
		mus[i] = make(types.Bytes, 0)
		mus[i] = append(mus[i], mu[i]...)
	}

	c, err := types.NewCall(meta, SegmentBook_SubmitProve, types.U64(id), types.Bytes(fid), mus, types.Bytes(sigma))
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[NewCall]")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[NewExtrinsic]")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[GetBlockHash]")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[GetRuntimeVersionLatest]")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[CreateStorageKey System Account]")
	}

	keye, err := types.CreateStorageKey(meta, "System", "Events", nil)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[CreateStorageKey System Events]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return configs.Code_500, errors.New("[GetStorageLatest return value is empty]")
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
		return configs.Code_500, errors.Wrap(err, "[Sign]")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[SubmitAndWatchExtrinsic]")
	}
	defer sub.Unsubscribe()
	var head *types.Header
	t := tools.RandomInRange(10000000, 99999999)
	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := MyEventRecords{}
				head, err = api.RPC.Chain.GetHeader(status.AsInBlock)
				if err == nil {
					Out.Sugar().Infof("[T:%v] [BN:%v]", t, head.Number)
				} else {
					Out.Sugar().Infof("[T:%v] [BH:%#x]", t, status.AsInBlock)
				}
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					return configs.Code_600, errors.Wrapf(err, "[T:%v]", t)
				}
				err = types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
				if err != nil {
					Out.Sugar().Infof("[T:%v]Decode event err:%v", t, err)
				}
				if events.SegmentBook_ChallengeProof != nil {
					for i := 0; i < len(events.SegmentBook_ChallengeProof); i++ {
						if events.SegmentBook_ChallengeProof[i].PeerId == types.U64(id) {
							Out.Sugar().Infof("[T:%v] Submit prove success", t)
							return configs.Code_200, nil
						}
					}
					return configs.Code_600, errors.Errorf("[T:%v] events.SegmentBook_SubmitProve data err", t)
				}
				return configs.Code_600, errors.Errorf("[T:%v] events.SegmentBook_SubmitProve not found", t)
			}
		case err = <-sub.Err():
			return configs.Code_500, err
		case <-timeout:
			return configs.Code_500, errors.New("Timeout")
		}
	}
}

//
func ClearInvalidFileNoChain(signaturePrk string, id uint64, fid types.Bytes) (int, error) {
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

	keyring, err := signature.KeyringPairFromSecret(signaturePrk, 0)
	if err != nil {
		return configs.Code_400, errors.Wrap(err, "[KeyringPairFromSecret]")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	b, err := types.EncodeToBytes(id)
	if err != nil {
		return configs.Code_400, errors.Wrap(err, "[EncodeToBytes]")
	}

	c, err := types.NewCall(meta, FileBank_ClearInvalidFile, b, fid)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[NewCall]")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[NewExtrinsic]")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[GetBlockHash]")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[GetRuntimeVersionLatest]")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[CreateStorageKey System Account]")
	}

	keye, err := types.CreateStorageKey(meta, "System", "Events", nil)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[CreateStorageKey System Events]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return configs.Code_500, errors.New("[GetStorageLatest return value is empty]")
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
		return configs.Code_500, errors.Wrap(err, "[Sign]")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[SubmitAndWatchExtrinsic]")
	}
	defer sub.Unsubscribe()
	var head *types.Header
	t := tools.RandomInRange(10000000, 99999999)
	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := MyEventRecords{}
				head, err = api.RPC.Chain.GetHeader(status.AsInBlock)
				if err == nil {
					Out.Sugar().Infof("[T:%v] [BN:%v]", t, head.Number)
				} else {
					Out.Sugar().Infof("[T:%v] [BH:%#x]", t, status.AsInBlock)
				}
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					return configs.Code_600, errors.Wrapf(err, "[T:%v]", t)
				}
				err = types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)
				if err != nil {
					Out.Sugar().Infof("[T:%v]Decode event err:%v", t, err)
				}
				if events.FileBank_ClearInvalidFile != nil {
					for i := 0; i < len(events.FileBank_ClearInvalidFile); i++ {
						if events.FileBank_ClearInvalidFile[i].Acc == types.NewAccountID(keyring.PublicKey) {
							Out.Sugar().Infof("[T:%v] Submit prove success", t)
							return configs.Code_200, nil
						}
					}
					return configs.Code_600, errors.Errorf("[T:%v] events.FileBank_ClearInvalidFile data err", t)
				}
				return configs.Code_600, errors.Errorf("[T:%v] events.FileBank_ClearInvalidFile not found", t)
			}
		case err = <-sub.Err():
			return configs.Code_500, err
		case <-timeout:
			return configs.Code_500, errors.New("Timeout")
		}
	}
}

//
func ChainTx_Test(rpcaddr, signaturePrk, pallert_method string) error {
	var (
		err         error
		accountInfo types.AccountInfo
	)
	api, err := gsrpc.NewSubstrateAPI(rpcaddr)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		return err
	}

	keyring, err := signature.KeyringPairFromSecret(signaturePrk, 0)
	if err != nil {
		return errors.Wrap(err, "[KeyringPairFromSecret]")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return errors.Wrap(err, "[GetMetadataLatest]")
	}

	// b, err := types.EncodeToBytes(id)
	// if err != nil {
	// 	return configs.Code_400, errors.Wrap(err, "[EncodeToBytes]")
	// }

	// var mus []types.Bytes = make([]types.Bytes, len(mu))
	// for i := 0; i < len(mu); i++ {
	// 	mus[i] = make(types.Bytes, 0)
	// 	mus[i] = append(mus[i], mu[i]...)
	// }
	c, err := types.NewCall(meta, pallert_method)
	if err != nil {
		return errors.Wrap(err, "[NewCall]")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return errors.Wrap(err, "[NewExtrinsic]")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return errors.Wrap(err, "[GetBlockHash]")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return errors.Wrap(err, "[GetRuntimeVersionLatest]")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return errors.Wrap(err, "[CreateStorageKey System Account]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return errors.New("[GetStorageLatest return value is empty]")
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
		return errors.Wrap(err, "[Sign]")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return errors.Wrap(err, "[SubmitAndWatchExtrinsic]")
	}
	defer sub.Unsubscribe()
	var head *types.Header
	timeout := time.After(time.Second * 15)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				fmt.Println("Block hash: %#v", status.AsInBlock)
				head, err = api.RPC.Chain.GetHeader(status.AsInBlock)
				if err == nil {
					fmt.Println("[Block number: %v]", head.Number)
				}
			}
		case err = <-sub.Err():
			return err
		case <-timeout:
			return errors.New("Timeout")
		}
	}
}
