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

type ReplyStatus string

const (
	StatusSuccess  ReplyStatus = "OKAY"
	StatusFailure  ReplyStatus = "FAIL"
	StatusSyncData ReplyStatus = "DATA"
	StatusSyncDone ReplyStatus = "DONE"
)

type Transport struct {
	sock        net.Conn
	readTimeout time.Duration
}

func newTransport(address string, readTimeout ...int) (transport Transport, err error) {
	if len(readTimeout) == 0 {
		readTimeout = []int{10}
	}
	transport.readTimeout = time.Duration(readTimeout[0])
	if transport.sock, err = net.Dial("tcp", address); err != nil {
		err = fmt.Errorf("adb transport: %w", err)
	}
	return
}

func (t Transport) Send(command string) (err error) {
	msg := fmt.Sprintf("%04x%s", len(command), command)
	debugLog(fmt.Sprintf("--> %s", command))
	return t._send([]byte(msg))
}

func (t Transport) ReadStatus() (status ReplyStatus, err error) {
	var recvMsg []byte
	if recvMsg, err = t._recvN(4); err != nil {
		return "", err
	}
	status = ReplyStatus(recvMsg)
	if status == StatusFailure {
		if recvMsg, err = t._recv(); err != nil {
		}
		err = fmt.Errorf("command failed: %s", recvMsg)
	}
	debugLog(fmt.Sprintf("<-- %s", status))
	return
}

func (t Transport) ReadAll() (raw []byte, err error) {
	return ioutil.ReadAll(t.sock)
}

func (t Transport) Recv() (msg []byte, err error) {
	var status ReplyStatus
	if status, err = t.ReadStatus(); err != nil {
		return nil, err
	}

	switch status {
	// case StatusFailure:
	// 	if msg, err = t._recv(); err != nil {
	// 		return nil, err
	// 	}
	// 	err = fmt.Errorf("command failed: %s", msg)
	case StatusSuccess:
		if msg, err = t._recv(); err != nil {
			return nil, err
		}
	case StatusSyncData:
		fmt.Println("TODO", status)
	case StatusSyncDone:
		fmt.Println("TODO", status)
	default:
		fmt.Println("UNKNOWN STATUS:", status)
	}

	// debugLog(fmt.Sprintf("<-- %s\n%s", status, string(msg)))
	debugLog(fmt.Sprintf("\r%s", string(msg)))
	return
}

func (t Transport) _send(msg []byte) (err error) {
	for totalSent := 0; totalSent < len(msg); {
		var sent int
		if sent, err = t.sock.Write(msg[totalSent:]); err != nil {
			return err
		}
		if sent == 0 {
			return ErrConnBroken
		}
		totalSent += sent
	}
	return
}

func (t Transport) _recv() (msg []byte, err error) {
	if msg, err = t._recvN(4); err != nil {
		return nil, err
	}

	var size int64
	if size, err = strconv.ParseInt(string(msg), 16, 64); err != nil {
		return nil, err
	}

	if msg, err = t._recvN(int(size)); err != nil {
		return nil, err
	}

	return
}

func (t Transport) _recvN(size int) (msg []byte, err error) {
	_ = t.sock.SetReadDeadline(time.Now().Add(time.Second * t.readTimeout))

	msg = make([]byte, 0, size)
	for len(msg) < size {
		buf := make([]byte, size-len(msg))
		var n int
		if n, err = t.sock.Read(buf); err != nil {
			return nil, err
		}
		if n == 0 {
			return nil, ErrConnBroken
		}
		msg = append(msg, buf...)
	}
	return
}

func (t Transport) Close() (err error) {
	return t.sock.Close()
}
