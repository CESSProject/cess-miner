package node

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/CESSProject/cess-bucket/configs"
)

func (n *Node) NewServer(conn NetConn, fileDir string) Server {
	n.Conn = &ConMgr{
		conn:    conn,
		fileDir: fileDir,
		stop:    make(chan struct{}),
	}
	return n
}

func (n *Node) Start() {
	n.Conn.conn.HandlerLoop()
	err := n.handler()
	if err != nil {
		log.Println(err)
	}
}

func (n *Node) handler() error {
	var fs *os.File
	var fillerFs *os.File
	var returnFile *os.File
	var err error

	defer func() {
		if fs != nil {
			_ = fs.Close()
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
				fmt.Println("VerifySign err: ", err)
				n.Conn.conn.SendMsg(NewNotifyMsg(n.Conn.fileName, Status_Err))
				return err
			}
			if !ok {
				fmt.Println("Verify Sign failed: ", err)
				n.Conn.conn.SendMsg(NewNotifyMsg(n.Conn.fileName, Status_Err))
				return err
			}

			n.Conn.fileName = m.FileName
			fmt.Println("recv head fileName is", n.Conn.fileName)

			fs, err = os.OpenFile(filepath.Join(n.Conn.fileDir, n.Conn.fileName), os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
			if err != nil {
				fmt.Println("os.Create err =", err)
				n.Conn.conn.SendMsg(NewNotifyMsg(n.Conn.fileName, Status_Err))
				return err
			}
			fmt.Println("Create and open file: ", filepath.Join(n.Conn.fileDir, n.Conn.fileName))
			fmt.Println("send head is ok")

			n.Conn.conn.SendMsg(NewNotifyMsg(n.Conn.fileName, Status_Ok))
		case MsgRecvHead:
			// Verify signature
			ok, err := VerifySign(m.Pubkey, m.SignMsg, m.Sign)
			if err != nil {
				fmt.Println("VerifySign err: ", err)
				n.Conn.conn.SendMsg(NewNotifyMsg(n.Conn.fileName, Status_Err))
				return err
			}
			if !ok {
				fmt.Println("Verify Sign failed: ", err)
				n.Conn.conn.SendMsg(NewNotifyMsg(n.Conn.fileName, Status_Err))
				return err
			}
			returnFile, err = os.Open(filepath.Join(n.Conn.fileDir, m.FileName))
			if err != nil {
				fmt.Println("file not found: ", err)
				n.Conn.conn.SendMsg(NewNotifyMsg(n.Conn.fileName, Status_Err))
				return err
			}

			n.Conn.conn.SendMsg(NewNotifyMsg(m.FileName, Status_Ok))

		case MsgRecvFile:
			if returnFile == nil {
				fmt.Println(n.Conn.fileName, "recv file is not open!")
				n.Conn.conn.SendMsg(NewNotifyMsg(m.FileName, Status_Err))
				return nil
			}
			fileInfo, _ := returnFile.Stat()
			for !n.Conn.conn.IsClose() {
				readBuf := BytesPool.Get().([]byte)
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
			n.Conn.conn.SendMsg(NewNotifyMsg(m.FileName, Status_Ok))
		case MsgFile:
			if fs == nil {
				fmt.Println(n.Conn.fileName, "file is not open!")
				n.Conn.conn.SendMsg(NewCloseMsg(n.Conn.fileName, Status_Err))
				return errors.New("file is not open !")
			}
			_, err = fs.Write(m.Bytes)
			if err != nil {
				fmt.Println("file.Write err =", err)
				n.Conn.conn.SendMsg(NewCloseMsg(n.Conn.fileName, Status_Err))
				return err
			}
		case MsgFiller:
			if fillerFs == nil {
				fmt.Println("Recv msg filler and creat filler: ", m.FileName)
				fillerFs, err = os.OpenFile(filepath.Join(n.Conn.fileDir, m.FileName), os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.ModePerm)
				if err != nil {
					fmt.Println(n.Conn.fileName, "create filler err!")
					n.Conn.conn.SendMsg(NewCloseMsg(m.FileName, Status_Err))
					return errors.New("file is not open !")
				}
			}
			fillerFs.Write(m.Bytes)
		case MsgEnd:
			info, err := fs.Stat()
			if err != nil {
				err = fmt.Errorf("fs.Stat err: file.size %v rece size %v \n", info.Size(), m.FileSize)
				n.Conn.conn.SendMsg(NewCloseMsg(n.Conn.fileName, Status_Err))
				return err
			}

			if info.Size() != int64(m.FileSize) {
				err = fmt.Errorf("file.size %v rece size %v \n", info.Size(), m.FileSize)
				n.Conn.conn.SendMsg(NewCloseMsg(n.Conn.fileName, Status_Err))
				return err
			}
			fs.Close()
			fs = nil
			fmt.Printf("save file %v is success \n", info.Name())
			n.Conn.conn.SendMsg(NewNotifyMsg(n.Conn.fileName, Status_Ok))

			fmt.Printf("close file %v is success \n", n.Conn.fileName)
		case MsgFillerEnd:
			fmt.Println("Recv msg filler end: ", m.FileName)
			fillerInfo, err := fillerFs.Stat()
			if err != nil {
				fillerFs.Close()
				fillerFs = nil
				err = fmt.Errorf("err: filler.size %v \n", m.FileSize)
				n.Conn.conn.SendMsg(NewNotifyMsg(m.FileHash, Status_Err))
				return err
			}
			fillerFs.Close()
			fillerFs = nil
			if fillerInfo.Size() != int64(m.FileSize) {
				err = fmt.Errorf("filler.size %v rece size %v \n", fillerInfo.Size(), m.FileSize)
				n.Conn.conn.SendMsg(NewNotifyMsg(n.Conn.fileName, Status_Err))
				return err
			}
			fmt.Printf("save filler %v is success \n", fillerInfo.Name())
			//n.Conn.conn.SendMsg(NewNotifyMsg(m.FileName, Status_Ok))
		case MsgNotify:
			fmt.Println("Recv msg notify: ", m.FileName)
			n.Conn.waitNotify <- m.Bytes[0] == byte(Status_Ok)
			if len(m.Bytes) > 1 {
				n.Conn.fillerId = string(m.Bytes[1:])
			}
		case MsgClose:
			fmt.Printf("recv close msg ....\n")
			n.Conn.conn.Close()
			if m.Bytes[0] != byte(Status_Ok) {
				return fmt.Errorf("server an error occurred")
			}
			return nil
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

func (n *Node) RecvFiller(pkey, signmsg, sign []byte) error {
	var err error
	n.Conn.conn.HandlerLoop()
	go func() {
		for !n.Conn.conn.IsClose() {
			_ = n.handler()
		}
	}()
	err = n.recvFiller(pkey, signmsg, sign)
	return err
}

func (c *ConMgr) sendFile(fid string, pkey, signmsg, sign []byte) error {
	defer func() {
		_ = c.conn.Close()
	}()

	var err error
	var lastmatrk bool
	for i := 0; i < len(c.sendFiles); i++ {
		if (i + 1) == len(c.sendFiles) {
			lastmatrk = true
		}
		err = c.sendSingleFile(c.sendFiles[i], fid, lastmatrk, pkey, signmsg, sign)
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

func (c *ConMgr) sendSingleFile(filePath string, fid string, lastmark bool, pkey, signmsg, sign []byte) error {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("open file err %v \n", err)
		return err
	}

	defer func() {
		if file != nil {
			_ = file.Close()
		}
	}()
	fileInfo, _ := file.Stat()

	log.Println("Ready to write file: ", filePath)
	m := NewHeadMsg(fileInfo.Name(), fid, lastmark, pkey, signmsg, sign)
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
		readBuf := BytesPool.Get().([]byte)

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

	log.Println("Send " + filePath + " file success...")
	return nil
}

func (n *Node) recvFiller(pkey, signmsg, sign []byte) error {
	defer func() {
		_ = n.Conn.conn.Close()
	}()

	log.Println("Send fillerHead")
	m := NewFillerHeadMsg(pkey, signmsg, sign)
	n.Conn.conn.SendMsg(m)
	timer := time.NewTimer(time.Second * 5)
	select {
	case ok := <-n.Conn.waitNotify:
		if !ok {
			return fmt.Errorf("send err")
		}
	case <-timer.C:
		return fmt.Errorf("wait server msg timeout")
	}

	log.Println("Recv a filler: ", n.Conn.fillerId)
	fillerHash := n.Conn.fillerId
	log.Println("Send filler tag req: ", fillerHash+".tag")
	m = NewFillerMsg(fillerHash + ".tag")
	n.Conn.conn.SendMsg(m)

	timer = time.NewTimer(time.Second * 10)
	select {
	case ok := <-n.Conn.waitNotify:
		if !ok {
			return fmt.Errorf("send err")
		}
	case <-timer.C:
		return fmt.Errorf("wait server msg timeout")
	}

	//log.Println("Recv a filler: ", n.Conn.fillerId)
	log.Println("Send filler req: ", fillerHash)
	m = NewFillerMsg(fillerHash)
	n.Conn.conn.SendMsg(m)

	timer = time.NewTimer(time.Second * 300)
	select {
	case ok := <-n.Conn.waitNotify:
		if !ok {
			return fmt.Errorf("send err")
		}
	case <-timer.C:
		return fmt.Errorf("wait server msg timeout")
	}

	log.Println("Send a close req: ", fillerHash)
	n.Conn.conn.SendMsg(NewCloseMsg(n.Conn.fillerId, Status_Ok))
	timer = time.NewTimer(time.Second * 3)
	select {
	case <-timer.C:
		return nil
	}
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
