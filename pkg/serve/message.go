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

// Message
type Message struct {
	DataLen uint32
	ID      uint32
	Data    []byte
}

// NewMsgPackage
func NewMsgPackage(ID uint32, data []byte) *Message {
	return &Message{
		DataLen: uint32(len(data)),
		ID:      ID,
		Data:    data,
	}
}

func (msg *Message) Init(ID uint32, data []byte) {
	msg.ID = ID
	msg.Data = data
	msg.DataLen = uint32(len(data))
}

// GetDataLen Get message length
func (msg *Message) GetDataLen() uint32 {
	return msg.DataLen
}

// GetMsgID Get Message ID
func (msg *Message) GetMsgID() uint32 {
	return msg.ID
}

// GetData Get Message Content
func (msg *Message) GetData() []byte {
	return msg.Data
}

// SetDataLen sets the message length
func (msg *Message) SetDataLen(len uint32) {
	msg.DataLen = len
}

// SetMsgID Set Message ID
func (msg *Message) SetMsgID(msgID uint32) {
	msg.ID = msgID
}

// SetData Set Message Content
func (msg *Message) SetData(data []byte) {
	msg.Data = data
}
