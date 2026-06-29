package cli

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestSuccessEnvelopeShape(t *testing.T) {
	env := successEnvelope(map[string]string{"status": "ok"})

	if !env.OK {
		t.Fatalf("OK = false, want true")
	}
	if env.Error != nil {
		t.Fatalf("Error = %#v, want nil", env.Error)
	}
	if env.Meta.Version != "v1" {
		t.Fatalf("version = %q, want v1", env.Meta.Version)
	}

	var buf bytes.Buffer
	if err := writeJSON(&buf, env); err != nil {
		t.Fatalf("writeJSON returned error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}

	for _, key := range []string{"ok", "data", "error", "meta"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("missing key %q in %s", key, buf.String())
		}
	}
}

func TestErrorEnvelopeShape(t *testing.T) {
	env := errorEnvelope(ErrorDeviceNotFound, "No Chromecast device found")

	if env.OK {
		t.Fatalf("OK = true, want false")
	}
	if env.Data != nil {
		t.Fatalf("Data = %#v, want nil", env.Data)
	}
	if env.Error == nil {
		t.Fatalf("Error = nil, want detail")
	}
	if env.Error.Code != ErrorDeviceNotFound {
		t.Fatalf("code = %q, want %q", env.Error.Code, ErrorDeviceNotFound)
	}
	if env.Meta.Version != "v1" {
		t.Fatalf("version = %q, want v1", env.Meta.Version)
	}
}
