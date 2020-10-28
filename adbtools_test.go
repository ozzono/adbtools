package adbtools

import (
	"fmt"
	"testing"
)

var firefox app

type testData struct {
	test   *testing.T
	device Device
}

type app struct {
	pkg      string
	activity string
}

func TestMethods(t *testing.T) {
	devices, err := Devices(true)
	if err != nil {
		t.Errorf("Failed to get device list: %v", err)
		return
	}
	fmt.Printf("device: %#v\n", devices[0])
	test := testData{
		test:   t,
		device: devices[0],
	}

	firefox.pkg = "org.mozilla.firefox"
	firefox.activity = "org.mozilla.gecko.BrowserApp"

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

	err = test.testWaitInScreen()
	if err != nil {
		t.Error(err)
		return
	}

}

func (t *testData) testDeviceSettings() error {
	if t.device.DefaultSleep <= 0 {
		return fmt.Errorf("invalid device.DefaultSleep; must be > 0")
	}
	if len(t.device.dumpPath) == 0 {
		return fmt.Errorf("invalid device.dumpPath; cannot be empty")
	}
	if err := t.device.ScreenSize(); err != nil {
		return err
	}
	return nil
}

func (t *testData) testScreenSize() error {
	t.test.Log("testing ScreenSize")
	t.device.ScreenSize()
	if t.device.Screen.Height == 0 || t.device.Screen.Width == 0 {
		return fmt.Errorf("Failed to get device screen size: %v", t.device.Screen)
	}
	t.test.Log("ScreenSize test passed")
	return nil
}

func (t *testData) testDumpPath() error {
	t.test.Log("testing DumpPath")
	t.device.XMLScreen(true)
	if cleanString(t.device.Shell(fmt.Sprintf("adb shell ls %s", t.device.dumpPath))) != t.device.dumpPath {
		return fmt.Errorf("Failed to fetch window_dump.xml")
	}
	t.test.Log("DumpPath test passed")
	return nil
}

func (t *testData) testStartApp() error {
	t.test.Log("testing StartApp; using firefox as test app")
	t.device.CloseApp(firefox.pkg)
	err := t.device.StartApp(firefox.pkg, firefox.activity, "")
	if err != nil {
		return err
	}

	if !t.device.WaitApp(firefox.pkg, 1000, 5) {
		return fmt.Errorf("failed to start %s", firefox.pkg)
	}

	t.test.Log("StartApp test passed")
	return nil
}

func (t *testData) testWaitInScreen() error {
	t.test.Log("testing WaitInScreen; using firefox as test app")
	t.device.WakeUp()
	t.device.Swipe([4]int{int(t.device.Screen.Width / 2), t.device.Screen.Height - 100, int(t.device.Screen.Width / 2), 100})
	if err := t.device.WaitInScreen(1, "browser_toolbar"); err != nil {
		return err
	}
	t.test.Log("WaitInScreen test passed")
	return nil
}
