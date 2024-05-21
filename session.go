// Copyright 2024 The ChromiumOS Authors
// Use of this source code is governed by a MIT License that can be
// found in the LICENSE file.

package gadb

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
)

// A Session represents a connection to a remote command or shell.
type Session struct {
	// Stdin specifies the remote process's standard input.
	// If Stdin is nil, the remote process reads from an empty
	// bytes.Buffer.
	Stdin io.Reader

	// Stdout and Stderr specify the remote process's standard
	// output and error.
	//
	// If either is nil, Run connects the corresponding file
	// descriptor to an instance of io.Discard. There is a
	// fixed amount of buffering that is shared for the two streams.
	// If either blocks it may eventually cause the remote
	// command to block.
	Stdout io.Writer
	Stderr io.Writer

	transport      *transport
	errorChan      chan error
	abort          bool
	handlesToClose []io.Closer
}

// ExitMissingError is returned if a session is torn down cleanly, but the server sends no confirmation of the exit status.
type ExitMissingError struct{}

// Error returns error message.
func (e *ExitMissingError) Error() string {
	return "command exited without exit status"
}

// An ExitError reports unsuccessful completion of a remote command.
type ExitError struct {
	Waitmsg
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("unexpected error code %d", e.exitStatus)
}

// Waitmsg stores the information about an exited remote command as reported by Wait.
type Waitmsg struct {
	exitStatus int
}

// ExitStatus returns the exit status of the remote command.
func (w Waitmsg) ExitStatus() int {
	return w.exitStatus
}

// NewSession opens a new Session for this client. (A session is a remote execution of a program.)
func (d Device) NewSession() (*Session, error) {
	tp, err := d.createDeviceTransport()
	if err != nil {
		return nil, fmt.Errorf("failed to create transport: %w", err)
	}
	return &Session{
		transport: &tp,
	}, nil
}

// Close frees resources associated with this Session, and aborts any running command.
func (s *Session) Close() error {
	var err error
	s.abort = true
	if s.transport != nil {
		err = s.transport.Close()
	}
	s.transport = nil
	err = errors.Join(err, s.closeFiles())
	return err
}

func (s *Session) closeFiles() error {
	var err error
	for _, f := range s.handlesToClose {
		fErr := f.Close()
		err = errors.Join(err, fErr)
	}
	s.handlesToClose = nil
	return err
}

// CombinedOutput runs cmd on the remote host and returns its combined standard output and standard error.
func (s *Session) CombinedOutput(cmd string) ([]byte, error) {
	if s.Stdout != nil {
		return nil, errors.New("can't set Stdout and call CombinedOutput()")
	}
	if s.Stderr != nil {
		return nil, errors.New("can't set Stdout and call CombinedOutput()")
	}
	var output bytes.Buffer
	s.Stdout = &output
	s.Stderr = &output
	if err := s.Run(cmd); err != nil {
		return output.Bytes(), err
	}
	return output.Bytes(), nil
}

// Output runs cmd on the remote host and returns its standard output.
func (s *Session) Output(cmd string) ([]byte, error) {
	if s.Stdout != nil {
		return nil, errors.New("can't set Stdout and call Output()")
	}
	var output bytes.Buffer
	s.Stdout = &output
	if err := s.Run(cmd); err != nil {
		return output.Bytes(), err
	}
	return output.Bytes(), nil
}

// Run runs cmd on the remote host.
func (s *Session) Run(cmd string) error {
	if err := s.Start(cmd); err != nil {
		return err
	}
	if err := s.Wait(); err != nil {
		return err
	}
	return nil
}

