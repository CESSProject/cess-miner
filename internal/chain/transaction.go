package chain

import (
	"math/big"
	"strconv"
	"strings"

	"time"

	"github.com/CESSProject/cess-bucket/configs"
	. "github.com/CESSProject/cess-bucket/internal/logger"
	"github.com/CESSProject/cess-bucket/internal/pattern"
	"github.com/CESSProject/cess-bucket/tools"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"
	"github.com/pkg/errors"
)

// Storage Miner Registration Function
func Register(api *gsrpc.SubstrateAPI, incomeAcc, ip string, port uint16, pledgeTokens uint64) (string, error) {
	defer func() {
		if err := recover(); err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()

	var (
		err         error
		txhash      string
		accountInfo types.AccountInfo
	)
	if api == nil {
		api, err = GetRpcClient_Safe(configs.C.RpcAddr)
		defer Free()
		if err != nil {
			return txhash, errors.Wrap(err, "[GetRpcClient_Safe]")
		}
	}

	meta, err := GetMetadata(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetMetadataLatest]")
	}

	b, err := tools.DecodeToCessPub(incomeAcc)
	if err != nil {
		return txhash, errors.Wrap(err, "[DecodeToPub]")
	}

	pTokens := strconv.FormatUint(pledgeTokens, 10)
	pTokens += configs.TokenAccuracy
	realTokens, ok := new(big.Int).SetString(pTokens, 10)
	if !ok {
		return txhash, errors.New("[big.Int.SetString]")
	}

	var ipType IpAddress

	if tools.IsIPv4(ip) {
		ipType.IPv4.Index = 0
		ips := strings.Split(ip, ".")
		for i := 0; i < 4; i++ {
			temp, _ := strconv.Atoi(ips[i])
			ipType.IPv4.Value[i] = types.U8(temp)
		}
		ipType.IPv4.Port = types.U16(port)
	} else {
		return txhash, errors.New("unsupported ip format")
	}

	acc, err := types.NewAccountID(b)
	if err != nil {
		return txhash, errors.Wrap(err, "[NewAccountID]")
	}

	c, err := types.NewCall(
		meta,
		TX_SMINER_REG,
		*acc,
		ipType.IPv4,
		types.NewU128(*realTokens),
	)
	if err != nil {
		return txhash, errors.Wrap(err, "[NewCall]")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return txhash, errors.Wrap(err, "[NewExtrinsic]")
	}

	genesisHash, err := GetGenesisHash(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetGenesisHash]")
	}

	rv, err := GetRuntimeVersion(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetRuntimeVersion]")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", pattern.GetMinerAcc())
	if err != nil {
		return txhash, errors.Wrap(err, "[CreateStorageKey System Account]")
	}

	ok, err = api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetStorageLatest]")
	}

	if !ok {
		return txhash, errors.New(ERR_Empty)
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

	kring, err := GetKeyring(configs.C.SignatureAcc)
	if err != nil {
		return txhash, errors.Wrap(err, "GetKeyring")
	}

	// Sign the transaction
	err = ext.Sign(kring, o)
	if err != nil {
		return txhash, errors.Wrap(err, "[Sign]")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return txhash, errors.Wrap(err, "[SubmitAndWatchExtrinsic]")
	}
	defer sub.Unsubscribe()
	timeout := time.After(configs.TimeToWaitEvents)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := EventRecords{}
				txhash, _ = codec.EncodeToHex(status.AsInBlock)
				keye, err := GetKeyEvents()
				if err != nil {
					return txhash, errors.Wrap(err, "GetKeyEvents")
				}
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					return txhash, errors.Wrap(err, "GetStorageRaw")
				}

				types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)

				if len(events.Sminer_Registered) > 0 {
					return txhash, nil
				}
				return txhash, errors.New(ERR_Failed)
			}
		case err = <-sub.Err():
			return txhash, errors.Wrap(err, "<-sub")
		case <-timeout:
			return txhash, errors.New(ERR_Timeout)
		}
	}
}

