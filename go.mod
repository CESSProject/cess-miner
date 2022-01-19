module storage-mining

go 1.16

require (
	github.com/BurntSushi/toml v0.4.1 // indirect
	github.com/centrifuge/go-substrate-rpc-client/v4 v4.0.0
	github.com/filecoin-project/go-address v0.0.6
	github.com/filecoin-project/go-fil-commcid v0.1.0
	github.com/filecoin-project/go-state-types v0.1.1
	github.com/filecoin-project/specs-actors v0.9.13
	github.com/filecoin-project/specs-actors/v5 v5.0.4
	github.com/gin-contrib/cors v1.3.1
	github.com/gin-gonic/gin v1.7.4
	github.com/ipfs/go-block-format v0.0.3 // indirect
	github.com/ipfs/go-cid v0.1.0
	github.com/ipfs/go-ipfs-chunker v0.0.5
	github.com/klauspost/reedsolomon v1.9.14
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1
	github.com/natefinch/lumberjack v2.0.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/shirou/gopsutil v3.21.10+incompatible
	github.com/spf13/viper v1.9.0
	go.uber.org/zap v1.19.1
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
)

//replace storage-mining/internal/cess-ffi => ./internal/cess-ffi
