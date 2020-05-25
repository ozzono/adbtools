package adbtools

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

var (
	deviceID string
	loglvl   bool
)

type device struct {
	ID  string
	Log bool
}

// TODO: Validate the need of the given commands
// free memory verification and storage: adb shell cat /proc/meminfo |grep MemFree

// Shell executes the given command in the Linux bash terminal
// and return the command output as string
func (device *device) Shell(arg string) string {
	if len(device.ID) > 0 {
		arg = strings.Replace(arg, "adb", fmt.Sprintf("adb -s %s", device.ID), -1)
	}
	if device.Log {
		log.Println(arg)
	}
	return shell(arg)
}

func shell(arg string) string {
	args := strings.Split(arg, " ")
	out, err := exec.Command(args[0], args[1:]...).Output()
	if err != nil {
		log.Printf("Command: '%v'; Output: %v; Error: %v\n", arg, string(out), err)
		return err.Error()
	}
	if out != nil && len(out) > 0 {
		return fmt.Sprintf("Output:\n %s", out)
	}
	return string(out)
}

// Verifies if the given package is on foreground
func (device *device) Foreground(appPackage string) bool {
	// TODO: futurally add string normalization
	return strings.Contains(strings.ToLower(device.Shell("adb shell dumpsys window windows|grep Focus")), strings.ToLower(appPackage))
}

// Taps the given coords and waits the given delay in Milliseconds
func (device *device) TapScreen(x, y, delay int) {
	device.Shell(fmt.Sprintf("adb shell input tap %d %d", x, y))
	sleep(delay)
	return
}

func sleep(delay int) {
	time.Sleep(time.Duration(delay) * time.Millisecond)
}

// Fetches the screen xml data
func (device *device) XMLScreen(newdump bool) string {
	if newdump {
		device.Shell("adb shell uiautomator dump")
	}
	return device.Shell("adb shell cat /sdcard/window_dump.xml")
}

// Tap and cleans the input
func (device *device) TapCleanInput(x, y, charcount int) {
	charcount = charcount/2 + 1
	device.TapScreen(x, y, 0)
	device.Shell("adb shell input keyevent KEYCODE_MOVE_END")
	for i := 0; i < charcount; i++ {
		device.Shell(`adb shell input keyevent --longpress $(printf 'KEYCODE_DEL %.0s' {1..2})`)
	}
}

func (device *device) Swipe(coords [4]int) {
	device.Shell(fmt.Sprintf("adb shell input swipe %d %d %d %d", coords[0], coords[1], coords[2], coords[3]))
}

func (device *device) CloseApp(app string) {
	device.Shell(fmt.Sprintf("adb shell am force-stop %s", app))
}

// Clears all the app data
func (device *device) ClearApp(app string) error {
	output := device.Shell(fmt.Sprintf("adb shell pm clear %s", app))
	if strings.Contains(output, "Success") {
		return nil
	}
	return fmt.Errorf("Failed to clear %s app data. Output: %s", app, output)
}

func (device *device) InputText(text string, splitted bool) error {
	if len(text) == 0 {
		return fmt.Errorf("invalid input; cannot be empty")
	}
	// Fixes whitespace input with adb and shell
	text = strings.Replace(text, " ", "\\s", -1)
	if splitted {
		for i := range text {
			device.Shell(fmt.Sprintf("adb shell input text %v", text[i]))
		}
		return nil
	}
	device.Shell("adb shell input text %s" + text)
	return nil
}

// Scroll down a fixed amount of pixels
func (device *device) PageDown() {
	// code 93 is equivalent to "KEYCODE_PAGE_DOWN"
	device.Shell("adb shell input keyevent 93")
}

// Scroll up a fixed amount of pixels
func (device *device) PageUp() {
	// code 92 is equivalent to "KEYCODE_PAGE_UP"
	device.Shell("adb shell input keyevent 92")
}

// Returns all the connected devices´ ID
func Devices() ([]device, error) {
	output := []device{}
	count := 0
	for _, row := range strings.Split(shell("adb devices"), "\n") {
		if strings.HasSuffix(row, "device") {
			output = append(output, device{ID: strings.Split(row, "	")[0], Log: false})
			count++
		}
	}
	if count == 0 {
		return nil, fmt.Errorf("no devices found")
	}
	log.Printf("device count: %d\n", count)
	return output, nil
}

func NewDevice(deviceID string) device {
	return device{ID: deviceID, Log: false}
}

// StartAVD starts the emulator with the given name
// This method requires the Android Studio and Screen
// programs to be installed
func StartAVD(name string) error {
	if !(shell("command -v android-studio|wc -l") == "1") {
		return fmt.Errorf("Cannot start AVD emulator; Android Studio is not installed")
	}
	if !(shell("command -v screen|wc -l") == "1") {
		return fmt.Errorf("Cannot start AVD emulator; Screen is not installed")
	}
	if strings.Contains(shell("adb devices"), "name") {
		return fmt.Errorf("Cannot start AVD emulator; %s is already running", name)
	}
	list := shell("$HOME/Android/Sdx/emulator/emulator -list-avds")
	avdlist := strings.Split(list, "\n")
	if len(avdlist) == 0 {
		return fmt.Errorf("Cannot start AVD emulator; 0 devices found")
	}
	if !strings.Contains(list, name) {
		return fmt.Errorf("Cannot start AVD emulator; Device %s not found", name)
	}
	shell(fmt.Sprintf("screen -dmS avd_%s bash -c '$HOME/Android/Sdk/emulator/emulator -avd 480x800_android7.0'", name))
	return nil
}

// Requires the package name with format com.packagename
// and activitie such as com.packagename.MainActivity
func (device *device) StartApp(pkg, activitie string) error {
	if !device.InstalledApp(pkg) {
		return fmt.Errorf("Cannot start %s; Package not found", pkg)
	}
	device.Shell(fmt.Sprintf("adb shell am start -n %s/%s", pkg, activitie))
	return nil
}

// Checks if the given app package is installed
func (device *device) InstalledApp(pkg string) bool {
	return len(strings.Split(device.Shell("adb shell pm list packages "+pkg), "\n")) > 0
}

// Records the screen as video with limited duration
func (device *device) ScreenRecord(filename string, duration int) {
	device.Shell(fmt.Sprintf("adb shell screenrecord -time-limit %d /sdcard/%s", duration, filename))
}

// Captures the screen as png
func (device *device) ScreenCap(filename string) {
	device.Shell("adb shell screencap /sdcard/" + filename)
}

// Enables all adb commands to be run as root
func (device *device) Root() error {
	output := device.Shell("adb root")
	if len(strings.Split(output, "\n")) > 1 {
		return fmt.Errorf("Unable to restart adb as root; err: %v", output)
	}
	return nil
}
