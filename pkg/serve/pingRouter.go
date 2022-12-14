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

package serve

import (
	"context"
	"fmt"
)

// PingRouter
type PingRouter struct {
	BaseRouter
}

// Ping Handle
func (this *PingRouter) Handle(ctx context.CancelFunc, request IRequest) {
	if request.GetMsgID() != Msg_Ping {
		fmt.Println("MsgId error")
		ctx()
		return
	}
	request.GetConnection().SendMsg(Msg_OK, nil)
}
