package adbtools

import (
	"fmt"
	"strings"
	"testing"
)

type testData struct {
	t      *testing.T
	device Device
}

func TestMethods(t *testing.T) {
	devices, err := Devices()
	if err != nil {
		t.Errorf("Failed to get device list: %v", err)
		return
	}
	test := testData{
		t:      t,
		device: devices[0],
	}

	err = test.testScreenSize()
	if err != nil {
		t.Error(err)
		return
	}

	err = test.testDumpPath()
	if err != nil {
		t.Error(err)
		return
	}

	err = test.testStartApp()
	if err != nil {
		t.Error(err)
		return
	}

}

func (t *testData) testScreenSize() error {
	t.t.Log("testing ScreenSize")
	t.device.ScreenSize()
	if t.device.Screen.Height == 0 || t.device.Screen.Width == 0 {
		return fmt.Errorf("Failed to get device screen size: %v", t.device.Screen)
	}
	t.t.Log("ScreenSize test passed")
	return nil
}

func (t *testData) testDumpPath() error {
	t.t.Log("testing DumpPath")
	t.device.XMLScreen(true)
	if cleanString(t.device.Shell(fmt.Sprintf("adb shell ls %s", t.device.dumpPath))) != t.device.dumpPath {
		return fmt.Errorf("Failed to fetch window_dump.xml")
	}
	t.t.Log("DumpPath test passed")
	return nil
}

func (t *testData) testStartApp() error {
	t.t.Log("testing StartApp; using firefox as test app")
	err := t.device.StartApp("org.mozilla.firefox", "org.mozilla.gecko.BrowserApp", "")
	if err != nil {
		return err
	}
	if !strings.Contains(t.device.Foreground(), "firefox") {
		return fmt.Errorf("Failed to start org.mozilla.firefox app; not on foreground")
	}
	t.t.Log("StartApp test passed")
	return nil
}
