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
	Out.Sugar().Infof("Close a conn: %v", n.Conn.conn.GetRemoteAddr())
	time.Sleep(time.Second * 3)
}

func (n *Node) handler() error {
	var (
		err          error
		fs           *os.File
		timeOutTimer *time.Timer
	)

	defer func() {
		n.Conn.conn.Close()
		close(n.Conn.waitNotify)
		if fs != nil {
			fs.Close()
		}
		if timeOutTimer != nil {
			timeOutTimer.Stop()
		}
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()

	for !n.Conn.conn.IsClose() {
		if timeOutTimer != nil {
			select {
			case <-timeOutTimer.C:
				return errors.New("Get msg timeout")
			default:
			}
		}

		m, ok := n.Conn.conn.GetMsg()
		if !ok {
			return fmt.Errorf("Getmsg failed")
		}

		if m == nil {
			if timeOutTimer == nil {
				timeOutTimer = time.NewTimer(configs.TCP_Time_WaitMsg)
			}
			continue
		} else {
			if timeOutTimer != nil {
				timeOutTimer.Stop()
				timeOutTimer = nil
			}
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
			switch cap(m.Bytes) {
			case configs.TCP_ReadBuffer:
				readBufPool.Put(m.Bytes)
			default:
			}
			// Verify signature
			ok, err := VerifySign(m.Pubkey, m.SignMsg, m.Sign)
			if err != nil || !ok {
				n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Err))
				time.Sleep(configs.TCP_Message_Interval)
				return errors.New("Signature error")
			}

			fs, err = os.Open(filepath.Join(n.Conn.fileDir, m.FileName))
			if err != nil {
				n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Err))
				time.Sleep(configs.TCP_Message_Interval)
				return err
			}

			n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Ok))

		case MsgRecvFile:
			switch cap(m.Bytes) {
			case configs.TCP_ReadBuffer:
				readBufPool.Put(m.Bytes)
			default:
			}
			if fs == nil {
				n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Err))
				time.Sleep(configs.TCP_Message_Interval)
				return errors.New("File not open")
			}
			fileInfo, _ := fs.Stat()
			readBuf := sendBufPool.Get().([]byte)
			defer func() {
				sendBufPool.Put(readBuf)
			}()

			for !n.Conn.conn.IsClose() {
				num, err := fs.Read(readBuf)
				if err != nil && err != io.EOF {
					return err
				}
				if num == 0 {
					break
				}
				n.Conn.conn.SendMsg(NewFileMsg(fileInfo.Name(), num, readBuf[:num]))
			}

			time.Sleep(time.Millisecond)
			n.Conn.conn.SendMsg(NewEndMsg(m.FileName, uint64(fileInfo.Size())))
			time.Sleep(time.Millisecond)
			n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Ok))

		case MsgFile:
			if fs == nil {
				n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Err))
				time.Sleep(configs.TCP_Message_Interval)
				n.Conn.conn.SendMsg(NewCloseMsg("", Status_Err))
				time.Sleep(configs.TCP_Message_Interval)
				return errors.New("file is not open !")
			}
			_, err = fs.Write(m.Bytes[:m.FileSize])
			if err != nil {
				n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Err))
				time.Sleep(configs.TCP_Message_Interval)
				n.Conn.conn.SendMsg(NewCloseMsg("", Status_Err))
				time.Sleep(configs.TCP_Message_Interval)
				return err
			}
			switch cap(m.Bytes) {
			case configs.TCP_ReadBuffer:
				readBufPool.Put(m.Bytes)
			default:
			}
		case MsgEnd:
			switch cap(m.Bytes) {
			case configs.TCP_ReadBuffer:
				readBufPool.Put(m.Bytes)
			default:
			}
			info, err := fs.Stat()
			if err != nil {
				err = fmt.Errorf("fs.Stat err: file.size %v rece size %v \n", info.Size(), m.FileSize)
				n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Err))
				time.Sleep(configs.TCP_Message_Interval)
				return err
			}

			if info.Size() != int64(m.FileSize) {
				err = fmt.Errorf("file.size %v rece size %v \n", info.Size(), m.FileSize)
				n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Err))
				time.Sleep(configs.TCP_Message_Interval)
				return err
			}
			fs.Close()
			fs = nil
			if m.FileType == FileType_filler && m.FileName == m.FileHash {
				fpath := ""
				if m.FileType == FileType_file {
					fpath = filepath.Join(configs.FilesDir, m.FileName)
				} else {
					fpath = filepath.Join(configs.SpaceDir, m.FileName)
				}

				hash, err := tools.CalcFileHash(fpath)
				if err != nil || hash != m.FileHash {
					os.Remove(fpath)
					os.Remove(fpath + ".tag")
					n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Err))
					time.Sleep(configs.TCP_Message_Interval)
					return err
				}
			}
			n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Ok))

		case MsgNotify:
			n.Conn.waitNotify <- m.Bytes[0] == byte(Status_Ok)
			switch cap(m.Bytes) {
			case configs.TCP_ReadBuffer:
				readBufPool.Put(m.Bytes)
			default:
			}

		case MsgClose:
			switch cap(m.Bytes) {
			case configs.TCP_ReadBuffer:
				readBufPool.Put(m.Bytes)
			default:
			}
			return errors.New("Close message")

		default:
			switch cap(m.Bytes) {
			case configs.TCP_ReadBuffer:
				readBufPool.Put(m.Bytes)
			default:
			}
			return errors.New("Invalid msgType")
		}
	}
	return err
}
