package gadb

import (
	"testing"
)

func Test_Transport(t *testing.T) {
	transport, err := newTransport("localhost:5037")
	if err != nil {
		t.Fatal(err)
	}

	err = transport.Send("host:version")
	if err != nil {
		t.Fatal(err)
	}

	raw, err := transport.Recv()
	if err != nil {
		t.Fatal(err)
	}
	_ = transport.Close()

	t.Log("hex:", string(raw))

	// transport, err = newTransport("localhost:5037")
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// err = transport.Send("host:devices")
	// if err != nil {
	// 	t.Fatal(err)
	// }
	//
	// raw, err = transport.Recv()
	// if err != nil {
	// 	t.Fatal(err)
	// }
	//
	// t.Log("\n" + string(raw))

	// transport, err = newTransport("localhost:5037")
	// if err != nil {
	// 	t.Fatal(err)
	// }
	//
	// err = transport.Send("host-serial:emulator-5554:get-state")
	// // err = transport.Send("host-serial:emulator-5554:get-serialno")
	// // err = transport.Send("host-serial:emulator-5554:get-devpath")
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// raw, err = transport.Recv()
	// if err != nil {
	// 	t.Fatal(err)
	// }
	//
	// t.Log("\n" + string(raw))

	// SetDebug(true)
	//
	// transport, err = newTransport("localhost:5037")
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// err = transport.Send("host:transport:21beda11")
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// status, err := transport.ReadStatus()
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// if status != StatusSuccess {
	// 	t.Fatal("should be:", StatusSuccess)
	// }
	// t.Log(status)
	//
	// err = transport.Send("shell:screencap -p")
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// raw, err = transport.ReadAll()
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// _ = transport.Close()
	// t.Log("\n" + string(raw))

	SetDebug(true)
	transport, err = newTransport("localhost:5037")
	if err != nil {
		t.Fatal(err)
	}
	err = transport.Send("host:connect:localhost:6790")
	if err != nil {
		t.Fatal(err)
	}

	raw, err = transport.Recv()
	if err != nil {
		t.Fatal(err)
	}

	t.Log("\n" + string(raw))

}
