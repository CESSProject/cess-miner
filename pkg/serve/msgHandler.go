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
	"strconv"
)

// MsgHandle
type MsgHandle struct {
	Apis map[uint32]IRouter
}

// NewMsgHandle
func NewMsgHandle() *MsgHandle {
	return &MsgHandle{
		Apis: make(map[uint32]IRouter),
	}
}

// DoMsgHandler
func (mh *MsgHandle) DoMsgHandler(ctx context.CancelFunc, request IRequest) {
	handler, ok := mh.Apis[request.GetMsgID()]
	if !ok {
		//fmt.Println("api msgID = ", request.GetMsgID(), " is not FOUND!")
		return
	}

	request.BindRouter(handler)
	handler.Handle(ctx, request)
}

// AddRouter
func (mh *MsgHandle) AddRouter(msgID uint32, router IRouter) {
	if _, ok := mh.Apis[msgID]; ok {
		panic("repeated api , msgID = " + strconv.Itoa(int(msgID)))
	}
	mh.Apis[msgID] = router
}
