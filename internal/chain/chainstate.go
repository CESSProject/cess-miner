package chain

import (
	"cess-bucket/configs"
	. "cess-bucket/internal/logger"
	"encoding/binary"
	"fmt"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

// Get miner information on the cess chain
func GetMinerItems(phrase string) (Chain_MinerItems, int, error) {
	var (
		err   error
		mdata Chain_MinerItems
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
		return mdata, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	account, err := signature.KeyringPairFromSecret(phrase, 0)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[KeyringPairFromSecret]")
	}

	key, err := types.CreateStorageKey(meta, State_Sminer, Sminer_MinerItems, account.PublicKey)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &mdata)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return mdata, configs.Code_404, nil
	}
	return mdata, configs.Code_200, nil
}

// Get miner information on the cess chain
func GetMinerDetailInfo(identifyAccountPhrase, chainModule, chainModuleMethod1, chainModuleMethod2 string) (CessChain_MinerInfo, error) {
	var (
		err   error
		mdata CessChain_MinerInfo
		m1    Chain_MinerItems
		m2    CessChain_MinerInfo2
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

	mdata.MinerInfo1.Peerid = m1.Peerid
	mdata.MinerInfo1.Beneficiary = m1.Beneficiary
	mdata.MinerInfo1.ServiceAddr = m1.ServiceAddr
	mdata.MinerInfo1.Collaterals = m1.Collaterals
	mdata.MinerInfo1.Earnings = m1.Earnings
	mdata.MinerInfo1.Locked = m1.Locked
	mdata.MinerInfo1.State = m1.State

	mdata.MinerInfo2.Address = m2.Address
	mdata.MinerInfo2.Beneficiary = m2.Beneficiary
	mdata.MinerInfo2.Power = m2.Power
	mdata.MinerInfo2.Space = m2.Space
	mdata.MinerInfo2.Total_reward = m2.Total_reward
	mdata.MinerInfo2.Total_rewards_currently_available = m2.Total_rewards_currently_available
	mdata.MinerInfo2.Totald_not_receive = m2.Totald_not_receive
	mdata.MinerInfo2.Collaterals = m2.Collaterals

	return mdata, nil
}

// Get scheduler information on the cess chain
func GetSchedulerInfo() ([]SchedulerInfo, error) {
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
		return nil, errors.Wrapf(err, "[%v.%v:GetMetadataLatest]", State_FileMap, FileMap_SchedulerInfo)
	}

	key, err := types.CreateStorageKey(meta, State_FileMap, FileMap_SchedulerInfo)
	if err != nil {
		return nil, errors.Wrapf(err, "[%v.%v:CreateStorageKey]", State_FileMap, FileMap_SchedulerInfo)
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return nil, errors.Wrapf(err, "[%v.%v:GetStorageLatest]", State_FileMap, FileMap_SchedulerInfo)
	}
	if !ok {
		return data, errors.Errorf("[%v.%v:GetStorageLatest value is nil]", State_FileMap, FileMap_SchedulerInfo)
	}
	return data, nil
}

func GetChallengesById(id uint64) ([]ChallengesInfo, error) {
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
		return nil, errors.Wrap(err, "[GetMetadataLatest]")
	}
	b, err := types.EncodeToBytes(id)
	if err != nil {
		return nil, errors.Wrapf(err, "[EncodeToBytes]")
	}
	key, err := types.CreateStorageKey(meta, State_SegmentBook, SegmentBook_ChallengeMap, b)
	if err != nil {
		return nil, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return nil, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return data, errors.New("[value is nil]")
	}
	return data, nil
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
		return data, configs.Code_404, errors.New("public key not found")
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

	key, err := types.CreateStorageKey(meta, State_FileBank, FileBank_InvalidFile)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return data, configs.Code_404, errors.New("public key not found")
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

type FileMetaInfo struct {
	//FileId      types.Bytes         `json:"acc"`         //File id
	File_name   types.Bytes         `json:"file_name"`   //File name
	FileSize    types.U64           `json:"file_size"`   //File size
	FileHash    types.Bytes         `json:"file_hash"`   //File hash
	Public      types.Bool          `json:"public"`      //Public or not
	UserAddr    types.AccountID     `json:"user_addr"`   //Upload user's address
	FileState   types.Bytes         `json:"file_state"`  //File state
	Backups     types.U8            `json:"backups"`     //Number of backups
	Downloadfee types.U128          `json:"downloadfee"` //Download fee
	FileDupl    []FileDuplicateInfo `json:"file_dupl"`   //File backup information list
}

type FileDuplicateInfo struct {
	MinerId   types.U64
	BlockNum  types.U32
	ScanSize  types.U32
	Acc       types.AccountID
	MinerIp   types.Bytes
	DuplId    types.Bytes
	RandKey   types.Bytes
	BlockInfo []BlockInfo
}

// Get miner information on the cess chain
func ChainSt_Test(rpcaddr, signaturePrk, pallert, method string) error {
	var (
		err   error
		mdata FileMetaInfo
	)
	api, err := gsrpc.NewSubstrateAPI(rpcaddr)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		return err
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return errors.Wrap(err, "[GetMetadataLatest]")
	}

	key, err := types.CreateStorageKey(meta, pallert, method)
	if err != nil {
		return errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &mdata)
	if err != nil {
		return errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return errors.New("empty")
	}
	fmt.Println(mdata)
	return nil
}
