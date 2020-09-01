package gadb

import (
	"fmt"
	"strconv"
	"strings"
)

const AdbServerPort = 5037
const AdbDaemonPort = 5555

type Client struct {
	host string
	port int
}

func NewClient() (Client, error) {
	return NewClientWith("localhost")
}

func NewClientWith(host string, port ...int) (adbClient Client, err error) {
	if len(port) == 0 {
		port = []int{AdbServerPort}
	}
	adbClient.host = host
	adbClient.port = port[0]

	var transport Transport
	if transport, err = newTransport(fmt.Sprintf("%s:%d", adbClient.host, adbClient.port)); err != nil {
		return Client{}, err
	}
	defer func() { _ = transport.Close() }()

	// if _, err = adbClient.ServerVersion(); err != nil {
	// 	return Client{}, err
	// }

	return
}

func (c Client) executeCommand(command string, onlyReadStatus ...bool) (raw []byte, err error) {
	if len(onlyReadStatus) == 0 {
		onlyReadStatus = []bool{false}
	}
	var transport Transport
	if transport, err = newTransport(fmt.Sprintf("%s:%d", c.host, c.port)); err != nil {
		return nil, err
	}
	defer func() { _ = transport.Close() }()

	if err = transport.Send(command); err != nil {
		return nil, err
	}

	if onlyReadStatus[0] {
		if _, err = transport.ReadStatus(); err != nil {
			return nil, err
		}
		return
	}

	raw, err = transport.Recv()
	return
}

func (c Client) ServerVersion() (version int, err error) {
	var msg []byte
	if msg, err = c.executeCommand("host:version"); err != nil {
		return 0, err
	}

	var v int64
	if v, err = strconv.ParseInt(string(msg), 16, 64); err != nil {
		return 0, err
	}

	version = int(v)
	return
}

func (c Client) DeviceSerialList() (serials []string, err error) {
	var raw []byte
	if raw, err = c.executeCommand("host:devices"); err != nil {
		return
	}

	lines := strings.Split(string(raw), "\n")
	serials = make([]string, 0, len(lines))

	for i := range lines {
		fields := strings.Fields(lines[i])
		if len(fields) < 2 {
			continue
		}
		serials = append(serials, fields[0])
	}

	return
}

func (c Client) DeviceList() (devices []Device, err error) {
	var raw []byte
	if raw, err = c.executeCommand("host:devices-l"); err != nil {
		return
	}

	lines := strings.Split(string(raw), "\n")
	devices = make([]Device, 0, len(lines))

	for i := range lines {
		if lines[i] == "" {
			continue
		}

		fields := strings.Fields(lines[i])
		if len(fields) < 5 {
			debugLog(fmt.Sprintf("can't parse: %s", lines[i]))
			continue
		}

		attr := fields[2:]
		attrs := map[string]string{}
		for _, field := range attr {
			split := strings.Split(field, ":")
			key, val := split[0], split[1]
			attrs[key] = val
		}

		devices = append(devices, Device{adbClient: c, serial: fields[0], attrs: attrs})
	}

	return
}

func (c Client) ForwardList() (string, error) {
	var raw []byte
	var err error
	if raw, err = c.executeCommand("host:list-forward"); err != nil {
		return "", err
	}
	return string(raw), nil
}

func (c Client) ForwardKillAll() (err error) {
	_, err = c.executeCommand("host:killforward-all", true)
	return
}

func (c Client) Connect(ip string, port ...int) (err error) {
	if len(port) == 0 {
		port = []int{AdbDaemonPort}
	}

	var raw []byte
	if raw, err = c.executeCommand(fmt.Sprintf("host:connect:%s:%d", ip, port[0])); err != nil {
		return err
	}
	if !strings.HasPrefix(string(raw), "connected to") && !strings.HasPrefix(string(raw), "already connected to") {
		return fmt.Errorf("adb connect: %s", raw)
	}
	return
}

func (c Client) Disconnect(ip string, port ...int) (err error) {
	cmd := fmt.Sprintf("host:disconnect:%s", ip)
	if len(port) != 0 {
		cmd = fmt.Sprintf("host:disconnect:%s:%d", ip, port[0])
	}

	var raw []byte
	if raw, err = c.executeCommand(cmd); err != nil {
		return err
	}

	if !strings.HasPrefix(string(raw), "disconnected") {
		return fmt.Errorf("adb disconnect: %s", raw)
	}
	return
}

func (c Client) DisconnectAll() (err error) {
	var raw []byte
	if raw, err = c.executeCommand("host:disconnect:"); err != nil {
		return err
	}

	if !strings.HasPrefix(string(raw), "disconnected everything") {
		return fmt.Errorf("adb disconnect all: %s", raw)
	}
	return
}

func (c Client) KillServer() (err error) {
	var transport Transport
	if transport, err = newTransport(fmt.Sprintf("%s:%d", c.host, c.port)); err != nil {
		return err
	}
	defer func() { _ = transport.Close() }()

	if err = transport.Send("host:kill"); err != nil {
		return err
	}

	return
}
