package providers

import (
	"errors"
	"testing"
)

func TestParseCattScanOutputMapsNoDevicesToNotFound(t *testing.T) {
	_, err := parseCattScanOutput("Scanning Chromecasts...\nError: No devices found.\n", errors.New("exit status 1"))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestParseCattScanOutputMapsEmptySuccessToNotFound(t *testing.T) {
	_, err := parseCattScanOutput("Scanning Chromecasts...\n", nil)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestParseCattScanOutputReturnsDevices(t *testing.T) {
	devices, err := parseCattScanOutput("192.168.1.36 - Living Room TV\n", nil)
	if err != nil {
		t.Fatalf("parseCattScanOutput returned error: %v", err)
	}
	if len(devices) != 1 || devices[0].Name != "Living Room TV" {
		t.Fatalf("devices = %#v, want Living Room TV", devices)
	}
}
