package gadb

import (
	"testing"
)

func TestClient_ServerVersion(t *testing.T) {
	adbClient, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	adbServerVersion, err := adbClient.ServerVersion()
	if err != nil {
		t.Fatal(err)
	}

	t.Log(adbServerVersion)
}

func TestClient_DeviceSerialList(t *testing.T) {
	adbClient, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	serials, err := adbClient.DeviceSerialList()
	if err != nil {
		t.Fatal(err)
	}

	for i := range serials {
		t.Log(serials[i])
	}
}

func TestClient_DeviceList(t *testing.T) {
	adbClient, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	devices, err := adbClient.DeviceList()
	if err != nil {
		t.Fatal(err)
	}

	for i := range devices {
		t.Log(devices[i].serial, devices[i].DeviceInfo())
	}
}

func TestClient_KillServer(t *testing.T) {
	adbClient, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	err = adbClient.KillServer()
	if err != nil {
		t.Fatal(err)
	}
}
