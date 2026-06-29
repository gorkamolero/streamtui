package domain

import "testing"

func TestParseCattStatus(t *testing.T) {
	status := ParseCattStatus("State: PLAYING\nTitle: The Batman\nDuration: 100.0\nCurrent time: 25.0\nVolume: 80\n", "Living Room TV")
	if status.State != PlaybackPlaying {
		t.Fatalf("state = %q, want playing", status.State)
	}
	if status.Title != "The Batman" || status.Device != "Living Room TV" {
		t.Fatalf("status = %#v", status)
	}
	if status.Position == nil || *status.Position != 25 {
		t.Fatalf("position = %v, want 25", status.Position)
	}
	if status.Progress == nil || *status.Progress != 0.25 {
		t.Fatalf("progress = %v, want 0.25", status.Progress)
	}
}

func TestParseSeek(t *testing.T) {
	tests := map[string]SeekCommand{
		"90":      {Action: "seek", Value: 90, Raw: "90"},
		"1:30":    {Action: "seek", Value: 90, Raw: "1:30"},
		"1:00:00": {Action: "seek", Value: 3600, Raw: "1:00:00"},
		"+30":     {Action: "ffwd", Value: 30, Raw: "+30"},
		"-10":     {Action: "rewind", Value: 10, Raw: "-10"},
	}
	for input, want := range tests {
		got, err := ParseSeek(input)
		if err != nil {
			t.Fatalf("ParseSeek(%q) returned error: %v", input, err)
		}
		if got != want {
			t.Fatalf("ParseSeek(%q) = %#v, want %#v", input, got, want)
		}
	}
}

func TestParseVolume(t *testing.T) {
	absolute, err := ParseVolume("150")
	if err != nil {
		t.Fatalf("ParseVolume returned error: %v", err)
	}
	if absolute.Action != "volume" || absolute.Value != 100 {
		t.Fatalf("absolute = %#v, want capped volume 100", absolute)
	}
	up, err := ParseVolume("+10")
	if err != nil {
		t.Fatalf("ParseVolume returned error: %v", err)
	}
	if up.Action != "volumeup" {
		t.Fatalf("up = %#v, want volumeup", up)
	}
}
