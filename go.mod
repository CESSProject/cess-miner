module storage-mining

go 1.16

require (
	github.com/BurntSushi/toml v0.4.1 // indirect
	github.com/centrifuge/go-substrate-rpc-client/v4 v4.0.0
	github.com/filecoin-project/go-address v0.0.6 // indirect
	github.com/filecoin-project/go-state-types v0.1.1
	github.com/filecoin-project/specs-actors v0.9.13
	github.com/gin-contrib/cors v1.3.1
	github.com/gin-gonic/gin v1.7.4
	github.com/ipfs/go-block-format v0.0.3 // indirect
	github.com/ipfs/go-cid v0.1.0
	github.com/ipfs/go-ipfs-chunker v0.0.5
	github.com/klauspost/reedsolomon v1.9.14
	github.com/natefinch/lumberjack v2.0.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/shirou/gopsutil v3.21.10+incompatible
	github.com/spf13/viper v1.9.0
	go.uber.org/zap v1.19.1
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	storage-mining/internal/cess-ffi v0.0.0-00010101000000-000000000000
)

replace storage-mining/internal/cess-ffi => ./internal/cess-ffi
