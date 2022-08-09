package chain

import (
	"reflect"
	"sync"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

var (
	first          bool = true
	l              *sync.Mutex
	api            *gsrpc.SubstrateAPI
	metadata       *types.Metadata
	keyEvents      types.StorageKey
	runtimeVersion *types.RuntimeVersion
	genesisHash    types.Hash
	keyring        signature.KeyringPair
)

func init() {
	l = new(sync.Mutex)
}

func GetRpcClient_Safe(rpcaddr string) (*gsrpc.SubstrateAPI, error) {
	var err error
	l.Lock()
	if api == nil {
		api, err = gsrpc.NewSubstrateAPI(rpcaddr)
		return api, err
	}
	id, err := healthchek(api)
	if id == 0 || err != nil {
		api, err = gsrpc.NewSubstrateAPI(rpcaddr)
		if err != nil {
			return nil, err
		}
	}
	return api, nil
}

func Free() {
	l.Unlock()
}

func NewRpcClient(rpcaddr string) (*gsrpc.SubstrateAPI, error) {
	return gsrpc.NewSubstrateAPI(rpcaddr)
}

func healthchek(a *gsrpc.SubstrateAPI) (uint64, error) {
	defer recover()
	h, err := a.RPC.System.Health()
	return uint64(h.Peers), err
}

func GetMetadata(api *gsrpc.SubstrateAPI) (*types.Metadata, error) {
	var err error
	if metadata == nil {
		metadata, err = api.RPC.State.GetMetadataLatest()
		return metadata, err
	}
	return metadata, nil
}

func GetGenesisHash(api *gsrpc.SubstrateAPI) (types.Hash, error) {
	var err error
	if first {
		genesisHash, err = api.RPC.Chain.GetBlockHash(0)
		if err == nil {
			first = false
		}
		return genesisHash, err
	}
	return genesisHash, nil
}

func GetRuntimeVersion(api *gsrpc.SubstrateAPI) (*types.RuntimeVersion, error) {
	var err error
	if runtimeVersion == nil {
		runtimeVersion, err = api.RPC.State.GetRuntimeVersionLatest()
		return runtimeVersion, err
	}
	return runtimeVersion, nil
}

func GetKeyEvents() (types.StorageKey, error) {
	var err error
	if len(keyEvents) == 0 {
		keyEvents, err = types.CreateStorageKey(metadata, "System", "Events", nil)
		return keyEvents, err
	}
	return keyEvents, nil
}

func GetKeyring(prk string) (signature.KeyringPair, error) {
	var err error
	if reflect.DeepEqual(keyring, signature.KeyringPair{}) {
		keyring, err = signature.KeyringPairFromSecret(prk, 0)
		return keyring, err
	}
	return keyring, nil
}

func GetSelfPublicKey(privatekey string) ([]byte, error) {
	kring, err := signature.KeyringPairFromSecret(privatekey, 0)
	if err != nil {
		return nil, err
	}
	return kring.PublicKey, nil
}
