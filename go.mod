module cess-bucket

go 1.16

require (
	github.com/BurntSushi/toml v0.4.1 // indirect
	github.com/CESSProject/cess-ffi v0.0.0-20220217052609-6c35c99d795c
	github.com/centrifuge/go-substrate-rpc-client/v4 v4.0.0
	github.com/deckarep/golang-set v1.7.1
	github.com/filecoin-project/go-address v0.0.6 // indirect
	github.com/filecoin-project/go-fil-commcid v0.1.0 // indirect
	github.com/filecoin-project/go-state-types v0.1.1
	github.com/filecoin-project/specs-actors v0.9.13
	github.com/filecoin-project/specs-actors/v5 v5.0.4 // indirect
	github.com/golang/protobuf v1.5.2
	github.com/gorilla/websocket v1.4.2
	github.com/ipfs/go-cid v0.1.0
	github.com/ipfs/go-ipfs-chunker v0.0.5
	github.com/klauspost/cpuid/v2 v2.0.6 // indirect
	github.com/natefinch/lumberjack v2.0.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/shirou/gopsutil v3.21.10+incompatible
	github.com/spf13/cobra v1.3.0
	github.com/spf13/viper v1.10.0
	go.uber.org/zap v1.19.1
	google.golang.org/protobuf v1.27.1
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
)

replace github.com/CESSProject/cess-ffi => ./internal/ffi