// Storage miners increase deposit function
func Increase(api *gsrpc.SubstrateAPI, identifyAccountPhrase, TransactionName string, tokens *big.Int) (string, error) {

	defer func() {
		if err := recover(); err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()

	var (
		txhash      string
		accountInfo types.AccountInfo
	)

	keyring, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return txhash, errors.Wrap(err, "KeyringPairFromSecret")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return txhash, errors.Wrap(err, "GetMetadataLatest")
	}

	c, err := types.NewCall(meta, TransactionName, types.NewUCompact(tokens))
	if err != nil {
		return txhash, errors.Wrap(err, "NewCall")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return txhash, errors.Wrap(err, "NewExtrinsic")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return txhash, errors.Wrap(err, "GetBlockHash")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return txhash, errors.Wrap(err, "GetRuntimeVersionLatest")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return txhash, errors.Wrap(err, "CreateStorageKey")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return txhash, errors.Wrap(err, "GetStorageLatest")
	}
	if !ok {
		return txhash, errors.New("GetStorageLatest return value is empty")
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
		return txhash, errors.Wrap(err, "Sign")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return txhash, errors.Wrap(err, "SubmitAndWatchExtrinsic")
	}
	defer sub.Unsubscribe()

	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				txhash, _ = codec.EncodeToHex(status.AsInBlock)
				return txhash, nil
			}
		case err = <-sub.Err():
			return txhash, err
		case <-timeout:
			return txhash, errors.New("timeout")
		}
	}
}

// Storage miner exits the mining function
func ExitMining(api *gsrpc.SubstrateAPI, identifyAccountPhrase, TransactionName string) (string, error) {
	defer func() {
		if err := recover(); err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	var (
		txhash      string
		accountInfo types.AccountInfo
	)
	keyring, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return txhash, errors.Wrap(err, "KeyringPairFromSecret")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return txhash, errors.Wrap(err, "GetMetadataLatest")
	}

	c, err := types.NewCall(meta, TransactionName)
	if err != nil {
		return txhash, errors.Wrap(err, "NewCall")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return txhash, errors.Wrap(err, "NewExtrinsic")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return txhash, errors.Wrap(err, "GetBlockHash")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return txhash, errors.Wrap(err, "GetRuntimeVersionLatest")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return txhash, errors.Wrap(err, "CreateStorageKey")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return txhash, errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return txhash, errors.New("GetStorageLatest return value is empty")
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
		return txhash, errors.Wrap(err, "Sign")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return txhash, errors.Wrap(err, "SubmitAndWatchExtrinsic")
	}
	defer sub.Unsubscribe()

	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				txhash, _ = codec.EncodeToHex(status.AsInBlock)
				return txhash, nil
			}
		case err = <-sub.Err():
			return "", err
		case <-timeout:
			return "", errors.New("timeout")
		}
	}
}

// Storage miner redemption deposit function
func Withdraw(api *gsrpc.SubstrateAPI, identifyAccountPhrase, TransactionName string) (string, error) {
	defer func() {
		if err := recover(); err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	var (
		txhash      string
		accountInfo types.AccountInfo
	)
	keyring, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return txhash, errors.Wrap(err, "KeyringPairFromSecret err")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return txhash, errors.Wrap(err, "GetMetadataLatest err")
	}

	c, err := types.NewCall(meta, TransactionName)
	if err != nil {
		return txhash, errors.Wrap(err, "NewCall err")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return txhash, errors.Wrap(err, "NewExtrinsic err")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return txhash, errors.Wrap(err, "GetBlockHash err")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return txhash, errors.Wrap(err, "GetRuntimeVersionLatest err")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return txhash, errors.Wrap(err, "CreateStorageKey err")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return txhash, errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return txhash, errors.New("GetStorageLatest return value is empty")
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
		return txhash, errors.Wrap(err, "Sign err")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return txhash, errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()

	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				txhash, _ = codec.EncodeToHex(status.AsInBlock)
				return txhash, nil
			}
		case err = <-sub.Err():
			return txhash, err
		case <-timeout:
			return txhash, errors.New("timeout")
		}
	}
}

