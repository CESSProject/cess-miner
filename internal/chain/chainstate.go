package chain

import (
	"cess-bucket/configs"
	. "cess-bucket/internal/logger"
	"cess-bucket/tools"

	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

// Get storage miner information
func GetMinerInfo(prvkey string) (MinerInfo, int, error) {
	var (
		err   error
		mdata MinerInfo
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		if err := recover(); err != nil {
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

// Get all challenges
func GetChallenges(privkey string) ([]ChallengesInfo, int, error) {
	var (
		err  error
		data []ChallengesInfo
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		if err := recover(); err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()

	keyring, err := signature.KeyringPairFromSecret(privkey, 0)
	if err != nil {
		return nil, configs.Code_500, errors.Wrap(err, "[KeyringPairFromSecret]")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	key, err := types.CreateStorageKey(meta, State_SegmentBook, SegmentBook_ChallengeMap, keyring.PublicKey)
	if err != nil {
		return nil, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return nil, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return data, configs.Code_404, errors.New("Not found")
	}
	return data, configs.Code_200, nil
}

// get public key
func GetPublicKey() (Chain_SchedulerPuk, int, error) {
	var (
		err  error
		data Chain_SchedulerPuk
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		if err := recover(); err != nil {
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

// Get all invalid files
func GetInvalidFiles(privkey string) ([]types.Bytes, int, error) {
	var (
		err  error
		data []types.Bytes
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		if err := recover(); err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()

	keyring, err := signature.KeyringPairFromSecret(privkey, 0)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[KeyringPairFromSecret]")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	key, err := types.CreateStorageKey(meta, State_FileBank, FileBank_InvalidFile, keyring.PublicKey)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return data, configs.Code_404, errors.New("Not found")
	}
	return data, configs.Code_200, nil
}

// Get all scheduling nodes
func GetSchedulingNodes() ([]SchedulerInfo, int, error) {
	var (
		err   error
		mdata []SchedulerInfo
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		if err := recover(); err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
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
		return mdata, configs.Code_404, errors.New("Not found")
	}
	return mdata, configs.Code_200, nil
}

// Get the block height when the miner exits
func GetBlockHeightExited(prk string) (types.U32, int, error) {
	var (
		err    error
		number types.U32
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		if err := recover(); err != nil {
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
		return number, configs.Code_404, errors.New("Not found")
	}
	return number, configs.Code_200, nil
}

// Get the current block height
func GetBlockHeight() (types.U32, error) {
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		if err := recover(); err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	block, err := api.RPC.Chain.GetBlockLatest()
	if err != nil {
		return 0, errors.Wrap(err, "[GetBlockLatest]")
	}
	return types.U32(block.Block.Header.Number), nil
}

// Get the CESS chain account
func GetCESSAccount(prk string) (string, error) {
	keyring, err := signature.KeyringPairFromSecret(prk, 0)
	if err != nil {
		return "", errors.Wrap(err, "[KeyringPairFromSecret]")
	}
	addr, err := tools.Encode(keyring.PublicKey, tools.ChainCessTestPrefix)
	if err != nil {
		return "", errors.Wrap(err, "[Encode]")
	}
	return addr, nil
}

// Get account public key
func GetAccountPublickey(prk string) ([]byte, error) {
	keyring, err := signature.KeyringPairFromSecret(prk, 0)
	if err != nil {
		return nil, errors.Wrap(err, "[KeyringPairFromSecret]")
	}
	return keyring.PublicKey, nil
}
