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

// Request
type Request struct {
	conn   IConnection
	msg    IMessage
	router IRouter
	index  int8
}

// GetConnection Get the request connection information
func (r *Request) GetConnection() IConnection {
	return r.conn
}

// GetData Get the requested data
func (r *Request) GetData() []byte {
	return r.msg.GetData()
}

// GetMsgID Get the requested message ID
func (r *Request) GetMsgID() uint32 {
	return r.msg.GetMsgID()
}

// BindRouter binding request message routing
func (r *Request) BindRouter(router IRouter) {
	r.router = router
}
