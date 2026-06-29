package providers

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestBuildCattArgsWithDevice(t *testing.T) {
	got := BuildCattArgs("Living Room TV", "status")
	want := []string{"-d", "Living Room TV", "status"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("args = %#v, want %#v", got, want)
	}
}

func TestCattControlAdapterMapsMissingDevice(t *testing.T) {
	dir := t.TempDir()
	script := "#!/bin/sh\nprintf 'Error: Specified device \"Living Room TV\" not found.\\n'\nexit 1\n"
	if err := os.WriteFile(filepath.Join(dir, "catt"), []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	err := CattControlAdapter{}.Play("Living Room TV")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Play error = %v, want ErrNotFound", err)
	}
}

func TestBuildCattArgsWithoutDevice(t *testing.T) {
	got := BuildCattArgs("", "play")
	want := []string{"play"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("args = %#v, want %#v", got, want)
	}
}

func TestCattControlAdapterUsesFakeBinary(t *testing.T) {
	dir := t.TempDir()
	argsPath := filepath.Join(dir, "catt-args.txt")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + shellQuote(argsPath) + "\nprintf 'State: PLAYING\\nTitle: Demo\\nCurrent time: 12\\nDuration: 120\\nVolume: 30\\n'\n"
	if err := os.WriteFile(filepath.Join(dir, "catt"), []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	status, err := CattControlAdapter{}.Status("Living Room TV")
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if status.State != "playing" || status.Title != "Demo" {
		t.Fatalf("status = %#v, want playing Demo", status)
	}
	args := waitForFile(t, argsPath)
	want := "-d\nLiving Room TV\nstatus\n"
	if args != want {
		t.Fatalf("args = %q, want %q", args, want)
	}
}
