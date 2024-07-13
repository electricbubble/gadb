# gadb
ADB Client in pure Golang.

## Installation
```shell script
go get github.com/electricbubble/gadb
```

## Example
```go
package main

import (
	"github.com/electricbubble/gadb"
	"log"
	"os"
	"strings"
)

func main() {
	adbClient, err := gadb.NewClient()
	checkErr(err, "fail to connect adb server")

	devices, err := adbClient.DeviceList()
	checkErr(err)

	if len(devices) == 0 {
		log.Fatalln("list of devices is empty")
	}

	dev := devices[0]

	userHomeDir, _ := os.UserHomeDir()
	apk, err := os.Open(userHomeDir + "/Desktop/xuexi_android_10002068.apk")
	checkErr(err)

	log.Println("starting to push apk")

	remotePath := "/data/local/tmp/xuexi_android_10002068.apk"
	err = dev.PushFile(apk, remotePath)
	checkErr(err, "adb push")

	log.Println("push completed")

	log.Println("starting to install apk")

	shellOutput, err := dev.RunShellCommand("pm install", remotePath)
	checkErr(err, "pm install")
	if !strings.Contains(shellOutput, "Success") {
		log.Fatalln("fail to install: ", shellOutput)
	}

	log.Println("install completed")

}

func checkErr(err error, msg ...string) {
	if err == nil {
		return
	}

	var output string
	if len(msg) != 0 {
		output = msg[0] + " "
	}
	output += err.Error()
	log.Fatalln(output)
}

```

## Thanks

Thank you [JetBrains](https://www.jetbrains.com/?from=gwda) for providing free open source licenses

---

Similar projects:

Repository|Description
---|---
[zach-klippenstein/goadb](https://github.com/zach-klippenstein/goadb)|A Golang library for interacting with adb.
[vidstige/jadb](https://github.com/vidstige/jadb)|ADB Client in pure Java.
[Swind/pure-python-adb](https://github.com/Swind/pure-python-adb)|This is pure-python implementation of the ADB client.
[codeskyblue/fa](https://github.com/codeskyblue/fa)|FA(fast adb) helps you win at ADB(Android Debug Bridge).
[mobile-dev-inc/dadb](https://github.com/mobile-dev-inc/dadb)|Connect directly to `adbd` without ADB binary or ADB server (Kotlin)
