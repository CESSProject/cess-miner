package chain

import (
	"cess-bucket/configs"
	. "cess-bucket/internal/logger"

	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

// Get miner information on the cess chain
// Get miner information on the chain
func GetMinerInfo(prvkey string) (MinerInfo, int, error) {
	var (
		err   error
		mdata MinerInfo
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()

	keyring, err := signature.KeyringPairFromSecret(prvkey, 0)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[KeyringPairFromSecret]")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	// b, err := types.EncodeToBytes(types.NewAccountID(pubkey))
	// if err != nil {
	// 	return mdata, configs.Code_500, errors.Wrap(err, "[EncodeToBytes]")
	// }

	key, err := types.CreateStorageKey(meta, State_Sminer, Sminer_MinerItems, keyring.PublicKey)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &mdata)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return mdata, configs.Code_404, errors.New("[value is empty]")
	}
	return mdata, configs.Code_200, nil
}

// Get scheduler information on the cess chain
func GetAllSchedulerInfo() (int, []SchedulerInfo, error) {
	var (
		err  error
		data []SchedulerInfo
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic] %v", err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return configs.Code_500, nil, errors.Wrap(err, "[GetMetadataLatest]")
	}

	key, err := types.CreateStorageKey(meta, State_FileMap, FileMap_SchedulerInfo)
	if err != nil {
		return configs.Code_500, nil, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return configs.Code_500, nil, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return configs.Code_404, data, errors.New("Not found")
	}
	return configs.Code_200, data, nil
}

func GetChallengesById(pubkey []byte) ([]ChallengesInfo, int, error) {
	var (
		err  error
		data []ChallengesInfo
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic] %v", err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	key, err := types.CreateStorageKey(meta, State_SegmentBook, SegmentBook_ChallengeMap, pubkey)
	if err != nil {
		return nil, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return nil, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return data, configs.Code_404, errors.New("value is empty")
	}
	return data, configs.Code_200, nil
}

//
func GetSchedulerPukFromChain() (Chain_SchedulerPuk, int, error) {
	var (
		err  error
		data Chain_SchedulerPuk
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	key, err := types.CreateStorageKey(meta, State_FileMap, FileMap_SchedulerPuk)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return data, configs.Code_404, errors.New("value is empty")
	}
	return data, configs.Code_200, nil
}

func GetInvalidFileById(id uint64) ([]types.Bytes, int, error) {
	var (
		err  error
		data []types.Bytes
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	b, err := types.EncodeToBytes(types.NewU64(id))
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	key, err := types.CreateStorageKey(meta, State_FileBank, FileBank_InvalidFile, b)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return data, configs.Code_404, errors.New("value is empty")
	}
	return data, configs.Code_200, nil
}

// Query Scheduler info
func GetSchedulerInfoOnChain() ([]SchedulerInfo, int, error) {
	var (
		err   error
		mdata []SchedulerInfo
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic] [%v.%v] [err:%v]", State_FileMap, FileMap_SchedulerInfo, err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	key, err := types.CreateStorageKey(meta, State_FileMap, FileMap_SchedulerInfo)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &mdata)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return mdata, configs.Code_404, errors.New("value is empty")
	}
	return mdata, configs.Code_200, nil
}

func GetFillerInfo(id types.U64, fileid string) (SpaceFileInfo, int, error) {
	var (
		err  error
		data SpaceFileInfo
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic] [%v.%v] [err:%v]", State_FileBank, FileBank_FillerMap, err)
		}
	}()

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	b, err := types.EncodeToBytes(id)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[EncodeToBytes]")
	}
	ids, err := types.EncodeToBytes(fileid)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[EncodeToBytes]")
	}
	key, err := types.CreateStorageKey(meta, State_FileBank, FileBank_FillerMap, b, ids)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return data, configs.Code_404, errors.New("value is empty")
	}
	return data, configs.Code_200, nil
}

// Get miner information on the cess chain
func GetMinerExitNumber(prk string) (types.U32, int, error) {
	var (
		err    error
		number types.U32
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return number, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	account, err := signature.KeyringPairFromSecret(prk, 0)
	if err != nil {
		return number, configs.Code_500, errors.Wrap(err, "[KeyringPairFromSecret]")
	}

	key, err := types.CreateStorageKey(meta, State_Sminer, Sminer_MinerColling, account.PublicKey)
	if err != nil {
		return number, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &number)
	if err != nil {
		return number, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return number, configs.Code_404, nil
	}
	return number, configs.Code_200, nil
}

// Get miner information on the cess chain
func GetLastNumber() (types.U32, error) {
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	block, err := api.RPC.Chain.GetBlockLatest()
	if err != nil {
		return 0, errors.Wrap(err, "[GetBlockLatest]")
	}

	return types.U32(block.Block.Header.Number), nil
}
