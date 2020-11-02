package adbtools

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	shell "github.com/ozzono/go-shell"
	"github.com/ozzono/normalize"
)

// Device main structure
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
		return fmt.Sprintf("shell.Cmd err: %v", err)
	}
	return out
}

// Foreground verifies if the given package is on foreground
func (device *Device) Foreground() string {
	if device.Log {
		log.Println("screening after foreground app")
	}
	// TODO: futurally add string normalization
	return strings.ToLower(device.Shell("adb shell dumpsys window windows|grep Focus"))
}

// TapScreen taps the given coords and waits the given delay in Milliseconds
func (device *Device) TapScreen(x, y, delay int) {
	if device.Log {
		log.Printf("tapping [%d,%d]", x, y)
	}
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
func (device *Device) XMLScreen(newdump bool) (string, error) {
	if device.Log {
		log.Println("dumping screen xml")
	}
	if len(device.dumpPath) == 0 {
		device.dumpPath = "/sdcard/window_dump.xml"
		if device.Log {
			log.Println("setting default dump path")
		}
	}
	if newdump {
		output := device.Shell("adb shell uiautomator dump")
		if !strings.Contains(output, "xml") {
			return "", fmt.Errorf("failed to dump xml screen; output: %v", output)
		}
		dumpPath := cleanString(strings.TrimPrefix(output, "UI hierchary dumped to: "))
		if device.dumpPath != dumpPath {
			device.dumpPath = dumpPath
			if device.Log {
				log.Printf("resetting default dump path to '%s'", device.dumpPath)
			}
		}
	}
	return device.Shell(fmt.Sprintf("adb shell cat %s", device.dumpPath)), nil
}

// TapCleanInput tap and cleans the input
func (device *Device) TapCleanInput(x, y, charcount int) {
	if device.Log {
		log.Printf("tapping [%d,%d] and cleaning input field", x, y)
	}
	charcount = charcount/2 + 1
	device.TapScreen(x, y, 0)
	device.Shell("adb shell input keyevent KEYCODE_MOVE_END")
	for i := 0; i < charcount; i++ {
		device.Shell(`adb shell input keyevent --longpress $(printf 'KEYCODE_DEL %.0s' {1..2})`)
	}
}

// Swipe swipes the screen with [x1,y1,x2,y2] coords format
func (device *Device) Swipe(coords [4]int) {
	if device.Log {
		log.Printf("swiping from [%d,%d] to [%d,%d]", coords[0], coords[1], coords[2], coords[3])
	}
	device.Shell(fmt.Sprintf("adb shell input swipe %d %d %d %d", coords[0], coords[1], coords[2], coords[3]))
}

// CloseApp closes the app
func (device *Device) CloseApp(app string) {
	if device.Log {
		log.Printf("closing %s", app)
	}
	device.Shell(fmt.Sprintf("adb shell am force-stop %s", app))
}

// ClearApp clears all the app data
func (device *Device) ClearApp(app string) error {
	if device.Log {
		log.Printf("clearing %s", app)
	}
	output := device.Shell(fmt.Sprintf("adb shell pm clear %s", app))
	if strings.Contains(output, "Success") {
		return nil
	}
	return fmt.Errorf("Failed to clear %s app data. Output: %s", app, output)
}

//InputText inserts a given text in a selected input
func (device *Device) InputText(text string, splitted bool) error {
	if device.Log {
		log.Printf("inputing text %s", text)
	}
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
	device.Shell("adb shell input text " + text)
	return nil
}

// PageDown scrolls down a fixed amount of pixels
func (device *Device) PageDown() {
	// code 93 is equivalent to "KEYCODE_PAGE_DOWN"
	device.Shell("adb shell input keyevent 93")
}

// PageUp scrolls up a fixed amount of pixels
func (device *Device) PageUp() {
	// code 92 is equivalent to "KEYCODE_PAGE_UP"
	device.Shell("adb shell input keyevent 92")
}

// Devices returns all the connected devices´ ID
func Devices(Log bool) ([]Device, error) {
	output := []Device{}
	count := 0
	out, err := shell.Cmd("adb devices")
	if err != nil {
		return nil, fmt.Errorf("shell.Cmd err: %v", err)
	}
	for _, row := range strings.Split(out, "\n") {
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
//
// Before starting it only checks if Anbox is installed
//
// Alert: Does not check if its dependencies are installed
//
// To install dependencies check the link:
// https://docs.anbox.io/userguide/install_kernel_modules.html
//
// To install Anbox check the link:
// https://docs.anbox.io/userguide/install.html
//
// Alert: no method was found to stop the anbox emulator
func StartAnbox() error {
	whereis, err := shell.Cmd("whereis anbox")
	if err != nil {
		return fmt.Errorf("shell.Cmd err: %v", err)
	}
	if len(cleanString(whereis)) == 0 {
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
//
// This method requires the Android Studio to be installed
// and allows the calling to be verified or not
//
// The returned function closes the emulator
//
// ALERT: Requires the emulator to be closed, elsewise the emulator remains active
func StartAVD(verified bool, deviceName string) (func(), error) {
	active, err := isAVDRunning(deviceName)
	if err != nil {
		return func() {}, err
	}
	if active {
		log.Printf("%s emulator already running", deviceName)
		log.Println("returned function won't close the emulator")
		return func() {}, nil
	}
	emulatorPath := fmt.Sprintf("%v/Android/Sdk/emulator/emulator", os.Getenv("HOME"))
	if err := checkEnv(emulatorPath, deviceName); err != nil {
		return func() {}, fmt.Errorf("checkEnv err: %v", err)
	}

	log.Printf("Booting '%s' emulator ", deviceName)
	pid, err := shell.LooseCmd(fmt.Sprintf("%s -avd %s", emulatorPath, deviceName))
	if err != nil {
		return func() {}, fmt.Errorf("LooseCmd err: %v", err)
	}

	log.Printf("successfully started avd '%s'; pid: %d", deviceName, pid)
	return func() {
		if active {
			log.Printf("'%s' emulator was alive before, will remain alive after", deviceName)
			return
		}
		proc, err := os.FindProcess(pid)
		if err != nil {
			log.Printf("os.FindProcess err: %v", err)
			return
		}
		err = proc.Kill()
		if err != nil {
			log.Printf("proc.Kill err: %v", err)
			return
		}
		log.Printf("Closing '%s' emulator ", deviceName)
	}, nil
}

// DeviceReady returns the readiness state of the device
func (device *Device) DeviceReady() bool {
	return cleanString(device.Shell("adb shell getprop sys.boot_completed")) == "1"
}

// WaitDeviceReady waits until the device.
// It's specially useful after a fresh boot.
func (device *Device) WaitDeviceReady(attemptCount int) error {
	attempts := attemptCount
	if device.DefaultSleep == 0 {
		if device.Log {
			log.Println("setting default sleep to 100ms")
		}
		device.DefaultSleep = 100
	}
	for !device.DeviceReady() && attempts > 0 {
		if attempts == 0 {
			return fmt.Errorf("reached max retry attempts of %d", attemptCount)
		}
		if device.Log {
			log.Printf("waiting app load; %d attempts left", attempts)
		}
		device.sleep(10)
		if device.DeviceReady() {
			break
		}
		attempts--
	}
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
	if device.Log {
		log.Printf("is %s installed?", pkg)
	}
	for _, item := range strings.Split(device.Shell("adb shell pm list packages "+pkg), "\n") {
		if len(strings.Replace(item, " ", "", -1)) > 0 && strings.Contains(item, pkg) {
			if device.Log {
				log.Printf("'%s' found", item)
			}
			return true
		}
	}
	if device.Log {
		log.Printf("'%s' not found", pkg)
	}
	return false
}

// ScreenRecord records the screen as video with limited duration.
// Uses mp4 format
func (device *Device) ScreenRecord(filename string, duration int) {
	device.Shell(fmt.Sprintf("adb shell screenrecord -time-limit %d /sdcard/%s", duration, filename))
}

// ScreenCap captures the screen as png
func (device *Device) ScreenCap(filename string) {
	device.Shell("adb shell screencap /sdcard/" + filename)
}

// Root enables all adb commands to be run as root.
// Only works in rooted devices or emulators
func (device *Device) Root() error {
	output := device.Shell("adb root")
	if len(strings.Split(output, "\n")) > 1 {
		return fmt.Errorf("Unable to restart adb as root; err: %v", output)
	}
	return nil
}

// XMLtoCoords converts XML block coords to center tap coords.
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
//
// 0: portrait
//
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

// WaitApp waits until the given app appears on the foreground.
//
// Waits for given miliseconds after each try.
//
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
	if device.Log {
		log.Println("waking the device up")
	}
	device.Shell("adb shell input keyevent KEYCODE_WAKEUP")
}

//ScreenSize fetches the physical screen size and return its height and width
func (device *Device) ScreenSize() error {
	if device.Log {
		log.Println("fetching screen dimensions")
	}
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
	if device.Log {
		log.Println("is screen on?")
	}
	return strings.Contains(device.Shell("adb shell dumpsys power | grep state"), "ON")
}

//HasInScreen verifies if the wanted text appear on screen
func (device *Device) HasInScreen(newDump bool, want ...string) bool {
	newWant := make([]string, len(want))
	j := copy(newWant, want)
	if j != len(want) {
		log.Printf("something went wrong on copying %v; copied items: %d", want, j)
		return false
	}
	if device.Log {
		log.Printf("has in screen: '%s'", strings.Join(newWant, "' or '"))
	}
	for len(newWant) > 0 {
		screen, err := device.XMLScreen(newDump)
		if err != nil {
			log.Printf("XMLScreen err: %v", err)
			return false
		}

		i := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(newWant))
		if device.Log {
			log.Printf("Searching screen %s", strings.ToLower(normalize.Norm(newWant[i])))
		}

		if strings.Contains(
			strings.ToLower(normalize.Norm(screen)),
			strings.ToLower(normalize.Norm(newWant[i])),
		) {
			return true
		}
		newWant = append(newWant[:i], newWant[i+1:]...)
	}
	return false
}

// WaitInScreen waits until the wanted text appear on screen.
// It requires a max retry count to avoid endless loop.
func (device *Device) WaitInScreen(attemptCount int, want ...string) error {
	if device.Log {
		log.Printf("wait in screen: %s", strings.Join(want, " or "))
	}
	attempts := attemptCount
	if device.DefaultSleep == 0 {
		return fmt.Errorf("Invalid device.DefaultSleep; must be > 0")
	}
	for !device.HasInScreen(true, want...) {
		attempts--
		if attempts == 0 {
			return fmt.Errorf("Reached max retry attempts of %d", attemptCount)
		}
		if device.Log {
			log.Printf("Waiting app load; %d attempts left", attempts)
		}
		device.sleep(10)
	}
	return nil
}

//
// ScreenTimeout sets the screen off timeout to key1 seconds
//
// Returns a function to be deferred setting timeout to key2 seconds
//
// Accepts only the following time intervals:
//
// 15s
//
// 30s
//
// 1m
//
// 2m
//
// 5m
//
// 10m
//
// 30m
func (device *Device) ScreenTimeout(waitTime string) (func(), error) {
	if device.Log {
		log.Printf("setting screen off timeout to %s", waitTime)
	}
	// set and validate waitTime
	timeout := map[string]string{
		"15s": "15000",
		"30s": "30000",
		"1m":  "60000",
		"2m":  "120000",
		"5m":  "300000",
		"10m": "600000",
		"30m": "1800000",
	}
	invalid := true
	for mapKey := range timeout {
		if waitTime == mapKey {
			invalid = false
			break
		}
	}
	if invalid {
		fmtTimeout := "wait time must be one of the following:\n"
		for key := range timeout {
			fmtTimeout += fmt.Sprintf("key: % 3s\n", key)
		}
		return func() {}, fmt.Errorf("invalid keys:\n%s", fmtTimeout)
	}

	current := cleanString(device.Shell("adb shell settings get system screen_off_timeout"))
	if timeout[waitTime] == current {
		return func() {}, nil
	}

	// sets screen off timeout
	if device.Log {
		log.Printf("Setting screen off timeout to %ss", strings.TrimSuffix(timeout[waitTime], "000"))
	}

	output := device.Shell(fmt.Sprintf("adb shell settings put system screen_off_timeout %s", timeout[waitTime]))
	if len(output) > 0 {
		return func() {}, fmt.Errorf("Failed to set screen_off_timeout: %s", output)
	}
	return func() {
		if device.Log {
			log.Printf("Setting screen off timeout back to %ss", current)
		}
		output := device.Shell(fmt.Sprintf("adb shell settings put system screen_off_timeout %s", current))
		if len(output) > 0 {
			log.Printf("Failed to set screen_off_timeout: %s", output)
			return
		}
	}, nil
}

// NodeList returns the unnested list of xml nodes
func (device *Device) NodeList(newDump bool) []string {
	if device.Log {
		log.Println("fetching node list")
	}
	nodes := []string{}
	screen, err := device.XMLScreen(newDump)
	if err != nil {
		log.Printf("XMLScreen err: %v", err)
		return []string{}
	}
	for _, item := range strings.Split(strings.Replace(screen, "><", ">\n<", -1), "\n") {
		if match("(\\[\\d+,\\d+\\]\\[\\d+,\\d+\\])", item) {
			nodes = append(nodes, item)
		}
	}
	return nodes
}

func isAVDRunning(name string) (bool, error) {

	psList := []string{}
	out, err := shell.Cmd("ps -ef")
	if err != nil {
		return false, fmt.Errorf("shell.Cmd err: %v", err)
	}
	for _, item := range strings.Split(out, "\n") {
		if strings.Contains(item, "emulator") &&
			strings.Contains(item, "-avd") &&
			strings.Contains(item, name) {
			psList = append(psList, item)
		}
	}
	for i := range psList {
		if strings.HasSuffix(psList[i], name) {
			return true, nil
		}
	}
	return false, nil
}

func cleanString(input string) string {
	input = strings.Replace(input, " ", "", -1)
	input = strings.Replace(input, "\n", "", -1)
	input = strings.Replace(input, "\r", "", -1)
	return input
}

func match(exp, text string) bool {
	return regexp.MustCompile(exp).MatchString(text)
}

func checkEnv(path, device string) error {
	log.Println("Verifying environment settings")
	out, err := shell.Cmd("which android-studio")
	if err != nil {
		return fmt.Errorf("shell.Cmd err: %v", err)
	}
	if !strings.Contains(out, "android-studio") {
		return fmt.Errorf("Cannot start AVD emulator; Android Studio is not installed; output: %s", out)
	}
	out, err = shell.Cmd("ls " + path)
	if err != nil {
		return fmt.Errorf("shell.Cmd err: %v; cmd: %s; output: %s", err, "ls "+path, out)
	}
	if !strings.Contains(out, path) {
		return fmt.Errorf("Cannot start AVD emulator; AVD manager not found")
	}
	list, err := shell.Cmd(fmt.Sprintf("%s -list-avds", path))
	if !strings.Contains(list, device) {
		return fmt.Errorf("Cannot start AVD emulator; %v device not found", device)
	}
	log.Println("Successfully verified environment settings")
	return nil
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
