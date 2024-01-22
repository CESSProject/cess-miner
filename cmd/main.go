/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/CESSProject/cess-bucket/cmd/console"
)

// program entry
func main() {
	exitCh := make(chan os.Signal)
	signal.Notify(exitCh, os.Interrupt, os.Kill, syscall.SIGTERM)
	go exitHandle(exitCh)
	console.Execute()
}

func exitHandle(exitCh chan os.Signal) {
	for {
		select {
		case sig := <-exitCh:
			panic(sig.String())
		}
	}
}
