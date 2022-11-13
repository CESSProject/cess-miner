package main

import (
	// "net/http"
	// _ "net/http/pprof"

	"github.com/CESSProject/cess-bucket/cmd"
)

// program entry
func main() {
	// go func() {
	// 	http.ListenAndServe(":8080", nil)
	// }()
	cmd.Execute()
}
