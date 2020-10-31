package adbtools

import (
	"fmt"
	"testing"
	"time"
)

var (
	chrome = app{
		pkg:      "com.android.chrome",
		activity: "com.google.android.apps.chrome.Main",
	}
)

const (
	emulator = "lite"
)

type testData struct {
	test   *testing.T
	device Device
}

type app struct {
	pkg      string
	activity string
}

func TestMethods(t *testing.T) {

	close, err := testStartAVD(emulator, t)
	if err != nil {
		t.Errorf("testStartAVD err: %v", err)
		close()
		return
	}
	defer close()

	active, err := isAVDRunning(emulator)
	if err != nil {
		t.Error(err)
		return
	}
	if !active {
		t.Errorf("%s emulator is not active", emulator)
		return
	}

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

	err = test.testNodeList()
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
	t.test.Log("testing StartApp; using chrome as test app")
	t.device.CloseApp(chrome.pkg)
	err := t.device.StartApp(chrome.pkg, chrome.activity, "")
	if err != nil {
		return err
	}

	if !t.device.WaitApp(chrome.pkg, 1000, 5) {
		return fmt.Errorf("failed to start %s", chrome.pkg)
	}

	t.test.Log("StartApp test passed")
	return nil
}

func (t *testData) testWaitInScreen() error {
	t.test.Log("testing WaitInScreen; using chrome as test app")
	t.device.WakeUp()
	t.device.Swipe([4]int{int(t.device.Screen.Width / 2), t.device.Screen.Height - 100, int(t.device.Screen.Width / 2), 100})
	if err := t.device.WaitInScreen(1, "Search or type web address"); err != nil {
		return err
	}
	t.test.Log("WaitInScreen test passed")
	return nil
}

func (t *testData) testNodeList() error {
	t.test.Log("testing NodeList; using chrome as test app")
	nodes := t.device.NodeList(true)
	if len(nodes) == 0 {
		return fmt.Errorf("Failed to fetch xml tree and separate the nodes")
	}
	t.test.Logf("XML tree has %d nodes", len(nodes))
	t.test.Log("NodeList test passed")
	return nil
}

func testStartAVD(deviceName string, t *testing.T) (func(), error) {
	d1, err := Devices(false)
	if err != nil {
		t.Logf("d1: %v", err)
	}

	close, err := StartAVD(true, deviceName)
	if err != nil {
		close()
		return func() {}, err
	}

	t.Log("5s nap time")
	time.Sleep(5 * time.Second)

	d2, err := Devices(false)
	if err != nil {
		t.Logf("d2: %v", err)
	}
	if len(d1) == len(d2) {
		t.Logf("Failed to start the %s emulator; devices %#v", deviceName, d2)
	}
	t.Log("successfully tested starting avd;")
	t.Log("closing avd will be tested uppon defer")
	return func() {
		t.Logf("stopping %s emulator", deviceName)
		close()
		t.Log("5s nap time")
		time.Sleep(5 * time.Second)

		d3, err := Devices(false)
		if err != nil {
			t.Logf("d3: %v", err)
		}
		if len(d1) != len(d3) {
			t.Fatalf("Failed to stop the '%s' emulator", deviceName)
			return
		}
		t.Logf("successfully closed '%s' emulator", deviceName)
	}, nil
}
