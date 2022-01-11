package chain

import (
	"math/big"

	"storage-mining/configs"
	"storage-mining/internal/logger"
	"storage-mining/tools"
	"strconv"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

// miner register
func RegisterToChain(identifyAccountPhrase, incomeAccountPublicKey, ipAddr, TransactionName string, pledgeTokens uint64, port, fileport uint32) error {
	var (
		err         error
		accountInfo types.AccountInfo
	)
	api := getSubstrateAPI()
	defer func() {
		releaseSubstrateAPI()
		err := recover()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	ipint, err := tools.InetAtoN(ipAddr)
	if err != nil {
		return errors.Wrap(err, "InetAtoN err")
	}

	pTokens := strconv.FormatUint(pledgeTokens, 10)
	pTokens += configs.TokenAccuracy
	realTokens, ok := new(big.Int).SetString(pTokens, 10)
	if !ok {
		return errors.New("SetString err")
	}
	amount := types.NewUCompact(realTokens)

	keyring, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return errors.Wrap(err, "KeyringPairFromSecret err")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return errors.Wrap(err, "GetMetadataLatest err")
	}

	incomeAccount, err := types.NewMultiAddressFromHexAccountID(incomeAccountPublicKey)
	if err != nil {
		return errors.Wrap(err, "NewMultiAddressFromHexAccountID err")
	}

	c, err := types.NewCall(meta, TransactionName, incomeAccount, types.NewU32(uint32(ipint)), types.NewU32(port), types.NewU32(fileport), amount)
	if err != nil {
		return errors.Wrap(err, "NewCall err")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return errors.Wrap(err, "NewExtrinsic err")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return errors.Wrap(err, "GetBlockHash err")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return errors.Wrap(err, "GetRuntimeVersionLatest err")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return errors.Wrap(err, "CreateStorageKey err")
	}

	ok, err = api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return errors.New("GetStorageLatest return value is empty")
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
		return errors.Wrap(err, "Sign err")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()

	timeout := time.After(10 * time.Second)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				logger.InfoLogger.Sugar().Infof("[%v] tx blockhash: %#x", TransactionName, status.AsInBlock)
				return nil
			}
		case <-timeout:
			return errors.Errorf("[%v] tx timeout", TransactionName)
		default:
			time.Sleep(time.Second)
		}
	}
}

//
func IntentSubmitToChain(identifyAccountPhrase, TransactionName string, segsizetype, segtype uint8, peerd uint64, unsealedcid [][]byte, hash, shardhash []byte) error {
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
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	keyring, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return errors.Wrap(err, "KeyringPairFromSecret err")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return errors.Wrap(err, "GetMetadataLatest err")
	}
	var uncid []types.Bytes = make([]types.Bytes, len(unsealedcid))
	for i := 0; i < len(unsealedcid); i++ {
		uncid[i] = make(types.Bytes, 0)
		uncid[i] = append(uncid[i], unsealedcid[i]...)
	}
	c, err := types.NewCall(meta, TransactionName, types.NewU8(segsizetype), types.NewU8(segtype), types.NewU64(peerd), uncid, types.NewBytes(hash), types.NewBytes(shardhash))
	if err != nil {
		return errors.Wrap(err, "NewCall err")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return errors.Wrap(err, "NewExtrinsic err")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return errors.Wrap(err, "GetBlockHash err")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return errors.Wrap(err, "GetRuntimeVersionLatest err")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return errors.Wrap(err, "CreateStorageKey err")
	}

	ok, err = api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return errors.New("GetStorageLatest return value is empty")
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
		return errors.Wrap(err, "Sign err")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()

	timeout := time.After(10 * time.Second)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				logger.InfoLogger.Sugar().Infof("[%v] tx blockhash: %#x", TransactionName, status.AsInBlock)
				return nil
			}
		case <-timeout:
			return errors.Errorf("[%v] tx timeout", TransactionName)
		default:
			time.Sleep(time.Second)
		}
	}
}

//
func IntentSubmitPostToChain(identifyAccountPhrase, TransactionName string, segid uint64, segsizetype, segtype uint8) error {
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
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	keyring, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return errors.Wrap(err, "KeyringPairFromSecret err")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return errors.Wrap(err, "GetMetadataLatest err")
	}

	c, err := types.NewCall(meta, TransactionName, types.NewU64(segid), types.NewU8(segsizetype), types.NewU8(segtype))
	if err != nil {
		return errors.Wrap(err, "NewCall err")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return errors.Wrap(err, "NewExtrinsic err")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return errors.Wrap(err, "GetBlockHash err")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return errors.Wrap(err, "GetRuntimeVersionLatest err")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return errors.Wrap(err, "CreateStorageKey err")
	}

	ok, err = api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return errors.New("GetStorageLatest return value is empty")
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
		return errors.Wrap(err, "Sign err")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()

	timeout := time.After(10 * time.Second)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				logger.InfoLogger.Sugar().Infof("[%v] tx blockhash: %#x", TransactionName, status.AsInBlock)
				return nil
			}
		case <-timeout:
			return errors.Errorf("[%v] tx timeout", TransactionName)
		default:
			time.Sleep(time.Second)
		}
	}
}

