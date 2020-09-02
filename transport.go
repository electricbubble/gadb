package gadb

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"strconv"
	"time"
)

var ErrConnBroken = errors.New("socket connection broken")

// type replyStatus string
//
// const (
// 	statusSuccess  replyStatus = "OKAY"
// 	StatusFailure  replyStatus = "FAIL"
// 	StatusSyncData replyStatus = "DATA"
// 	StatusSyncDone replyStatus = "DONE"
// )

var DefaultAdbReadTimeout = 10

type transport struct {
	sock        net.Conn
	readTimeout time.Duration
}

func newTransport(address string, readTimeout ...int) (tp transport, err error) {
	if len(readTimeout) == 0 {
		readTimeout = []int{DefaultAdbReadTimeout}
	}
	tp.readTimeout = time.Duration(readTimeout[0])
	if tp.sock, err = net.Dial("tcp", address); err != nil {
		err = fmt.Errorf("adb transport: %w", err)
	}
	return
}

func (t transport) Send(command string) (err error) {
	msg := fmt.Sprintf("%04x%s", len(command), command)
	debugLog(fmt.Sprintf("--> %s", command))
	return _send(t.sock, []byte(msg))
}

func (t transport) VerifyResponse() (err error) {
	var status string
	if status, err = t._readStringN(4); err != nil {
		return err
	}
	if status == "OKAY" {
		debugLog(fmt.Sprintf("<-- %s", status))
		return nil
	}

	var sError string
	if sError, err = t.UnpackString(); err != nil {
		return err
	}
	err = fmt.Errorf("command failed: %s", sError)
	debugLog(fmt.Sprintf("<-- %s %s", status, sError))
	return
}

func (t transport) ReadStringAll() (s string, err error) {
	var raw []byte
	raw, err = t.ReadBytesAll()
	return string(raw), err
}

func (t transport) ReadBytesAll() (raw []byte, err error) {
	raw, err = ioutil.ReadAll(t.sock)
	debugLog(fmt.Sprintf("\r%s", raw))
	return
}

func (t transport) UnpackString() (s string, err error) {
	var raw []byte
	raw, err = t.UnpackBytes()
	return string(raw), err
}

func (t transport) UnpackBytes() (raw []byte, err error) {
	var length string
	if length, err = t._readStringN(4); err != nil {
		return nil, err
	}
	var size int64
	if size, err = strconv.ParseInt(length, 16, 64); err != nil {
		return nil, err
	}

	raw, err = t._readBytesN(int(size))
	debugLog(fmt.Sprintf("\r%s", raw))
	return
}

func (t transport) _readStringN(size int) (s string, err error) {
	var raw []byte
	if raw, err = t._readBytesN(size); err != nil {
		return "", err
	}
	return string(raw), nil
}

func (t transport) _readBytesN(size int) (raw []byte, err error) {
	_ = t.sock.SetReadDeadline(time.Now().Add(time.Second * t.readTimeout))

	raw = make([]byte, 0, size)
	for len(raw) < size {
		buf := make([]byte, size-len(raw))
		var n int
		if n, err = t.sock.Read(buf); err != nil {
			return nil, err
		}
		if n == 0 {
			return nil, ErrConnBroken
		}
		raw = append(raw, buf...)
	}
	return
}

func (t transport) Close() (err error) {
	if t.sock == nil {
		return nil
	}
	return t.sock.Close()
}

func _send(conn net.Conn, msg []byte) (err error) {
	for totalSent := 0; totalSent < len(msg); {
		var sent int
		if sent, err = conn.Write(msg[totalSent:]); err != nil {
			return err
		}
		if sent == 0 {
			return ErrConnBroken
		}
		totalSent += sent
	}
	return
}
