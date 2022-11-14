package node

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
	. "github.com/CESSProject/cess-bucket/internal/logger"
	"github.com/CESSProject/cess-bucket/tools"
)

func (n *Node) NewServer(conn NetConn, fileDir string) Server {
	n.Conn = &ConMgr{
		conn:       conn,
		fileDir:    fileDir,
		waitNotify: make(chan bool, 1),
	}
	return n
}

func (n *Node) Start() {
	n.Conn.conn.HandlerLoop()
	n.handler()
	time.Sleep(time.Second)
	Out.Sugar().Infof("Close a conn: %v", n.Conn.conn.GetRemoteAddr())
	n.Conn = nil
	n = nil
}

func (n *Node) handler() error {
	var err error
	var fs *os.File

	defer func() {
		if err != nil {
			time.Sleep(time.Second)
		}
		err := recover()
		if err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
		n.Conn.conn.Close()
		close(n.Conn.waitNotify)
		if fs != nil {
			fs.Close()
		}
	}()

	for !n.Conn.conn.IsClose() {
		m, ok := n.Conn.conn.GetMsg()
		if !ok {
			return fmt.Errorf("Getmsg failed")
		}

		if m == nil {
			continue
		}

		switch m.MsgType {
		case MsgHead:
			// Verify signature
			ok, err := VerifySign(m.Pubkey, m.SignMsg, m.Sign)
			if err != nil || !ok {
				n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Err))
				return errors.New("Signature error")
			}

			if m.FileType == FileType_file {
				fs, err = os.OpenFile(filepath.Join(configs.FilesDir, m.FileName), os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
				if err != nil {
					n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Err))
					return err
				}
			} else if m.FileType == FileType_filler {
				fs, err = os.OpenFile(filepath.Join(configs.SpaceDir, m.FileName), os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
				if err != nil {
					n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Err))
					return err
				}
			} else {
				n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Err))
				return err
			}
			n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Ok))
		case MsgRecvHead:
			// Verify signature
			ok, err := VerifySign(m.Pubkey, m.SignMsg, m.Sign)
			if err != nil || !ok {
				n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Err))
				return errors.New("Signature error")
			}

			fs, err = os.Open(filepath.Join(n.Conn.fileDir, m.FileName))
			if err != nil {
				n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Err))
				return err
			}

			n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Ok))

		case MsgRecvFile:
			if fs == nil {
				n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Err))
				return errors.New("File not open")
			}
			fileInfo, _ := fs.Stat()
			for !n.Conn.conn.IsClose() {
				readBuf := bytesPool.Get().([]byte)
				num, err := fs.Read(readBuf)
				if err != nil && err != io.EOF {
					return err
				}
				if num == 0 {
					break
				}
				n.Conn.conn.SendMsg(NewFileMsg(m.FileName, readBuf[:num]))
			}
			time.Sleep(time.Millisecond)
			n.Conn.conn.SendMsg(NewEndMsg(m.FileName, uint64(fileInfo.Size())))
			time.Sleep(time.Millisecond)
			n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Ok))
		case MsgFile:
			if fs == nil {
				n.Conn.conn.SendMsg(NewCloseMsg("", Status_Err))
				return errors.New("file is not open !")
			}
			_, err = fs.Write(m.Bytes)
			if err != nil {
				n.Conn.conn.SendMsg(NewCloseMsg("", Status_Err))
				return err
			}

		case MsgEnd:
			info, err := fs.Stat()
			if err != nil {
				err = fmt.Errorf("fs.Stat err: file.size %v rece size %v \n", info.Size(), m.FileSize)
				n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Err))
				return err
			}

			if info.Size() != int64(m.FileSize) {
				err = fmt.Errorf("file.size %v rece size %v \n", info.Size(), m.FileSize)
				n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Err))
				return err
			}
			fs.Close()
			fs = nil
			n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Ok))

		case MsgNotify:
			n.Conn.waitNotify <- m.Bytes[0] == byte(Status_Ok)

		case MsgClose:
			n.Conn.conn.Close()
			if m.Bytes[0] != byte(Status_Ok) {
				return fmt.Errorf("closed due to error")
			}
			return nil

		default:
			return errors.New("Invalid msgType")
		}
	}
	return err
}
