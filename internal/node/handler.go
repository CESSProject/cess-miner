package node

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
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
		stop:       make(chan struct{}),
	}
	return n
}

func (n *Node) Start() {
	n.Conn.conn.HandlerLoop()
	n.handler()
	time.Sleep(time.Second)
	Out.Sugar().Infof("Close a conn: %v", n.Conn.conn.GetRemoteAddr())
	n = nil
	runtime.Goexit()
}

func (n *Node) handler() error {
	var err error
	var fs *os.File
	var returnFile *os.File

	defer func() {
		if err != nil {
			time.Sleep(time.Second)
		}
		err := recover()
		if err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
		n.Conn.conn.Close()
		close(n.Conn.stop)
		close(n.Conn.waitNotify)
		if fs != nil {
			fs.Close()
		}
		if returnFile != nil {
			returnFile.Close()
		}
	}()

	for !n.Conn.conn.IsClose() {
		m, ok := n.Conn.conn.GetMsg()
		if !ok {
			return fmt.Errorf("close by connect")
		}

		if m == nil {
			continue
		}

		switch m.MsgType {
		case MsgHead:
			// Verify signature
			ok, err := VerifySign(m.Pubkey, m.SignMsg, m.Sign)
			if err != nil {
				n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Err))
				return err
			}
			if !ok {
				n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Err))
				return err
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
			if err != nil {
				n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Err))
				return err
			}
			if !ok {
				n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Err))
				return err
			}
			returnFile, err = os.Open(filepath.Join(n.Conn.fileDir, m.FileName))
			if err != nil {
				n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Err))
				return err
			}

			n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Ok))

		case MsgRecvFile:
			if returnFile == nil {
				n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Err))
				return nil
			}
			fileInfo, _ := returnFile.Stat()
			for !n.Conn.conn.IsClose() {
				readBuf := bytesPool.Get().([]byte)
				num, err := returnFile.Read(readBuf)
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

func (n *Node) NewClient(conn NetConn, fileDir string, files []string) Client {
	n.Conn = &ConMgr{
		conn:       conn,
		fileDir:    fileDir,
		sendFiles:  files,
		waitNotify: make(chan bool, 1),
		stop:       make(chan struct{}),
	}
	return n
}

func (n *Node) SendFile(fid string, pkey, signmsg, sign []byte) error {
	var err error
	n.Conn.conn.HandlerLoop()
	go func() {
		_ = n.handler()
	}()
	err = n.Conn.sendFile(fid, pkey, signmsg, sign)
	return err
}

func (c *ConMgr) sendFile(fid string, pkey, signmsg, sign []byte) error {
	defer func() {
		_ = c.conn.Close()
	}()

	var err error
	var lastmatrk bool
	for i := 0; i < len(c.sendFiles); i++ {
		err = c.sendSingleFile(c.sendFiles[i], fid, pkey, signmsg, sign)
		if err != nil {
			return err
		}
		if lastmatrk {
			for _, v := range c.sendFiles {
				os.Remove(v)
			}
		}
	}

	c.conn.SendMsg(NewCloseMsg(c.fileName, Status_Ok))
	return err
}

func (c *ConMgr) sendSingleFile(filePath string, fid string, pkey, signmsg, sign []byte) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}

	defer func() {
		if file != nil {
			_ = file.Close()
		}
	}()
	fileInfo, _ := file.Stat()

	m := NewHeadMsg(fileInfo.Name(), fid, pkey, signmsg, sign)
	c.conn.SendMsg(m)

	timer := time.NewTimer(5 * time.Second)
	select {
	case ok := <-c.waitNotify:
		if !ok {
			return fmt.Errorf("send err")
		}
	case <-timer.C:
		return fmt.Errorf("wait server msg timeout")
	}

	for !c.conn.IsClose() {
		readBuf := bytesPool.Get().([]byte)

		n, err := file.Read(readBuf)
		if err != nil && err != io.EOF {
			return err
		}

		if n == 0 {
			break
		}

		c.conn.SendMsg(NewFileMsg(c.fileName, readBuf[:n]))
	}

	c.conn.SendMsg(NewEndMsg(c.fileName, uint64(fileInfo.Size())))
	waitTime := fileInfo.Size() / 1024 / 10
	if waitTime < 5 {
		waitTime = 5
	}

	timer = time.NewTimer(time.Second * time.Duration(waitTime))
	select {
	case ok := <-c.waitNotify:
		if !ok {
			return fmt.Errorf("send err")
		}
	case <-timer.C:
		return fmt.Errorf("wait server msg timeout")
	}

	return nil
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

func CalcFileBlockSizeAndScanSize(fsize int64) (int64, int64) {
	var (
		blockSize     int64
		scanBlockSize int64
	)
	if fsize < configs.SIZE_1KiB {
		return fsize, fsize
	}
	if fsize > math.MaxUint32 {
		blockSize = math.MaxUint32
		scanBlockSize = blockSize / 8
		return blockSize, scanBlockSize
	}
	blockSize = fsize / 16
	scanBlockSize = blockSize / 8
	return blockSize, scanBlockSize
}
