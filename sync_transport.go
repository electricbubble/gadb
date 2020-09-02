package gadb

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"
)

type syncTransport struct {
	sock        net.Conn
	readTimeout time.Duration
}

func newSyncTransport(sock net.Conn, readTimeout time.Duration) syncTransport {
	return syncTransport{sock: sock, readTimeout: readTimeout}
}

func (sync syncTransport) Send(command, data string) (err error) {
	if len(command) != 4 {
		return errors.New("sync commands must have length 4")
	}
	msg := bytes.NewBufferString(command)
	if err = binary.Write(msg, binary.LittleEndian, int32(len(data))); err != nil {
		return fmt.Errorf("sync transport write: %w", err)
	}
	msg.WriteString(data)

	debugLog(fmt.Sprintf("--> %s", msg.String()))
	return _send(sync.sock, msg.Bytes())
}

func (sync syncTransport) SendStream(reader io.Reader) (err error) {
	syncMaxChunkSize := 64 * 1024
	for err == nil {
		tmp := make([]byte, syncMaxChunkSize)
		var n int
		n, err = reader.Read(tmp)
		if err == io.EOF {
			err = nil
			break
		}
		if err == nil {
			err = sync.sendChunk(tmp[:n])
		}
	}

	return
}

func (sync syncTransport) SendStatus(statusCode string, n uint32) (err error) {
	msg := bytes.NewBufferString(statusCode)
	if err = binary.Write(msg, binary.LittleEndian, n); err != nil {
		return fmt.Errorf("sync transport write: %w", err)
	}
	debugLog(fmt.Sprintf("--> %s", msg.String()))
	return _send(sync.sock, msg.Bytes())
}

func (sync syncTransport) sendChunk(buffer []byte) (err error) {
	msg := bytes.NewBufferString("DATA")
	if err = binary.Write(msg, binary.LittleEndian, int32(len(buffer))); err != nil {
		return fmt.Errorf("sync transport write: %w", err)
	}
	debugLog(fmt.Sprintf("--> %s ......", msg.String()))
	msg.Write(buffer)
	return _send(sync.sock, msg.Bytes())
}

func (sync syncTransport) VerifyStatus() (err error) {
	var status string
	if status, err = sync.ReadStringN(4); err != nil {
		return err
	}
	// debugLog(fmt.Sprintf("<-- %s", status))
	log := bytes.NewBufferString(fmt.Sprintf("<-- %s", status))
	defer func() {
		debugLog(log.String())
	}()

	var tmpUin32 int32
	if err = binary.Read(sync.sock, binary.LittleEndian, &tmpUin32); err != nil {
		return fmt.Errorf("sync transport read (status): %w", err)
	}
	log.WriteString(fmt.Sprintf(" %d\t", tmpUin32))

	var msg string
	if msg, err = sync.ReadStringN(int(tmpUin32)); err != nil {
		return err
	}
	log.WriteString(msg)

	if status == "FAIL" {
		err = fmt.Errorf("sync verify status: %s", msg)
		return
	}

	if status != "OKAY" {
		err = fmt.Errorf("sync verify status: Unknown error: %s", msg)
		return
	}

	return
}

func (sync syncTransport) ReadDirectoryEntry() (entry DeviceFileInfo, err error) {
	var status string
	if status, err = sync.ReadStringN(4); err != nil {
		return DeviceFileInfo{}, err
	}

	log := bytes.NewBufferString(fmt.Sprintf("<-- %s", status))
	defer func() {
		debugLog(log.String())
	}()

	if status == "DONE" {
		return
	}

	log = bytes.NewBufferString(fmt.Sprintf("<-- %s\t", status))

	if err = binary.Read(sync.sock, binary.LittleEndian, &entry.Mode); err != nil {
		return DeviceFileInfo{}, fmt.Errorf("sync transport read (mode): %w", err)
	}
	log.WriteString(entry.Mode.String() + "\t")

	if err = binary.Read(sync.sock, binary.LittleEndian, &entry.Size); err != nil {
		return DeviceFileInfo{}, fmt.Errorf("sync transport read (size): %w", err)
	}
	log.WriteString(fmt.Sprintf("%10d", entry.Size) + "\t")

	var tmpUin32 uint32
	if err = binary.Read(sync.sock, binary.LittleEndian, &tmpUin32); err != nil {
		return DeviceFileInfo{}, fmt.Errorf("sync transport read (time): %w", err)
	}
	entry.LastModified = time.Unix(int64(tmpUin32), 0)
	log.WriteString(entry.LastModified.String() + "\t")

	if err = binary.Read(sync.sock, binary.LittleEndian, &tmpUin32); err != nil {
		return DeviceFileInfo{}, fmt.Errorf("sync transport read (file name length): %w", err)
	}
	log.WriteString(strconv.Itoa(int(tmpUin32)) + "\t")

	if entry.Name, err = sync.ReadStringN(int(tmpUin32)); err != nil {
		return DeviceFileInfo{}, fmt.Errorf("sync transport read (file name): %w", err)
	}
	log.WriteString(entry.Name + "\t")

	return
}

func (sync syncTransport) ReadStringN(size int) (s string, err error) {
	var raw []byte
	if raw, err = sync.ReadBytesN(size); err != nil {
		return "", err
	}
	return string(raw), nil
}

func (sync syncTransport) ReadBytesN(size int) (raw []byte, err error) {
	_ = sync.sock.SetReadDeadline(time.Now().Add(time.Second * sync.readTimeout))
	return _readN(sync.sock, size)
}

func (sync syncTransport) Close() (err error) {
	if sync.sock == nil {
		return nil
	}
	return sync.sock.Close()
}
