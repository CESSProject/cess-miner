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
	"bytes"
	"encoding/binary"
	"errors"

	"github.com/CESSProject/cess-bucket/configs"
)

var defaultHeaderLen uint32 = 8

// DataPack
type DataPack struct{}

// NewDataPack
func NewDataPack() IDataPack {
	return &DataPack{}
}

// GetHeadLen Get the header length method
func (dp *DataPack) GetHeadLen() uint32 {
	//ID (uint32) +  DataLen (uint32)
	return defaultHeaderLen
}

// Pack packaging method (compressed data)
func (dp *DataPack) Pack(msg IMessage) ([]byte, error) {
	dataBuff := bytes.NewBuffer([]byte{})

	// write dataLen
	if err := binary.Write(dataBuff, binary.LittleEndian, msg.GetDataLen()); err != nil {
		return nil, err
	}

	// write msgID
	if err := binary.Write(dataBuff, binary.LittleEndian, msg.GetMsgID()); err != nil {
		return nil, err
	}

	// write data
	if err := binary.Write(dataBuff, binary.LittleEndian, msg.GetData()); err != nil {
		return nil, err
	}

	return dataBuff.Bytes(), nil
}

// Unpack method (decompress data)
func (dp *DataPack) Unpack(binaryData []byte) (IMessage, error) {
	dataBuff := bytes.NewReader(binaryData)

	msg := &Message{}

	// read dataLen
	if err := binary.Read(dataBuff, binary.LittleEndian, &msg.DataLen); err != nil {
		return nil, err
	}

	// read msgID
	if err := binary.Read(dataBuff, binary.LittleEndian, &msg.ID); err != nil {
		return nil, err
	}

	// Judge whether the length of dataLen exceeds the maximum packet length
	if configs.TCP_MaxPacketSize > 0 && msg.DataLen > configs.TCP_MaxPacketSize {
		return nil, errors.New("too large msg data received")
	}

	return msg, nil
}
