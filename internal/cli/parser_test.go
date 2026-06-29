package cli

import "testing"

func TestParseSearch(t *testing.T) {
	result, code, err := Parse([]string{"--json", "search", "the batman"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d", code, ExitSuccess)
	}
	if !result.Globals.JSON {
		t.Fatalf("JSON global was not parsed")
	}
	if result.Command.Name != "search" {
		t.Fatalf("command = %q, want search", result.Command.Name)
	}
	if got := result.Command.Args[0]; got != "the batman" {
		t.Fatalf("query = %q, want the batman", got)
	}
}

func TestParseDevicesDefaultSet(t *testing.T) {
	result, code, err := Parse([]string{"devices", "default", "set", "Living", "Room", "TV", "--json"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d", code, ExitSuccess)
	}
	if result.Command.Name != "devices default set" {
		t.Fatalf("command = %q, want devices default set", result.Command.Name)
	}
	if got := result.Command.Args[0]; got != "Living Room TV" {
		t.Fatalf("device = %q, want Living Room TV", got)
	}
	if got := result.Command.Options["json"]; got != true {
		t.Fatalf("json = %v, want true", got)
	}
}

func TestParseAgentFirstPlay(t *testing.T) {
	result, code, err := Parse([]string{"play", "the", "batman", "--lang", "es", "--device", "Living Room TV", "--json"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d", code, ExitSuccess)
	}
	if result.Command.Name != "play" {
		t.Fatalf("command = %q, want play", result.Command.Name)
	}
	if got := result.Command.Args[0]; got != "the batman" {
		t.Fatalf("query = %q, want the batman", got)
	}
	if got := result.Command.Options["lang"]; got != "es" {
		t.Fatalf("lang = %v, want es", got)
	}
	if got := result.Command.Options["device"]; got != "Living Room TV" {
		t.Fatalf("device = %v, want Living Room TV", got)
	}
	if _, ok := result.Command.Options["vlc"]; ok {
		t.Fatalf("vlc option should not be set")
	}
	if got := result.Command.Options["json"]; got != true {
		t.Fatalf("json = %v, want true", got)
	}
}

func TestParseAgentFirstPlayVLC(t *testing.T) {
	result, code, err := Parse([]string{"play", "the", "batman", "--lang", "es", "--vlc", "--json"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d", code, ExitSuccess)
	}
	if result.Command.Name != "play" {
		t.Fatalf("command = %q, want play", result.Command.Name)
	}
	if got := result.Command.Args[0]; got != "the batman" {
		t.Fatalf("query = %q, want the batman", got)
	}
	if got := result.Command.Options["vlc"]; got != true {
		t.Fatalf("vlc = %v, want true", got)
	}
	if got := result.Command.Options["json"]; got != true {
		t.Fatalf("json = %v, want true", got)
	}
}

func TestParseSearchJoinsUnquotedQuery(t *testing.T) {
	result, code, err := Parse([]string{"search", "the", "batman", "--json"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d", code, ExitSuccess)
	}
	if got := result.Command.Args[0]; got != "the batman" {
		t.Fatalf("query = %q, want the batman", got)
	}
}

func TestParseIDCommandRejectsExtraPositionals(t *testing.T) {
	tests := [][]string{
		{"streams", "tt1877830", "extra"},
		{"trending", "extra"},
		{"pause", "extra"},
	}

	for _, args := range tests {
		_, code, err := Parse(args)
		if err == nil {
			t.Fatalf("Parse(%v) returned nil error", args)
		}
		if code != ExitInvalidArgs {
			t.Fatalf("Parse(%v) code = %d, want %d", args, code, ExitInvalidArgs)
		}
	}
}

func TestParseTrailingGlobalStyleFlags(t *testing.T) {
	tests := [][]string{
		{"search", "the batman", "--json"},
		{"streams", "tt1877830", "--quiet"},
		{"devices", "refresh", "--json"},
		{"pause", "--json"},
	}

	for _, args := range tests {
		result, code, err := Parse(args)
		if err != nil {
			t.Fatalf("Parse(%v) returned error: %v", args, err)
		}
		if code != ExitSuccess {
			t.Fatalf("Parse(%v) code = %d, want %d", args, code, ExitSuccess)
		}
		if result.Command.Options["json"] != true && result.Command.Options["quiet"] != true {
			t.Fatalf("Parse(%v) options = %#v, want trailing flag", args, result.Command.Options)
		}
	}
}

func TestParsePlaybackPlay(t *testing.T) {
	result, code, err := Parse([]string{"play"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d", code, ExitSuccess)
	}
	if result.Command.Name != "playback play" {
		t.Fatalf("command = %q, want playback play", result.Command.Name)
	}
}

func TestParsePlaybackPlayWithTrailingJSON(t *testing.T) {
	result, code, err := Parse([]string{"play", "--json"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if code != ExitSuccess {
		t.Fatalf("code = %d, want %d", code, ExitSuccess)
	}
	if result.Command.Name != "playback play" {
		t.Fatalf("command = %q, want playback play", result.Command.Name)
	}
	if got := result.Command.Options["json"]; got != true {
		t.Fatalf("json = %v, want true", got)
	}
}

func TestParseInvalidCommand(t *testing.T) {
	_, code, err := Parse([]string{"unknown"})
	if err == nil {
		t.Fatalf("Parse returned nil error")
	}
	if code != ExitInvalidArgs {
		t.Fatalf("code = %d, want %d", code, ExitInvalidArgs)
	}
}
