package chain

import (
	"cess-bucket/configs"
	. "cess-bucket/internal/logger"
	"cess-bucket/tools"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

// Get storage miner information
func GetMinerInfo(api *gsrpc.SubstrateAPI) (MinerInfo, error) {
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()

	var err error
	var data MinerInfo

	if api == nil {
		api, err = GetRpcClient_Safe(configs.C.RpcAddr)
		defer Free()
		if err != nil {
			return data, errors.Wrap(err, "[GetRpcClient_Safe]")
		}
	}

	meta, err := GetMetadata(api)
	if err != nil {
		return data, errors.Wrap(err, "[GetMetadata]")
	}

	key, err := types.CreateStorageKey(meta, State_Sminer, Sminer_MinerItems, configs.PublicKey)
	if err != nil {
		return data, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return data, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return data, errors.New(ERR_Empty)
	}
	return data, nil
}

// Get all challenges
func GetChallenges() ([]ChallengesInfo, error) {
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()

	var data []ChallengesInfo

	api, err := GetRpcClient_Safe(configs.C.RpcAddr)
	defer Free()
	if err != nil {
		return nil, errors.Wrap(err, "[GetRpcClient_Safe]")
	}

	meta, err := GetMetadata(api)
	if err != nil {
		return data, errors.Wrap(err, "[GetMetadata]")
	}

	key, err := types.CreateStorageKey(meta, State_SegmentBook, SegmentBook_ChallengeMap, configs.PublicKey)
	if err != nil {
		return nil, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return nil, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return nil, errors.New(ERR_Empty)
	}
	return data, nil
}

// get public key
func GetSchedulerPublicKey() (Chain_SchedulerPuk, error) {
	var data Chain_SchedulerPuk

	api, err := GetRpcClient_Safe(configs.C.RpcAddr)
	defer Free()
	if err != nil {
		return data, errors.Wrap(err, "[GetRpcClient_Safe]")
	}

	meta, err := GetMetadata(api)
	if err != nil {
		return data, errors.Wrap(err, "[GetMetadata]")
	}

	key, err := types.CreateStorageKey(meta, State_FileMap, FileMap_SchedulerPuk)
	if err != nil {
		return data, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return data, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return data, errors.New(ERR_Empty)
	}
	return data, nil
}

// Get all invalid files
func GetInvalidFiles() ([]types.Bytes, error) {
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()

	var data []types.Bytes

	api, err := GetRpcClient_Safe(configs.C.RpcAddr)
	defer Free()
	if err != nil {
		return nil, errors.Wrap(err, "[GetRpcClient_Safe]")
	}

	meta, err := GetMetadata(api)
	if err != nil {
		return nil, errors.Wrap(err, "[GetMetadata]")
	}

	key, err := types.CreateStorageKey(meta, State_FileBank, FileBank_InvalidFile, configs.PublicKey)
	if err != nil {
		return nil, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return nil, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return nil, errors.New(ERR_Empty)
	}
	return data, nil
}

// Get all scheduling nodes
func GetSchedulingNodes() ([]SchedulerInfo, error) {
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()

	var data []SchedulerInfo

	api, err := GetRpcClient_Safe(configs.C.RpcAddr)
	defer Free()
	if err != nil {
		return nil, errors.Wrap(err, "[GetRpcClient_Safe]")
	}

	meta, err := GetMetadata(api)
	if err != nil {
		return nil, errors.Wrap(err, "[GetMetadata]")
	}

	key, err := types.CreateStorageKey(meta, State_FileMap, FileMap_SchedulerInfo)
	if err != nil {
		return nil, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return nil, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return nil, errors.New(ERR_Empty)
	}
	return data, nil
}

// Get the block height when the miner exits
func GetBlockHeightExited(api *gsrpc.SubstrateAPI) (types.U32, error) {
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()

	var (
		err    error
		number types.U32
	)

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return number, errors.Wrap(err, "[GetMetadataLatest]")
	}

	key, err := types.CreateStorageKey(meta, State_Sminer, Sminer_MinerColling, configs.PublicKey)
	if err != nil {
		return number, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &number)
	if err != nil {
		return number, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return number, errors.New(ERR_Empty)
	}
	return number, nil
}

// Get the current block height
func GetBlockHeight(api *gsrpc.SubstrateAPI) (types.U32, error) {
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
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
	addr, err := tools.Encode(configs.PublicKey, tools.ChainCessTestPrefix)
	if err != nil {
		return "", errors.Wrap(err, "[Encode]")
	}
	return addr, nil
}
