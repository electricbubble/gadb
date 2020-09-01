package gadb

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

type DeviceState string

const (
	StateUnknown      DeviceState = "UNKNOWN"
	StateOnline       DeviceState = "online"
	StateOffline      DeviceState = "offline"
	StateDisconnected DeviceState = "disconnected"
)

var deviceStateStrings = map[string]DeviceState{
	"":        StateDisconnected,
	"offline": StateOffline,
	"device":  StateOnline,
}

func deviceStateConv(k string) (deviceState DeviceState) {
	var ok bool
	if deviceState, ok = deviceStateStrings[k]; !ok {
		return StateUnknown
	}
	return
}

type Device struct {
	adbClient Client
	serial    string
	attrs     map[string]string
}

func (d Device) Product() string {
	return d.attrs["product"]
}

func (d Device) Model() string {
	return d.attrs["model"]
}

func (d Device) Usb() string {
	return d.attrs["usb"]
}

func (d Device) transportId() string {
	return d.attrs["transport_id"]
}

func (d Device) DeviceInfo() map[string]string {
	return d.attrs
}

func (d Device) Serial() string {
	// 	raw, err := d.adbClient.executeCommand(fmt.Sprintf("host-serial:%s:get-serialno", d.serial))
	return d.serial
}

func (d Device) IsUsb() bool {
	return d.Usb() != ""
}

func (d Device) State() (DeviceState, error) {
	raw, err := d.adbClient.executeCommand(fmt.Sprintf("host-serial:%s:get-state", d.serial))
	return deviceStateConv(string(raw)), err
}

func (d Device) DevicePath() (string, error) {
	raw, err := d.adbClient.executeCommand(fmt.Sprintf("host-serial:%s:get-devpath", d.serial))
	return string(raw), err
}

func (d Device) Forward(localPort, remotePort int, noRebind ...bool) (err error) {
	command := ""
	local := fmt.Sprintf("tcp:%d", localPort)
	remote := fmt.Sprintf("tcp:%d", remotePort)
	if len(noRebind) != 0 && noRebind[0] {
		command = fmt.Sprintf("host-serial:%s:forward:norebind:%s;%s", d.serial, local, remote)
	} else {
		command = fmt.Sprintf("host-serial:%s:forward:%s;%s", d.serial, local, remote)
	}
	_, err = d.adbClient.executeCommand(command, true)
	return
}

func (d Device) ForwardList() (string, error) {
	raw, err := d.adbClient.executeCommand(fmt.Sprintf("host-serial:%s:list-forward", d.serial))
	lines := strings.Split(string(raw), "\n")
	result := bytes.NewBufferString("")
	for i := range lines {
		line := lines[i]
		if line == "" {
			continue
		}
		field := strings.Fields(line)[0]
		if field == d.serial {
			result.WriteString(line)
		}
	}
	return result.String(), err
}

func (d Device) ForwardKill(localPort int) (err error) {
	local := fmt.Sprintf("tcp:%d", localPort)
	_, err = d.adbClient.executeCommand(fmt.Sprintf("host-serial:%s:killforward:%s", d.serial, local), true)
	return
}

func (d Device) executeCommand(command string, onlyReadStatus ...bool) (raw []byte, err error) {
	if len(onlyReadStatus) == 0 {
		onlyReadStatus = []bool{false}
	}

	var transport Transport
	if transport, err = newTransport(fmt.Sprintf("%s:%d", d.adbClient.host, d.adbClient.port)); err != nil {
		return nil, err
	}
	defer func() { _ = transport.Close() }()

	if err = transport.Send(fmt.Sprintf("host:transport:%s", d.serial)); err != nil {
		return nil, err
	}
	if _, err = transport.ReadStatus(); err != nil {
		return nil, err
	}

	if err = transport.Send(command); err != nil {
		return nil, err
	}

	if onlyReadStatus[0] {
		if _, err = transport.ReadStatus(); err != nil {
			return nil, err
		}
		return
	}

	raw, err = transport.ReadAll()
	return
}

func (d Device) RunShellCommand(cmd string, args ...string) (string, error) {
	if len(args) > 0 {
		cmd = fmt.Sprintf("%s %s", cmd, strings.Join(args, " "))
	}
	if strings.TrimSpace(cmd) == "" {
		return "", errors.New("adb shell: command cannot be empty")
	}
	raw, err := d.executeCommand(fmt.Sprintf("shell:%s", cmd))
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func (d Device) EnableAdbOverTCP(port ...int) (err error) {
	if len(port) == 0 {
		port = []int{AdbDaemonPort}
	}

	_, err = d.executeCommand(fmt.Sprintf("tcpip:%d", port[0]), true)

	return
}