// Start runs cmd on the remote host.
func (s *Session) Start(cmd string) error {
	if s.errorChan != nil {
		return errors.New("Start() already called")
	}

	if err := s.transport.Send(fmt.Sprintf("shell,v2,raw:%s", cmd)); err != nil {
		return fmt.Errorf("failed to send shell cmd: %w", err)
	}
	if err := s.transport.VerifyResponse(); err != nil {
		return fmt.Errorf("failed to verify shell cmd: %w", err)
	}
	shellTp, err := s.transport.CreateShellTransport()
	if err != nil {
		return fmt.Errorf("failed to create shell transport: %w", err)
	}

	s.errorChan = make(chan error)
	s.abort = false
	// Copy stdin to remote command
	if s.Stdin != nil {
		go func() {
			buffer := make([]byte, 1024)
			for !s.abort {
				n, err := s.Stdin.Read(buffer)
				if err == io.EOF {
					if err := shellTp.Send(shellCloseStdin, []byte{}); err != nil {
						s.errorChan <- fmt.Errorf("failed to close stdin: %w", err)
					}
					return
				}
				if err != nil {
					s.errorChan <- fmt.Errorf("failed to copy stdin: %w", err)
					return
				}
				shellTp.Send(shellStdin, buffer[0:n])
			}
		}()
	} else {
		if err := shellTp.Send(shellCloseStdin, []byte{}); err != nil {
			return fmt.Errorf("failed to close stdin: %w", err)
		}
	}
	go func() {
		for !s.abort {
			msgType, msg, err := shellTp.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				s.errorChan <- fmt.Errorf("failed to read shell msg: %w", err)
				return
			}
			switch msgType {
			case shellStdout: // stdout
				if s.Stdout != nil {
					if _, err := s.Stdout.Write(msg); err != nil {
						s.errorChan <- fmt.Errorf("failed to write stdout: %w", err)
						return
					}
				}
			case shellStderr: // stderr
				if s.Stderr != nil {
					if _, err := s.Stderr.Write(msg); err != nil {
						s.errorChan <- fmt.Errorf("failed to write stderr: %w", err)
						return
					}
				}
			case shellExit: // exit
				exitCode := int(msg[0])
				err := s.closeFiles()
				if err != nil {
					s.errorChan <- fmt.Errorf("failed to close files: %w", err)
				}
				if exitCode == 0 {
					s.errorChan <- nil
				} else {
					s.errorChan <- &ExitError{
						Waitmsg: Waitmsg{exitStatus: exitCode},
					}
				}
				return
			default:
				s.errorChan <- fmt.Errorf("unexpected shell message %d: %w", msgType, err)
				return
			}
		}
		s.errorChan <- &ExitMissingError{}
	}()
	return nil
}

// StderrPipe returns a pipe that will be connected to the remote command's standard error when the command starts.
func (s *Session) StderrPipe() (io.Reader, error) {
	if s.Stderr != nil {
		return nil, errors.New("can't set Stderr and call StderrPipe()")
	}
	if s.errorChan != nil {
		return nil, errors.New("StderrPipe called after Start()")
	}
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	s.Stderr = pw
	s.handlesToClose = append(s.handlesToClose, pw)
	return pr, nil
}

// StdinPipe returns a pipe that will be connected to the remote command's standard input when the command starts.
func (s *Session) StdinPipe() (io.WriteCloser, error) {
	if s.Stdin != nil {
		return nil, errors.New("can't set Stdin and call StdinPipe()")
	}
	if s.errorChan != nil {
		return nil, errors.New("StdinPipe called after Start()")
	}
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	s.Stdin = pr
	s.handlesToClose = append(s.handlesToClose, pr)
	return pw, nil
}

// StdoutPipe returns a pipe that will be connected to the remote command's standard output when the command starts.
func (s *Session) StdoutPipe() (io.Reader, error) {
	if s.Stdout != nil {
		return nil, errors.New("can't set Stdout and call StdoutPipe()")
	}
	if s.errorChan != nil {
		return nil, errors.New("StdoutPipe called after Start()")
	}
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	s.Stdout = pw
	s.handlesToClose = append(s.handlesToClose, pw)
	return pr, nil
}

// Wait waits for the remote command to exit.
func (s *Session) Wait() error {
	if s.errorChan == nil {
		return errors.New("Wait() called before Start()")
	}
	if s.abort {
		return errors.New("Wait() called twice or after Close()")
	}
	backgroundErr := <-s.errorChan
	if err := s.Close(); err != nil {
		return errors.Join(backgroundErr, err)
	}
	return backgroundErr
}
