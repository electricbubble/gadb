// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a MIT License that can be
// found in the LICENSE file.

package gadb

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
)

type shellTransport struct {
	sock        net.Conn
	readTimeout time.Duration
}

// Shell protocol message types.
type shellMessageType byte

const (
	shellStdin      shellMessageType = 0
	shellStdout     shellMessageType = 1
	shellStderr     shellMessageType = 2
	shellExit       shellMessageType = 3
	shellCloseStdin shellMessageType = 4
)

func newShellTransport(sock net.Conn, readTimeout time.Duration) shellTransport {
	return shellTransport{sock: sock, readTimeout: readTimeout}
}

// Send creates and sends a packet over the shell protocol.
func (s *shellTransport) Send(command shellMessageType, data []byte) (err error) {
	msg := new(bytes.Buffer)
	if err := msg.WriteByte(byte(command)); err != nil {
		return fmt.Errorf("shell transport write: %w", err)
	}
	if err = binary.Write(msg, binary.LittleEndian, int32(len(data))); err != nil {
		return fmt.Errorf("shell transport write: %w", err)
	}
	if _, err := msg.Write(data); err != nil {
		return fmt.Errorf("shell transport write: %w", err)
	}

	debugLog(fmt.Sprintf("--> %v", msg.Bytes()))
	return _send(s.sock, msg.Bytes())
}

func (s *shellTransport) Read() (command shellMessageType, data []byte, err error) {
	err = binary.Read(s.sock, binary.LittleEndian, &command)
	if err == io.EOF {
		return 255, nil, err
	}
	if err != nil {
		return 255, nil, fmt.Errorf("failed to read response msg type: %w", err)
	}
	var msgLen uint32
	err = binary.Read(s.sock, binary.LittleEndian, &msgLen)
	if err != nil {
		return command, nil, fmt.Errorf("failed to read response msg len: %w", err)
	}
	data, err = s.ReadBytesN(int(msgLen))
	if err != nil {
		return command, data, fmt.Errorf("failed to read response msg body: %w", err)
	}
	return command, data, nil
}

func (s *shellTransport) ReadBytesN(size int) (raw []byte, err error) {
	_ = s.sock.SetReadDeadline(time.Time{})
	return _readN(s.sock, size)
}

func (s *shellTransport) Close() (err error) {
	if s.sock == nil {
		return nil
	}
	return s.sock.Close()
}
