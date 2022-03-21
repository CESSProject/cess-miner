package chain

import (
	"encoding/binary"
	"fmt"
	"storage-mining/internal/logger"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

type CessChain_MinerInfo struct {
	CessChain_MinerInfo1
	CessChain_MinerInfo2
}

type CessChain_MinerInfo1 struct {
	Peerid       types.U64       `json:"peerid"`
	Beneficiary1 types.AccountID `json:"beneficiary"`
	ServiceAddr  types.Bytes     `json:"ip"`
	Collaterals1 types.U128      `json:"collaterals"`
	Earnings     types.U128      `json:"earnings"`
	Locked       types.U128      `json:"locked"`
	State        types.Bytes     `json:"state"`
}

type CessChain_MinerInfo2 struct {
	Address                           types.AccountID `json:"address"`
	Beneficiary2                      types.AccountID `json:"beneficiary"`
	Power                             types.U128      `json:"power"`
	Space                             types.U128      `json:"space"`
	Total_reward                      types.U128      `json:"total_reward"`
	Total_rewards_currently_available types.U128      `json:"total_rewards_currently_available"`
	Totald_not_receive                types.U128      `json:"totald_not_receive"`
	Collaterals2                      types.U128      `json:"collaterals"`
}

type ParamInfo struct {
	Peer_id    types.U64 `json:"peer_id"`
	Segment_id types.U64 `json:"segment_id"`
	Rand       types.U32 `json:"rand"`
}

type IpostParaInfo struct {
	Peer_id    types.U64   `json:"peer_id"`
	Segment_id types.U64   `json:"segment_id"`
	Sealed_cid types.Bytes `json:"sealed_cid"`
	Size_type  types.U128  `json:"size_type"`
}

type UnsealedCidInfo struct {
	Peer_id    types.U64     `json:"peer_id"`
	Segment_id types.U64     `json:"segment_id"`
	Uncid      []types.Bytes `json:"uncid"`
	Rand       types.U32     `json:"rand"`
	Shardhash  types.Bytes   `json:"shardhash"`
}

type FpostParaInfo struct {
	Peer_id    types.U64     `json:"peer_id"`
	Segment_id types.U64     `json:"segment_id"`
	Sealed_cid []types.Bytes `json:"sealed_cid"`
	Hash       types.Bytes   `json:"hash"`
	Size_type  types.U128    `json:"size_type"`
}

// Get miner information on the cess chain
func GetMinerInfo1(identifyAccountPhrase, chainModule, chainModuleMethod string) (CessChain_MinerInfo1, error) {
	var (
		err   error
		mdata CessChain_MinerInfo1
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return mdata, errors.Wrap(err, "GetMetadataLatest err")
	}

	account, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return mdata, errors.Wrap(err, "KeyringPairFromSecret err")
	}

	key, err := types.CreateStorageKey(meta, chainModule, chainModuleMethod, account.PublicKey)
	if err != nil {
		return mdata, errors.Wrap(err, "CreateStorageKey err")
	}

	_, err = api.RPC.State.GetStorageLatest(key, &mdata)
	if err != nil {
		return mdata, errors.Wrap(err, "GetStorageLatest err")
	}
	return mdata, nil
}

// Get miner information on the cess chain
func GetMinerDetailInfo(identifyAccountPhrase, chainModule, chainModuleMethod1, chainModuleMethod2 string) (CessChain_MinerInfo, error) {
	var (
		err   error
		mdata CessChain_MinerInfo
		m1    CessChain_MinerInfo1
		m2    CessChain_MinerInfo2
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return mdata, errors.Wrap(err, "GetMetadataLatest err")
	}

	account, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return mdata, errors.Wrap(err, "KeyringPairFromSecret err")
	}

	key, err := types.CreateStorageKey(meta, chainModule, chainModuleMethod1, account.PublicKey)
	if err != nil {
		return mdata, errors.Wrap(err, "CreateStorageKey err")
	}

	_, err = api.RPC.State.GetStorageLatest(key, &m1)
	if err != nil {
		return mdata, errors.Wrap(err, "GetStorageLatest err")
	}

	eraIndexSerialized := make([]byte, 8)
	binary.LittleEndian.PutUint64(eraIndexSerialized, uint64(m1.Peerid))

	key, err = types.CreateStorageKey(meta, chainModule, chainModuleMethod2, types.NewBytes(eraIndexSerialized))
	if err != nil {
		return mdata, errors.Wrap(err, "CreateStorageKey err")
	}

	_, err = api.RPC.State.GetStorageLatest(key, &m2)
	if err != nil {
		return mdata, errors.Wrap(err, "GetStorageLatest err")
	}

	mdata.Peerid = m1.Peerid
	mdata.Beneficiary1 = m1.Beneficiary1
	mdata.ServiceAddr = m1.ServiceAddr
	mdata.Collaterals1 = m1.Collaterals1
	mdata.Earnings = m1.Earnings
	mdata.Locked = m1.Locked
	mdata.State = m1.State

	mdata.Address = m2.Address
	mdata.Beneficiary2 = m2.Beneficiary2
	mdata.Power = m2.Power
	mdata.Space = m2.Space
	mdata.Total_reward = m2.Total_reward
	mdata.Total_rewards_currently_available = m2.Total_rewards_currently_available
	mdata.Totald_not_receive = m2.Totald_not_receive
	mdata.Collaterals2 = m2.Collaterals2

	return mdata, nil
}

