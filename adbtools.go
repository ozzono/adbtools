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
	"github.com/ozzono/normalize"
)

var (
	deviceID string
)

// Device may structure
type Device struct {
	ID           string
	Log          bool
	dumpPath     string
	DefaultSleep int
	Screen       struct {
		Width  int
		Height int
	}
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
	if device.DefaultSleep == 0 {
		device.DefaultSleep = 100
	}
	out, err := shell.Cmd(arg)
	if err != nil {
		log.Printf("shell.Cmd err: %v", err)
	}
	return out
}

// Foreground verifies if the given package is on foreground
func (device *Device) Foreground() string {
	// TODO: futurally add string normalization
	return strings.ToLower(device.Shell("adb shell dumpsys window windows|grep Focus"))
}

// TapScreen taps the given coords and waits the given delay in Milliseconds
func (device *Device) TapScreen(x, y, delay int) {
	device.Shell(fmt.Sprintf("adb shell input tap %d %d", x, y))
	device.sleep(delay)
	return
}

func (device *Device) sleep(delay int) {
	if device.DefaultSleep == 0 {
		device.DefaultSleep = 100
	}
	time.Sleep(time.Duration(device.DefaultSleep*delay) * time.Millisecond)
}

// XMLScreen fetches the screen xml data
func (device *Device) XMLScreen(newdump bool) string {
	if len(device.dumpPath) == 0 {
		device.dumpPath = "/sdcard/window_dump.xml"
		if device.Log {
			log.Println("setting default dump path")
		}
	}
	if newdump {
		dumpPath := cleanString(strings.TrimPrefix(device.Shell("adb shell uiautomator dump"), "UI hierchary dumped to: "))
		if device.dumpPath != dumpPath {
			device.dumpPath = dumpPath
			if device.Log {
				log.Printf("resetting default dump path to '%s'", device.dumpPath)
			}
		}
	}
	return device.Shell(fmt.Sprintf("adb shell cat %s", device.dumpPath))
}

// TapCleanInput tap and cleans the input
func (device *Device) TapCleanInput(x, y, charcount int) {
	charcount = charcount/2 + 1
	device.TapScreen(x, y, 0)
	device.Shell("adb shell input keyevent KEYCODE_MOVE_END")
	for i := 0; i < charcount; i++ {
		device.Shell(`adb shell input keyevent --longpress $(printf 'KEYCODE_DEL %.0s' {1..2})`)
	}
}

// Swipe swipes the screen with [x1,y1,x2,y2] coords format
func (device *Device) Swipe(coords [4]int) {
	device.Shell(fmt.Sprintf("adb shell input swipe %d %d %d %d", coords[0], coords[1], coords[2], coords[3]))
}

// CloseApp closes the app
func (device *Device) CloseApp(app string) {
	device.Shell(fmt.Sprintf("adb shell am force-stop %s", app))
}

// ClearApp clears all the app data
func (device *Device) ClearApp(app string) error {
	output := device.Shell(fmt.Sprintf("adb shell pm clear %s", app))
	if strings.Contains(output, "Success") {
		return nil
	}
	return fmt.Errorf("Failed to clear %s app data. Output: %s", app, output)
}

//InputText inserts a given text in a selected input
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

// PageDown scrolls down a fixed amount of pixels
func (device *Device) PageDown() {
	// code 93 is equivalent to "KEYCODE_PAGE_DOWN"
	device.Shell("adb shell input keyevent 93")
}

// PageUp scrolls up a fixed amount of pixels
// Scrolls up a fixed amount of pixels
func (device *Device) PageUp() {
	// code 92 is equivalent to "KEYCODE_PAGE_UP"
	device.Shell("adb shell input keyevent 92")
}

