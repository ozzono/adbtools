package adbtools

import (
	"fmt"
	"testing"
)

func TestMethods(t *testing.T) {
	devices, err := Devices()
	if err != nil {
		t.Errorf("Failed to get device list: %v", err)
		return
	}
	device := devices[0]
	err = device.testScreenSize()
	if err != nil {
		t.Error(err)
		return
	}
}

func (device *Device) testScreenSize() error {
	device.ScreenSize()
	if device.Screen.Height == 0 || device.Screen.Width == 0 {
		return fmt.Errorf("Failed to get device screen size: %v", device.Screen)
	}
	return nil
}
