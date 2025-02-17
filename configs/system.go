/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package configs

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"runtime"

	out "github.com/CESSProject/cess-miner/pkg/fout"
	"google.golang.org/grpc/credentials"
)

const (
	// Name is the name of the program
	Name = "miner"
	// version
	Version = "v0.8.2 2502171422-1739773355316"
	// Description is the description of the program
	Description = "Storage miner implementation in CESS networks"
	// NameSpace is the cached namespace
	NameSpaces = Name
)

// Chain version
var ChainVersionStr = [3]string{"0", "7", "9"}
var ChainVersionInt = [3]int{0, 7, 9}

var cp *x509.CertPool

// system init
func SysInit(cpus uint32) int {
	if !RunOnLinuxSystem() {
		out.Err("Please run on a linux system")
		os.Exit(1)
	}
	if err := initCert(); err != nil {
		out.Err("Invalid certificate, please check configs/.pem")
		os.Exit(1)
	}
	return SetCpuNumber(cpus)
}

func SetCpuNumber(cpus uint32) int {
	actualUseCpus := runtime.NumCPU()
	if cpus == 0 || int(cpus) >= actualUseCpus {
		runtime.GOMAXPROCS(actualUseCpus)
		return actualUseCpus
	}
	actualUseCpus = int(cpus)
	runtime.GOMAXPROCS(actualUseCpus)
	return actualUseCpus
}

func RunOnLinuxSystem() bool {
	return runtime.GOOS == "linux"
}

func initCert() error {
	cp = x509.NewCertPool()
	if !cp.AppendCertsFromPEM([]byte(pem)) {
		return fmt.Errorf("credentials: failed to append certificates")
	}
	return nil
}

func GetCert() credentials.TransportCredentials {
	return credentials.NewTLS(&tls.Config{ServerName: "", RootCAs: cp})
}
