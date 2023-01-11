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
	"errors"
	"fmt"
	"sync"
)

// Connection Manager
type ConnManager struct {
	connections map[uint32]IConnection
	connLock    sync.RWMutex
}

// NewConnManager
func NewConnManager() *ConnManager {
	return &ConnManager{
		connections: make(map[uint32]IConnection),
	}
}

// Add a Connection
func (connMgr *ConnManager) Add(conn IConnection) {
	connMgr.connLock.Lock()
	connMgr.connections[conn.GetConnID()] = conn
	connMgr.connLock.Unlock()
	//fmt.Println("connection add to ConnManager successfully: conn num = ", connMgr.Len())
}

// Remove a Connection
func (connMgr *ConnManager) Remove(conn IConnection) {
	connMgr.connLock.Lock()
	delete(connMgr.connections, conn.GetConnID())
	connMgr.connLock.Unlock()
	//fmt.Println("connection Remove ConnID=", conn.GetConnID(), " successfully: conn num = ", connMgr.Len())
}

// Get Get the ID of the Connection
func (connMgr *ConnManager) Get(connID uint32) (IConnection, error) {
	connMgr.connLock.RLock()
	defer connMgr.connLock.RUnlock()

	if conn, ok := connMgr.connections[connID]; ok {
		return conn, nil
	}

	return nil, errors.New("connection not found")

}

// Len gets the number of connections
func (connMgr *ConnManager) Len() int {
	connMgr.connLock.RLock()
	length := len(connMgr.connections)
	connMgr.connLock.RUnlock()
	return length
}

// ClearConn clears and stops all connections
func (connMgr *ConnManager) ClearConn() {
	connMgr.connLock.Lock()

	for _, conn := range connMgr.connections {
		conn.Stop()
	}

	connMgr.connLock.Unlock()
	fmt.Println("Clear All Connections successfully: conn num = ", connMgr.Len())
}

// ClearOneConn stops and clears a connection
func (connMgr *ConnManager) ClearOneConn(connID uint32) {
	connMgr.connLock.Lock()
	defer connMgr.connLock.Unlock()

	connections := connMgr.connections
	if conn, ok := connections[connID]; ok {
		conn.Stop()
		fmt.Println("Clear Connections ID:  ", connID, "succeed")
		return
	}

	fmt.Println("Clear Connections ID:  ", connID, "err")
}
