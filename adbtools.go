package adbtools

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	shell "github.com/ozzono/go-shell"
)

var (
	deviceID     string
	loglvl       bool
	globalLogLvl bool
)

type Device struct {
	ID  string
	Log bool
}

// TODO: Validate the need of the given commands
// free memory verification and storage: adb shell cat /proc/meminfo |grep MemFree

// Shell executes the given command in the Linux bash terminal
// and return the command output as string
func (device *Device) Shell(arg string) string {
	if len(device.ID) > 0 {
		arg = strings.Replace(arg, "adb", fmt.Sprintf("adb -s %s", device.ID), -1)
	}
	if device.Log {
		log.Println(arg)
	}
	out, err := shell.Cmd(arg)
	if err != nil {
		log.Printf("shell.Cmd err: %v", err)
	}
	return out
}

// Verifies if the given package is on foreground
func (device *Device) Foreground() string {
	// TODO: futurally add string normalization
	return strings.ToLower(device.Shell("adb shell dumpsys window windows|grep Focus"))
}

// Taps the given coords and waits the given delay in Milliseconds
func (device *Device) TapScreen(x, y, delay int) {
	device.Shell(fmt.Sprintf("adb shell input tap %d %d", x, y))
	sleep(delay)
	return
}

func sleep(delay int) {
	time.Sleep(time.Duration(delay) * time.Millisecond)
}

// Fetches the screen xml data
func (device *Device) XMLScreen(newdump bool) string {
	if newdump {
		device.Shell("adb shell uiautomator dump")
	}
	return device.Shell("adb shell cat /sdcard/window_dump.xml")
}

// Tap and cleans the input
func (device *Device) TapCleanInput(x, y, charcount int) {
	charcount = charcount/2 + 1
	device.TapScreen(x, y, 0)
	device.Shell("adb shell input keyevent KEYCODE_MOVE_END")
	for i := 0; i < charcount; i++ {
		device.Shell(`adb shell input keyevent --longpress $(printf 'KEYCODE_DEL %.0s' {1..2})`)
	}
}

func (device *Device) Swipe(coords [4]int) {
	device.Shell(fmt.Sprintf("adb shell input swipe %d %d %d %d", coords[0], coords[1], coords[2], coords[3]))
}

func (device *Device) CloseApp(app string) {
	device.Shell(fmt.Sprintf("adb shell am force-stop %s", app))
}

// Clears all the app data
func (device *Device) ClearApp(app string) error {
	output := device.Shell(fmt.Sprintf("adb shell pm clear %s", app))
	if strings.Contains(output, "Success") {
		return nil
	}
	return fmt.Errorf("Failed to clear %s app data. Output: %s", app, output)
}

