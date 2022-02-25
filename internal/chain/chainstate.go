package chain

import (
	"encoding/binary"
	"fmt"
	"storage-mining/internal/logger"
	"time"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/minio/blake2b-simd"
	"github.com/pkg/errors"
)

type CessChain_MinerItems struct {
	Peerid      types.U64       `json:"peerid"`
	Beneficiary types.AccountID `json:"beneficiary"`
	Ip          types.U32       `json:"ip"`
	Collaterals types.U128      `json:"collaterals"`
	Earnings    types.U128      `json:"earnings"`
	Locked      types.U128      `json:"locked"`
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
	Hash       types.Bytes   `json:"hash"`
	Shardhash  types.Bytes   `json:"shardhash"`
}

type FpostParaInfo struct {
	Peer_id    types.U64     `json:"peer_id"`
	Segment_id types.U64     `json:"segment_id"`
	Sealed_cid []types.Bytes `json:"sealed_cid"`
	Hash       types.Bytes   `json:"hash"`
	Size_type  types.U128    `json:"size_type"`
}

type ProofInfoPPAInfo struct {
	Size_type  types.U32       `json:"size_type"`
	Proof      types.Bytes     `json:"proof"`
	Sealed_cid types.Bytes     `json:"sealed_cid"`
	Block_num  types.OptionU64 `json:"block_num"`
}

type FileInfo struct {
	Filename       types.Bytes     `json:"filename"`
	Owner          types.AccountID `json:"owner"`
	Filehash       types.Bytes     `json:"filehash"`
	Similarityhash types.Bytes     `json:"similarityhash"`
	Ispublic       types.U8        `json:"ispublic"`
	Backups        types.U8        `json:"backups"`
	Creator        types.Bytes     `json:"creator"`
	Filesize       types.U128      `json:"filesize"`
	Keywords       types.Bytes     `json:"keywords"`
	Email          types.Bytes     `json:"email"`
	Uploadfee      types.U128      `json:"uploadfee"`
	Downloadfee    types.U128      `json:"downloadfee"`
	Deadline       types.U128      `json:"deadline"`
}


// Get miner information on the cess chain
func GetMinerDataOnChain(identifyAccountPhrase, chainModule, chainModuleMethod string) (CessChain_MinerItems, error) {
	var (
		err   error
		mdata CessChain_MinerItems
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

// Get vpa post on the cess chain
func GetFileInfoOnChain() (FileInfo, error) {
	var (
		err       error
		paramdata FileInfo
	)
	//paramdata.Sealed_cid = make([]types.OptionBytes, 0)
	//paramdata.Proof = make([]types.OptionBytes, 0)
	api, err := gsrpc.NewSubstrateAPI("ws://106.15.44.155:9947")
	if err != nil {
		panic(err)
	}
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return paramdata, errors.Wrap(err, "GetMetadataLatest err")
	}

	// account, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	// if err != nil {
	// 	return paramdata, errors.Wrap(err, "KeyringPairFromSecret err")
	// }

	eraIndexSerialized := make([]byte, 8)
	binary.LittleEndian.PutUint64(eraIndexSerialized, uint64(1832))

	// t := types.NewOptionU64Empty()
	// t.SetNone()
	// b, err := types.EncodeToBytes(t)
	// if err != nil {
	// 	panic(err)
	// }
	//bbb := HexToBytes("9395eef17d20e2a74edb87be3f3c319345a7f317fa561ad31832d0b8755036ca")
	b, err := types.EncodeToBytes("9395eef17d20e2a74edb87be3f3c319345a7f317fa561ad31832d0b8755036ca")
	if err != nil {
		panic(err)
	}
	key, err := types.CreateStorageKey(meta, "FileBank", "File", types.NewBytes(b))
	if err != nil {
		return paramdata, errors.Wrap(err, "CreateStorageKey err")
	}
	_, err = api.RPC.State.GetStorageLatest(key, &paramdata)
	if err != nil {
		return paramdata, errors.Wrap(err, "GetStorageLatest err")
	}
	//fmt.Println(types.NewAddressFromAccountID(paramdata.Owner[:]))
	return paramdata, nil
}

// Get vpa post on the cess chain
func GetDoubleMapOnChain() ([]ProofInfoPPAInfo, error) {
	var (
		err       error
		paramdata []ProofInfoPPAInfo
	)

	api, err := gsrpc.NewSubstrateAPI("ws://106.15.44.155:9947")
	if err != nil {
		panic(err)
	}
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return paramdata, errors.Wrap(err, "GetMetadataLatest err")
	}

	account, err := signature.KeyringPairFromSecret("repair high another sell behave clock when auction tortoise real track cupboard", 0)
	if err != nil {
		return paramdata, errors.Wrap(err, "KeyringPairFromSecret err")
	}

	eraIndexSerialized := make([]byte, 8)
	binary.LittleEndian.PutUint64(eraIndexSerialized, uint64(1832))

	// t := types.NewOptionU64Empty()
	// t.SetNone()
	// b, err := types.EncodeToBytes(t)
	// if err != nil {
	// 	panic(err)
	// }

	// bbb2, err := types.EncodeToBytes("9395eef17d20e2a74edb87be3f3c319345a7f317fa561ad31832d0b8755036ca")
	// if err != nil {
	// 	panic(err)
	// }
	// bbb2, err := types.EncodeToBytes(types.NewOptionU64Empty())
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(bbb2)
	// _ = bbb2

	// var buffer = bytes.Buffer{}
	// var encoder = *scale.NewEncoder(&buffer)
	// types.NewOptionU64Empty().Encode(encoder)

	bys, err := types.EncodeToBytes(types.NewOptionU64(0))
	if err != nil {
		panic(err)
	}
	key, err := types.CreateStorageKey(meta, "SegmentBook", "PrePoolA", account.PublicKey, bys)
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
