package configs

// RespMsg
type RespMsg struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

var (
	Service_ADDR string
	Service_PORT string
)