func (device *Device) InputText(text string, splitted bool) error {
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
func (device *Device) PageDown() {
	// code 93 is equivalent to "KEYCODE_PAGE_DOWN"
	device.Shell("adb shell input keyevent 93")
}

// Scroll up a fixed amount of pixels
func (device *Device) PageUp() {
	// code 92 is equivalent to "KEYCODE_PAGE_UP"
	device.Shell("adb shell input keyevent 92")
}

// Returns all the connected devices´ ID
func Devices() ([]Device, error) {
	output := []Device{}
	count := 0
	for _, row := range strings.Split(shell.Cmd("adb devices"), "\n") {
		if strings.HasSuffix(row, "device") {
			output = append(output, Device{ID: strings.Split(row, "	")[0], Log: false})
			count++
		} else if strings.HasSuffix(row, "offline") {
			id := strings.Split(row, "	")[0]
			log.Printf("%s device is offline", id)
		}
	}
	if count == 0 {
		return nil, fmt.Errorf("no devices found")
	}
	log.Printf("device count: %d\n", count)
	return output, nil
}

//NewDevice creates a new device management struct
func NewDevice(deviceID string) Device {
	return Device{ID: deviceID, Log: false}
}

// StartAVD starts the emulator with the given name
// This method requires the Android Studio to be installed
// ALERT: This method must be used as goroutine
func StartAVD(name string) error {
	if len(strings.Split(shell.Cmd("which android-studio"), "\n")) == 0 {
		return fmt.Errorf("Cannot start AVD emulator; Android Studio is not installed")
	}
	if strings.Contains(shell.Cmd("adb devices"), name) {
		return fmt.Errorf("Cannot start AVD emulator; %s is already running", name)
	}
	home := os.Getenv("HOME")
	if len(strings.Split(shell.Cmd(fmt.Sprintf("ls %v/Android/Sdk/emulator/emulator", home)), "\n")) == 0 {
		return fmt.Errorf("Cannot start AVD emulator; AVD manager not found")
	}
	list := shell.Cmd(fmt.Sprintf("%v/Android/Sdk/emulator/emulator -list-avds", home))
	if !(strings.Contains(list, name)) {
		return fmt.Errorf("Cannot start AVD emulator; %v device not found", name)
	}
	log.Printf("Booting avd: %v", name)
	shell.Cmd(home + "/Android/Sdk/emulator/emulator -avd " + name)
	return nil
}

// Requires the package name with format com.packagename
// and activitie such as com.packagename.MainActivity
func (device *Device) StartApp(pkg, activitie, options string) error {
	if !device.InstalledApp(pkg) {
		return fmt.Errorf("Cannot start %s; Package not found", pkg)
	}
	output := device.Shell(fmt.Sprintf("adb shell am start -a -n %s/%s %s", pkg, activitie, options))
	if output == "Success" {
		return nil
	}
	return fmt.Errorf("Failed to start %s: %s", pkg, output)
}

// Checks if the given app package is installed
func (device *Device) InstalledApp(pkg string) bool {
	return len(strings.Split(device.Shell("adb shell pm list packages "+pkg), "\n")) > 0
}

// Records the screen as video with limited duration
func (device *Device) ScreenRecord(filename string, duration int) {
	device.Shell(fmt.Sprintf("adb shell screenrecord -time-limit %d /sdcard/%s", duration, filename))
}

// Captures the screen as png
func (device *Device) ScreenCap(filename string) {
	device.Shell("adb shell screencap /sdcard/" + filename)
}

// Enables all adb commands to be run as root
func (device *Device) Root() error {
	output := device.Shell("adb root")
	if len(strings.Split(output, "\n")) > 1 {
		return fmt.Errorf("Unable to restart adb as root; err: %v", output)
	}
	return nil
}

// Coverts XML block coords to center tap coords
// Accepts [x1,y1][x2,y2] format as string and returns [2]int coords
func XMLtoCoords(xmlcoords string) ([2]int, error) {
	re := regexp.MustCompile("(\\[\\d+,\\d+\\]\\[\\d+,\\d+\\])")
	if !re.MatchString(xmlcoords) {
		return [2]int{0, 0}, fmt.Errorf("Unable to parse xmlcoords; Invalid format: %s", xmlcoords)
	}
	stringcoords := strings.Split(xmlcoords, "][")
	leftcoords := strings.Split(string(stringcoords[0][1:]), ",")
	rightcoords := strings.Split(string(stringcoords[1][:len(stringcoords[1])-1]), ",")
	x1, err := strconv.Atoi(leftcoords[0])
	if err != nil {
		return [2]int{0, 0}, fmt.Errorf("atoi err: %v", err)
	}
	y1, err := strconv.Atoi(leftcoords[1])
	if err != nil {
		return [2]int{0, 0}, fmt.Errorf("atoi err: %v", err)
	}
	x2, err := strconv.Atoi(rightcoords[0])
	if err != nil {
		return [2]int{0, 0}, fmt.Errorf("atoi err: %v", err)
	}
	y2, err := strconv.Atoi(rightcoords[1])
	if err != nil {
		return [2]int{0, 0}, fmt.Errorf("atoi err: %v", err)
	}
	x := (x1 + x2) / 2
	y := (y1 + y2) / 2
	return [2]int{x, y}, nil
}

func (device *Device) Orientation() (int, error) {
	output := device.Shell("adb shell dumpsys input | grep 'SurfaceOrientation' | awk '{ print $2 }'")
	orientation, err := strconv.Atoi(output)
	if err != nil {
		return 0, fmt.Errorf("Failed to fetch device's orientation: %v", output)
	}
	return orientation, nil
}

func (device *Device) Portrait() error {
	orientation, err := device.Orientation()
	if err != nil {
		return fmt.Errorf("Failed to fetch the orientation: %v", err)
	}
	if orientation == 1 {
		device.AutoRotate(false)
		device.Shell("adb shell input keyevent 26")
	}
	return nil
}

func (device *Device) Landscape() error {
	orientation, err := device.Orientation()
	if err != nil {
		return fmt.Errorf("Failed to fetch the orientation: %v", err)
	}
	if orientation == 1 {
		device.AutoRotate(false)
		device.Shell("adb shell input keyevent 26")
	}
	return nil
}

func (device *Device) PowerButton() {
	device.Shell("adb shell input keyevent 26")
}

func (device *Device) AutoRotate(rotate bool) {
	if rotate {
		device.Shell("adb shell content insert --uri content://settings/system --bind name:s:accelerometer_rotation --bind value:i:1")
	} else {
		device.Shell("adb shell content insert --uri content://settings/system --bind name:s:accelerometer_rotation --bind value:i:0")
	}
}

// Returns all package's activities
func (device *Device) Activities(packagename string) []string {
	list := strings.Split(device.Shell(fmt.Sprintf("adb shell dumpsys package | grep -i %s |grep Activity", packagename)), "\n")
	output := []string{}
	for i := range list {
		output = append(output, strings.TrimPrefix(list[i], "package:"))
	}
	return output
}

// Loads the page in a default browser's new tab
func (device *Device) DefaultBrowser(url string) error {
	output := device.Shell(fmt.Sprintf("adb shell am start -a \"android.intent.action.VIEW\" -d \"%s\"", url))
	if strings.Contains(strings.ToLower(output), "error") {
		return fmt.Errorf("Failed to load page; output: \n%s", output)
	}
	return nil
}

func GlobalLogLvl(lvl bool) {
	globalLogLvl = lvl
}

func (device *Device) GetImei() string {
	return device.Shell("adb shell \"service call iphonesubinfo 1 | toybox cut -d \\\"'\\\" -f2 | toybox grep -Eo '[0-9]' | toybox xargs | toybox sed 's/\\ //g'\"")
}

func (devive *Device) Shutdown() {
	devive.Shell("adb shell reboot -p")
}
