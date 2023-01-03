/*
   Copyright 2022 CESS (Cumulus Encrypted Storage System) authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

        http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package node

import (
	"net/http"

	"github.com/CESSProject/cess-bucket/pkg/chain"
)

type Report struct {
	Cert      string
	Ias_sig   string
	Quote     string
	Quote_sig string
}

const (
	M_Pending  = "pending"
	M_Positive = "positive"
	M_Frozen   = "frozen"
	M_Exit     = "exit"
)

const (
	Cach_Blockheight = "blockheight:"
)

var (
	Ch_Report       chan Report
	Ch_Tag          chan chain.Result
	globalTransport *http.Transport
)

func init() {
	globalTransport = &http.Transport{
		DisableKeepAlives: true,
	}
	Ch_Report = make(chan Report, 0)
	Ch_Tag = make(chan chain.Result, 0)
}