// Devices returns all the connected devices´ ID
func Devices(Log bool) ([]Device, error) {
	output := []Device{}
	count := 0
	cmd, err := shell.Cmd("adb devices")
	if err != nil {
		return nil, fmt.Errorf("shell.Cmd err: %v", err)
	}
	for _, row := range strings.Split(cmd, "\n") {
		if strings.HasSuffix(row, "device") {
			output = append(output, Device{ID: strings.Split(row, "	")[0], Log: Log, DefaultSleep: 100})
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
func NewDevice(deviceID string, log bool) Device {
	return Device{ID: deviceID, Log: log}
}

// StartAnbox starts an Anbox Emulator
// Before starting it only checks if Anbox is installed
// Alert: Does not check if its dependencies are installed
// To install dependencies check the link below:
// https://docs.anbox.io/userguide/install_kernel_modules.html
// To install Anbox check the link below:
// https://docs.anbox.io/userguide/install.html
// Alert: no method was found to stop the anbox emulator
func StartAnbox() error {
	whereis, err := shell.Cmd("whereis anbox")
	if err != nil {
		return fmt.Errorf("shell.Cmd err: %v", err)
	}
	if len(whereis) == 0 {
		return fmt.Errorf("anbox package not found")
	}
	log.Println("Starting Anbox emulator")
	_, err = shell.Cmd("anbox launch --package=org.anbox.appmgr --component=org.anbox.appmgr.AppViewActivity")
	if err != nil {
		return fmt.Errorf("shell.Cmd err: %v", err)
	}
	return nil
}

// StartAVD starts the emulator with the given name
// This method requires the Android Studio to be installed
// ALERT: This method must be used as goroutine
func StartAVD(name string) error {
	cmd, err := shell.Cmd("which android-studio")
	if err != nil {
		return fmt.Errorf("shell.Cmd err: %v", err)
	}
	if len(strings.Split(cmd, "\n")) == 0 {
		return fmt.Errorf("Cannot start AVD emulator; Android Studio is not installed")
	}
	cmd, err = shell.Cmd("adb devices")
	if err != nil {
		return fmt.Errorf("shell.Cmd err: %v", err)
	}
	if strings.Contains(cmd, name) {
		return fmt.Errorf("Cannot start AVD emulator; %s is already running", name)
	}
	home := os.Getenv("HOME")
	cmd, err = shell.Cmd(fmt.Sprintf("ls %v/Android/Sdk/emulator/emulator", home))
	if err != nil {
		return fmt.Errorf("shell.Cmd err: %v", err)
	}
	if len(strings.Split(cmd, "\n")) == 0 {
		return fmt.Errorf("Cannot start AVD emulator; AVD manager not found")
	}
	list, err := shell.Cmd(fmt.Sprintf("%v/Android/Sdk/emulator/emulator -list-avds", home))
	if !(strings.Contains(list, name)) {
		return fmt.Errorf("Cannot start AVD emulator; %v device not found", name)
	}
	log.Printf("Booting avd: %v", name)
	shell.Cmd(home + "/Android/Sdk/emulator/emulator -avd " + name)
	return nil
}

// StartApp requires the package name with format com.packagename
// and activity such as com.packagename.MainActivity
func (device *Device) StartApp(pkg, activity, options string) error {
	if !device.InstalledApp(pkg) {
		return fmt.Errorf("Cannot start %s; Package not found", pkg)
	}
	output := device.Shell(fmt.Sprintf("adb shell am start -a -n %s/%s %s", pkg, activity, options))
	if strings.Contains(output, "Starting") {
		return nil
	}
	return fmt.Errorf("Failed to start %s: %s", pkg, output)
}

// InstalledApp checks if the given app package is installed
func (device *Device) InstalledApp(pkg string) bool {
	return len(strings.Split(device.Shell("adb shell pm list packages "+pkg), "\n")) > 0
}

// ScreenRecord records the screen as video with limited duration
// Uses mp4 format
func (device *Device) ScreenRecord(filename string, duration int) {
	device.Shell(fmt.Sprintf("adb shell screenrecord -time-limit %d /sdcard/%s", duration, filename))
}

// ScreenCap captures the screen as png
func (device *Device) ScreenCap(filename string) {
	device.Shell("adb shell screencap /sdcard/" + filename)
}

// Root enables all adb commands to be run as root
// Only works in rooted devices or emulators
func (device *Device) Root() error {
	output := device.Shell("adb root")
	if len(strings.Split(output, "\n")) > 1 {
		return fmt.Errorf("Unable to restart adb as root; err: %v", output)
	}
	return nil
}

// XMLtoCoords converts XML block coords to center tap coords
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

// Orientation returns the devices orientation
// 0: portrait
// 1: landscape
func (device *Device) Orientation() (int, error) {
	output := device.Shell("adb shell dumpsys input | grep 'SurfaceOrientation' | awk '{ print $2 }'")
	orientation, err := strconv.Atoi(output)
	if err != nil {
		return 0, fmt.Errorf("Failed to fetch device's orientation: %v", output)
	}
	return orientation, nil
}

//PowerButton emulates the pressing of the power button
func (device *Device) PowerButton() {
	// KEYCODE_POWER also works
	device.Shell("adb shell input keyevent 26")
}

// AutoRotate enables or disables the device auto rotation behaviour
func (device *Device) AutoRotate(rotate bool) {
	if rotate {
		device.Shell("adb shell content insert --uri content://settings/system --bind name:s:accelerometer_rotation --bind value:i:1")
	} else {
		device.Shell("adb shell content insert --uri content://settings/system --bind name:s:accelerometer_rotation --bind value:i:0")
	}
}

// Activities returns all package's activities
func (device *Device) Activities(packagename string) []string {
	list := strings.Split(device.Shell(fmt.Sprintf("adb shell dumpsys package | grep -i %s |grep Activity", packagename)), "\n")
	output := []string{}
	for i := range list {
		output = append(output, strings.TrimPrefix(list[i], "package:"))
	}
	return output
}

// DefaultBrowser loads the page in a default browser's new tab
func (device *Device) DefaultBrowser(url string) error {
	output := device.Shell(fmt.Sprintf("adb shell am start -a \"android.intent.action.VIEW\" -d \"%s\"", url))
	if strings.Contains(strings.ToLower(output), "error") {
		return fmt.Errorf("Failed to load page; output: \n%s", output)
	}
	return nil
}

// GetImei returns the device IMEI
func (device *Device) GetImei() string {
	return device.Shell("adb shell \"service call iphonesubinfo 1 | toybox cut -d \\\"'\\\" -f2 | toybox grep -Eo '[0-9]' | toybox xargs | toybox sed 's/\\ //g'\"")
}

// Shutdown turns the device off
func (device *Device) Shutdown() {
	device.Shell("adb shell reboot -p")
}

// WaitApp waits until the given app appears on the foreground
// Waits for given miliseconds after each try
// Note: Has limited retry count
func (device *Device) WaitApp(pkg string, delay, maxRetry int) bool {
	for !strings.Contains(device.Foreground(), pkg) {
		device.sleep(device.DefaultSleep * delay)

		if maxRetry == 0 {
			log.Println("Reached max retry count")
			log.Printf("%s package not found at the foreground", pkg)
			return false
		}
		maxRetry--

		if strings.Contains(device.Foreground(), pkg) {
			break
		}

		if device.Log {
			log.Printf("Waiting %s loading", pkg)
		}
	}
	return true
}

//WakeUp wakes the device up
func (device *Device) WakeUp() {
	device.Shell("adb shell input keyevent KEYCODE_WAKEUP")
}

//ScreenSize fetches the physical screen size and return its height and width
func (device *Device) ScreenSize() error {
	screen := device.Shell("adb shell wm size")
	if !regexp.MustCompile("Physical size: (\\d+x\\d+)").MatchString(screen) {
		return fmt.Errorf("Failed to fetch physical screen size; output: %s", screen)
	}
	sizes := strings.Split(strings.TrimPrefix(screen, "Physical size: "), "x")
	device.Screen.Width, _ = strconv.Atoi(cleanString(sizes[0]))
	device.Screen.Height, _ = strconv.Atoi(cleanString(sizes[1]))
	return nil
}

// IsScreenON verifies if the is on
func (device *Device) IsScreenON() bool {
	return strings.Contains(device.Shell("adb shell dumpsys power | grep state"), "ON")
}

//HasInScreen verifies if the wanted text appear on screen
func (device *Device) HasInScreen(newDump bool, want ...string) bool {
	for i := range want {
		if strings.Contains(
			strings.ToLower(normalize.Norm(device.XMLScreen(newDump))),
			strings.ToLower(normalize.Norm(want[i])),
		) {
			return true
		}
	}
	return false
}

// WaitInScreen waits until the wanted text appear on screen
// It requires a max retry count to avoid endless loop
func (device *Device) WaitInScreen(attemptCount int, want ...string) error {
	attempts := attemptCount
	if device.DefaultSleep == 0 {
		return fmt.Errorf("Invalid device.DefaultSleep; must be > 0")
	}
	for !device.HasInScreen(true, want...) {
		if attempts == 0 {
			return fmt.Errorf("Reached max retry attempts of %d", attemptCount)
		}
		if device.Log {
			log.Printf("Waiting app load; %d attempts left", attempts)
		}
		device.sleep(10)
		if device.HasInScreen(true, want...) {
			break
		}
		attempts--
	}
	return nil
}

func cleanString(input string) string {
	input = strings.Replace(input, " ", "", -1)
	input = strings.Replace(input, "\n", "", -1)
	input = strings.Replace(input, "\r", "", -1)
	return input
}

//TODO: this method requires revision
// func (device *Device) Portrait() error {
// 	orientation, err := device.Orientation()
// 	if err != nil {
// 		return fmt.Errorf("Failed to fetch the orientation: %v", err)
// 	}
// 	if orientation == 1 {
// 		device.AutoRotate(false)
// 		device.Shell("adb shell input keyevent 26")
// 	}
// 	return nil
// }

//TODO: this method requires revision
// func (device *Device) Landscape() error {
// 	orientation, err := device.Orientation()
// 	if err != nil {
// 		return fmt.Errorf("Failed to fetch the orientation: %v", err)
// 	}
// 	if orientation == 1 {
// 		device.AutoRotate(false)
// 		device.Shell("adb shell input keyevent 26")
// 	}
// 	return nil
// }