// Get seed number on the cess chain
func GetSeedNumOnChain(identifyAccountPhrase, chainModule, chainModuleMethod string) (ParamInfo, error) {
	var (
		err       error
		ok        bool
		paramdata ParamInfo
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return paramdata, errors.Wrap(err, "GetMetadataLatest err")
	}

	account, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return paramdata, errors.Wrap(err, "KeyringPairFromSecret err")
	}

	key, err := types.CreateStorageKey(meta, chainModule, chainModuleMethod, account.PublicKey)
	if err != nil {
		return paramdata, errors.Wrap(err, "CreateStorageKey err")
	}

	ok, err = api.RPC.State.GetStorageLatest(key, &paramdata)
	if err != nil {
		return paramdata, errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return paramdata, errors.New("paramdata data is empty")
	}
	return paramdata, nil
}

// Get vpa post on the cess chain
func GetVpaPostOnChain(identifyAccountPhrase, chainModule, chainModuleMethod string) ([]IpostParaInfo, error) {
	var (
		err       error
		paramdata []IpostParaInfo
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return paramdata, errors.Wrap(err, "GetMetadataLatest err")
	}

	account, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return paramdata, errors.Wrap(err, "KeyringPairFromSecret err")
	}

	key, err := types.CreateStorageKey(meta, chainModule, chainModuleMethod, account.PublicKey)
	if err != nil {
		return paramdata, errors.Wrap(err, "CreateStorageKey err")
	}

	_, err = api.RPC.State.GetStorageLatest(key, &paramdata)
	if err != nil {
		return paramdata, errors.Wrap(err, "GetStorageLatest err")
	}
	return paramdata, nil
}

// Get unsealcid on the cess chain
func GetunsealcidOnChain(identifyAccountPhrase, chainModule, chainModuleMethod string) ([]UnsealedCidInfo, error) {
	var (
		err       error
		paramdata []UnsealedCidInfo
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return paramdata, errors.Wrap(err, "GetMetadataLatest err")
	}

	account, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return paramdata, errors.Wrap(err, "KeyringPairFromSecret err")
	}

	key, err := types.CreateStorageKey(meta, chainModule, chainModuleMethod, account.PublicKey)
	if err != nil {
		return paramdata, errors.Wrap(err, "CreateStorageKey err")
	}

	_, err = api.RPC.State.GetStorageLatest(key, &paramdata)
	if err != nil {
		return paramdata, errors.Wrap(err, "GetStorageLatest err")
	}
	return paramdata, nil
}

// Get vpc post on the cess chain
func GetVpcPostOnChain(identifyAccountPhrase, chainModule, chainModuleMethod string) ([]FpostParaInfo, error) {
	var (
		err       error
		paramdata []FpostParaInfo
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return paramdata, errors.Wrap(err, "GetMetadataLatest err")
	}

	account, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return paramdata, errors.Wrap(err, "KeyringPairFromSecret err")
	}

	key, err := types.CreateStorageKey(meta, chainModule, chainModuleMethod, account.PublicKey)
	if err != nil {
		return paramdata, errors.Wrap(err, "CreateStorageKey err")
	}

	_, err = api.RPC.State.GetStorageLatest(key, &paramdata)
	if err != nil {
		return paramdata, errors.Wrap(err, "GetStorageLatest err")
	}
	return paramdata, nil
}

// Renewal tokens
func RenewalTokens() error {
	//TODO:
	return errors.New("test")
}

//not use
func GetLatestBlockHeight() {
	api, err := gsrpc.NewSubstrateAPI("ws://106.15.44.155:9947")
	if err != nil {
		panic(err)
	}
	sub, err := api.RPC.Chain.SubscribeNewHeads()
	if err != nil {
		panic(err)
	}
	defer sub.Unsubscribe()

	count := 0

	for {
		head := <-sub.Chan()
		fmt.Printf("Chain is at block: #%v\n", head.Number)
		count++
		if count == 10 {
			sub.Unsubscribe()
			break
		}
	}
}