// Bulk submission proof
func SubmitProofs(data []ProveInfo) (string, error) {
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()
	var (
		txhash      string
		accountInfo types.AccountInfo
	)

	api, err := GetRpcClient_Safe(configs.C.RpcAddr)
	defer Free()
	if err != nil {
		return txhash, errors.Wrap(err, "[GetRpcClient_Safe]")
	}

	meta, err := GetMetadata(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetMetadataLatest]")
	}

	c, err := types.NewCall(meta, TX_AUDIT_REPORTPROOF, data)
	if err != nil {
		return txhash, errors.Wrap(err, "[NewCall]")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return txhash, errors.Wrap(err, "[NewExtrinsic]")
	}

	genesisHash, err := GetGenesisHash(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetGenesisHash]")
	}

	rv, err := GetRuntimeVersion(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetRuntimeVersion]")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", pattern.GetMinerAcc())
	if err != nil {
		return txhash, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetStorageLatest]")
	}

	if !ok {
		return txhash, errors.New(ERR_Empty)
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

	kring, err := GetKeyring(configs.C.SignatureAcc)
	if err != nil {
		return txhash, errors.Wrap(err, "GetKeyring")
	}

	// Sign the transaction
	err = ext.Sign(kring, o)
	if err != nil {
		return txhash, errors.Wrap(err, "[Sign]")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return txhash, errors.Wrap(err, "[SubmitAndWatchExtrinsic]")
	}
	defer sub.Unsubscribe()
	timeout := time.After(configs.TimeToWaitEvents)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := EventRecords{}
				txhash, _ = codec.EncodeToHex(status.AsInBlock)
				keye, err := GetKeyEvents()
				if err != nil {
					return txhash, errors.Wrap(err, "GetKeyEvents")
				}
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					return txhash, errors.Wrap(err, "GetStorageRaw")
				}

				types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)

				if len(events.SegmentBook_ChallengeProof) > 0 && len(data) > 0 {
					return txhash, nil
				}
				return txhash, errors.New(ERR_Failed)
			}
		case err = <-sub.Err():
			return txhash, errors.Wrap(err, "<-sub")
		case <-timeout:
			return txhash, errors.New(ERR_Timeout)
		}
	}
}

// Clear invalid files
func ClearInvalidFiles(fid FileHash) (string, error) {
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()
	var (
		txhash      string
		accountInfo types.AccountInfo
	)
	api, err := GetRpcClient_Safe(configs.C.RpcAddr)
	defer Free()
	if err != nil {
		return txhash, errors.Wrap(err, "[GetRpcClient_Safe]")
	}

	meta, err := GetMetadata(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetMetadataLatest]")
	}

	c, err := types.NewCall(meta, TX_FILEBANK_DELFILE, fid)
	if err != nil {
		return txhash, errors.Wrap(err, "[NewCall]")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return txhash, errors.Wrap(err, "[NewExtrinsic]")
	}

	genesisHash, err := GetGenesisHash(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetGenesisHash]")
	}

	rv, err := GetRuntimeVersion(api)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetRuntimeVersion]")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", pattern.GetMinerAcc())
	if err != nil {
		return txhash, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return txhash, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return txhash, errors.New(ERR_Empty)
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

	kring, err := GetKeyring(configs.C.SignatureAcc)
	if err != nil {
		return txhash, errors.Wrap(err, "GetKeyring")
	}

	// Sign the transaction
	err = ext.Sign(kring, o)
	if err != nil {
		return txhash, errors.Wrap(err, "[Sign]")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return txhash, errors.Wrap(err, "[SubmitAndWatchExtrinsic]")
	}
	defer sub.Unsubscribe()
	timeout := time.After(configs.TimeToWaitEvents)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				return codec.EncodeToHex(status.AsInBlock)
			}
		case err = <-sub.Err():
			return txhash, errors.Wrap(err, "<-sub")
		case <-timeout:
			return txhash, errors.New(ERR_Timeout)
		}
	}
}

// Clear all filler files
func ClearFiller(api *gsrpc.SubstrateAPI, signaturePrk string) (int, error) {
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()

	var accountInfo types.AccountInfo

	keyring, err := signature.KeyringPairFromSecret(signaturePrk, 0)
	if err != nil {
		return configs.Code_400, errors.Wrap(err, "[KeyringPairFromSecret]")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	c, err := types.NewCall(meta, TX_FILEBANK_DELALLFILLER)
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
	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				return configs.Code_200, nil
			}
		case err = <-sub.Err():
			return configs.Code_500, err
		case <-timeout:
			return configs.Code_500, errors.New("Timeout")
		}
	}
}