// Submit To Vpa or Vpb
func SegmentSubmitToVpaOrVpb(identifyAccountPhrase, TransactionName string, peerid, segmentid uint64, proofs, cid []byte) error {
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
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	keyring, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return errors.Wrapf(err, "KeyringPairFromSecret err [%v]", TransactionName)
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return errors.Wrapf(err, "GetMetadataLatest err [%v]", TransactionName)
	}

	c, err := types.NewCall(meta, TransactionName, types.NewU64(peerid), types.NewU64(segmentid), types.NewBytes(proofs), types.NewBytes(cid))
	if err != nil {
		return errors.Wrapf(err, "NewCall err [%v]", TransactionName)
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return errors.Wrap(err, "NewExtrinsic err")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return errors.Wrap(err, "GetBlockHash err")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return errors.Wrap(err, "GetRuntimeVersionLatest err")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return errors.Wrap(err, "CreateStorageKey err")
	}

	ok, err = api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return errors.New("GetStorageLatest return value is empty")
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
		return errors.Wrap(err, "Sign err")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()

	timeout := time.After(10 * time.Second)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				logger.InfoLogger.Sugar().Infof("[%v] tx blockhash: %#x", TransactionName, status.AsInBlock)
				return nil
			}
		case <-timeout:
			return errors.Errorf("[%v] tx timeout", TransactionName)
		}
	}
}

// Submit To Vpc
func SegmentSubmitToVpc(identifyAccountPhrase, TransactionName string, peerid, segmentid uint64, proofs [][]byte, sealcid []types.Bytes) error {
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
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	keyring, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return errors.Wrap(err, "KeyringPairFromSecret err")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return errors.Wrap(err, "GetMetadataLatest err")
	}

	var fileVpc []types.Bytes = make([]types.Bytes, len(proofs))
	for i := 0; i < len(proofs); i++ {
		fileVpc[i] = make(types.Bytes, 0)
		fileVpc[i] = append(fileVpc[i], proofs[i]...)
	}
	// var sealedcid []types.Bytes = make([]types.Bytes, len(sealcid))
	// for i := 0; i < len(sealcid); i++ {
	// 	sealedcid[i] = make(types.Bytes, 0)
	// 	sealedcid[i] = append(sealedcid[i], sealcid[i]...)
	// }
	c, err := types.NewCall(meta, TransactionName, types.NewU64(peerid), types.NewU64(segmentid), fileVpc, sealcid)
	if err != nil {
		return errors.Wrap(err, "NewCall err")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return errors.Wrap(err, "NewExtrinsic err")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return errors.Wrap(err, "GetBlockHash err")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return errors.Wrap(err, "GetRuntimeVersionLatest err")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return errors.Wrap(err, "CreateStorageKey err")
	}

	ok, err = api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return errors.New("GetStorageLatest return value is empty")
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
		return errors.Wrap(err, "Sign err")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()

	timeout := time.After(10 * time.Second)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				logger.InfoLogger.Sugar().Infof("[%v] tx blockhash: %#x", TransactionName, status.AsInBlock)
				return nil
			}
		case <-timeout:
			return errors.Errorf("[%v] tx timeout", TransactionName)
		default:
			time.Sleep(time.Second)
		}
	}
}

// Submit To Vpd
func SegmentSubmitToVpd(identifyAccountPhrase, TransactionName string, peerid, segmentid uint64, proofs [][]byte, sealcid []types.Bytes) error {
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
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	keyring, err := signature.KeyringPairFromSecret(identifyAccountPhrase, 0)
	if err != nil {
		return errors.Wrap(err, "KeyringPairFromSecret err")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return errors.Wrap(err, "GetMetadataLatest err")
	}

	var fileVpd []types.Bytes = make([]types.Bytes, len(proofs))
	for i := 0; i < len(proofs); i++ {
		fileVpd[i] = make(types.Bytes, 0)
		fileVpd[i] = append(fileVpd[i], proofs[i]...)
	}
	c, err := types.NewCall(meta, TransactionName, types.NewU64(peerid), types.NewU64(segmentid), fileVpd, sealcid)
	if err != nil {
		return errors.Wrap(err, "NewCall err")
	}

	ext := types.NewExtrinsic(c)
	if err != nil {
		return errors.Wrap(err, "NewExtrinsic err")
	}

	genesisHash, err := api.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return errors.Wrap(err, "GetBlockHash err")
	}

	rv, err := api.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return errors.Wrap(err, "GetRuntimeVersionLatest err")
	}

	key, err := types.CreateStorageKey(meta, "System", "Account", keyring.PublicKey)
	if err != nil {
		return errors.Wrap(err, "CreateStorageKey err")
	}

	ok, err = api.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil {
		return errors.Wrap(err, "GetStorageLatest err")
	}
	if !ok {
		return errors.New("GetStorageLatest return value is empty")
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
		return errors.Wrap(err, "Sign err")
	}

	// Do the transfer and track the actual status
	sub, err := api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return errors.Wrap(err, "SubmitAndWatchExtrinsic err")
	}
	defer sub.Unsubscribe()

	timeout := time.After(10 * time.Second)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				logger.InfoLogger.Sugar().Infof("[%v] tx blockhash: %#x", TransactionName, status.AsInBlock)
				return nil
			}
		case <-timeout:
			return errors.Errorf("[%v] tx timeout", TransactionName)
		default:
			time.Sleep(time.Second)
		}
	}
}