func UpdateAddress(transactionPrK, ip, port string) (string, error) {
	var (
		err         error
		accountInfo types.AccountInfo
	)

	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()

	api, err := NewRpcClient(configs.C.RpcAddr)
	if err != nil {
		return "", errors.Wrap(err, "NewRpcClient err")
	}

	keyring, err := signature.KeyringPairFromSecret(transactionPrK, 0)
	if err != nil {
		return "", errors.Wrap(err, "KeyringPairFromSecret err")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return "", errors.Wrap(err, "GetMetadataLatest err")
	}

	var ipType IpAddress

	if tools.IsIPv4(ip) {
		ipType.IPv4.Index = 0
		ips := strings.Split(ip, ".")
		for i := 0; i < 4; i++ {
			temp, _ := strconv.Atoi(ips[i])
			ipType.IPv4.Value[i] = types.U8(temp)
		}
		temp, _ := strconv.Atoi(port)
		ipType.IPv4.Port = types.U16(temp)
	} else {
		return "", errors.New("unsupported ip format")
	}

	c, err := types.NewCall(meta, TX_SMINER_UPDATEADDR, ipType.IPv4)
	if err != nil {
		return "", errors.Wrap(err, "NewCall err")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return "", errors.Wrap(err, "NewExtrinsic err")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return "", errors.Wrap(err, "GetBlockHash err")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return "", errors.Wrap(err, "GetRuntimeVersionLatest err")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return "", errors.Wrap(err, "CreateStorageKey System  Account err")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return "", errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return "", errors.New("GetStorageLatest return value is empty")
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
		return "", errors.Wrap(err, "Sign err")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return "", errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()
	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := EventRecords{}
				txhash, _ := codec.EncodeToHex(status.AsInBlock)
				keye, err := types.CreateStorageKey(meta, "System", "Events", nil)
				if err != nil {
					return txhash, errors.Wrap(err, "GetKeyEvents")
				}
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					return txhash, errors.Wrap(err, "GetStorageRaw")
				}

				types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)

				if len(events.Sminer_UpdataIp) > 0 {
					return txhash, nil
				}
				return txhash, errors.New(ERR_Failed)
			}
		case err = <-sub.Err():
			return "", errors.Wrap(err, "<-sub")
		case <-timeout:
			return "", errors.Errorf("timeout")
		}
	}
}

func UpdateIncome(transactionPrK string, acc types.AccountID) (string, error) {
	var (
		err         error
		accountInfo types.AccountInfo
	)
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()
	api, err := NewRpcClient(configs.C.RpcAddr)
	if err != nil {
		return "", errors.Wrap(err, "NewRpcClient err")
	}
	keyring, err := signature.KeyringPairFromSecret(transactionPrK, 0)
	if err != nil {
		return "", errors.Wrap(err, "KeyringPairFromSecret err")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return "", errors.Wrap(err, "GetMetadataLatest err")
	}

	c, err := types.NewCall(meta, TX_SMINER_UPDATEACC, acc)
	if err != nil {
		return "", errors.Wrap(err, "NewCall err")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return "", errors.Wrap(err, "NewExtrinsic err")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return "", errors.Wrap(err, "GetBlockHash err")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return "", errors.Wrap(err, "GetRuntimeVersionLatest err")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return "", errors.Wrap(err, "CreateStorageKey System  Account err")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return "", errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return "", errors.New("GetStorageLatest return value is empty")
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
		return "", errors.Wrap(err, "Sign err")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return "", errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()
	timeout := time.After(time.Second * configs.TimeToWaitEvents_S)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := EventRecords{}
				txhash, _ := codec.EncodeToHex(status.AsInBlock)
				keye, err := types.CreateStorageKey(meta, "System", "Events", nil)
				if err != nil {
					return txhash, errors.Wrap(err, "GetKeyEvents")
				}
				h, err := api.RPC.State.GetStorageRaw(keye, status.AsInBlock)
				if err != nil {
					return txhash, errors.Wrap(err, "GetStorageRaw")
				}

				types.EventRecordsRaw(*h).DecodeEventRecords(meta, &events)

				if len(events.Sminer_UpdataBeneficiary) > 0 {
					return txhash, nil
				}
				return txhash, errors.New(ERR_Failed)
			}
		case err = <-sub.Err():
			return "", errors.Wrap(err, "<-sub")
		case <-timeout:
			return "", errors.Errorf("timeout")
		}
	}
}
