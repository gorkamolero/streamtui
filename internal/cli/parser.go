package cli

import (
	"fmt"
	"strings"
)

type GlobalOptions struct {
	JSON   bool
	Device string
	Quiet  bool
	Config string
}

type ParsedCommand struct {
	Name    string         `json:"name"`
	Args    []string       `json:"args"`
	Options map[string]any `json:"options"`
}

type ParseResult struct {
	Globals GlobalOptions `json:"globals"`
	Command ParsedCommand `json:"command"`
}

func Parse(args []string) (ParseResult, ExitCode, error) {
	globals, rest, err := parseGlobals(args)
	if err != nil {
		return ParseResult{}, ExitInvalidArgs, err
	}
	if len(rest) == 0 {
		return ParseResult{}, ExitInvalidArgs, fmt.Errorf("command required")
	}

	command, exit, err := parseCommand(rest)
	if err != nil {
		return ParseResult{}, exit, err
	}

	return ParseResult{
		Globals: globals,
		Command: command,
	}, ExitSuccess, nil
}

func parseGlobals(args []string) (GlobalOptions, []string, error) {
	var globals GlobalOptions
	rest := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--json", "-j":
			globals.JSON = true
		case "--quiet", "-q":
			globals.Quiet = true
		case "--device", "-d":
			value, next, err := optionValue(args, i, arg)
			if err != nil {
				return globals, nil, err
			}
			globals.Device = value
			i = next
		case "--config", "-c":
			value, next, err := optionValue(args, i, arg)
			if err != nil {
				return globals, nil, err
			}
			globals.Config = value
			i = next
		default:
			rest = append(rest, args[i:]...)
			return globals, rest, nil
		}
	}

	return globals, rest, nil
}

func parseCommand(args []string) (ParsedCommand, ExitCode, error) {
	name := normalizeCommand(args[0])
	values := args[1:]

	switch name {
	case "search":
		return parseQueryOptions(name, values, optionSpec{
			flags: map[string]string{"--limit": "limit", "-l": "limit", "--type": "type", "-t": "type"},
		})
	case "trending":
		return parseOptionsWithExact(name, values, 0, optionSpec{
			flags: map[string]string{"--window": "window", "-w": "window", "--limit": "limit", "-l": "limit", "--type": "type", "-t": "type"},
		})
	case "info":
		return parseOptionsWithExact(name, values, 1, optionSpec{
			flags: map[string]string{"--type": "type", "-t": "type"},
		})
	case "streams":
		return parseOptionsWithExact(name, values, 1, optionSpec{
			flags: map[string]string{"--season": "season", "-s": "season", "--episode": "episode", "-e": "episode", "--quality": "quality", "-Q": "quality", "--limit": "limit", "-l": "limit", "--sort": "sort"},
		})
	case "subtitles":
		return parseOptionsWithExact(name, values, 1, optionSpec{
			flags: map[string]string{"--lang": "lang", "-l": "lang", "--season": "season", "-s": "season", "--episode": "episode", "-e": "episode", "--limit": "limit"},
			bools: map[string]string{"--trusted": "trusted", "--hearing-impaired": "hearing_impaired"},
		})
	case "devices":
		return parseDevices(values)
	case "cast":
		return parseOptionsWithExact(name, values, 1, optionSpec{
			flags: map[string]string{"--device": "device", "-d": "device", "--lang": "lang", "--subtitle-id": "subtitle_id", "--season": "season", "-s": "season", "--episode": "episode", "-e": "episode", "--quality": "quality", "-Q": "quality", "--index": "index", "-i": "index", "--subtitle-file": "subtitle_file", "--subtitle-delay": "subtitle_delay"},
			bools: map[string]string{"--vlc": "vlc", "--no-subtitle": "no_subtitle"},
		})
	case "cast-magnet":
		return parseOptionsWithExact(name, values, 1, optionSpec{
			flags: map[string]string{"--device": "device", "-d": "device", "--subtitle-file": "subtitle_file", "--subtitle-delay": "subtitle_delay", "--file-idx": "file_idx", "-i": "file_idx"},
			bools: map[string]string{"--vlc": "vlc"},
		})
	case "status", "pause", "stop", "seek", "volume":
		return parseControl(name, values)
	case "play":
		return parsePlay(values)
	default:
		return ParsedCommand{}, ExitInvalidArgs, fmt.Errorf("unknown command %q", args[0])
	}
}

func normalizeCommand(name string) string {
	switch name {
	case "s":
		return "search"
	case "tr":
		return "trending"
	case "i":
		return "info"
	case "st":
		return "streams"
	case "sub":
		return "subtitles"
	case "cm":
		return "cast-magnet"
	default:
		return name
	}
}

type optionSpec struct {
	flags map[string]string
	bools map[string]string
}

func parseRequiredArgs(name string, args []string, required int) (ParsedCommand, ExitCode, error) {
	if len(args) < required {
		return ParsedCommand{}, ExitInvalidArgs, fmt.Errorf("%s requires %d argument(s)", name, required)
	}
	return ParsedCommand{Name: name, Args: args, Options: map[string]any{}}, ExitSuccess, nil
}

func parseOptionsWithRequired(name string, args []string, required int, spec optionSpec) (ParsedCommand, ExitCode, error) {
	positionals, options, err := splitOptions(args, spec)
	if err != nil {
		return ParsedCommand{}, ExitInvalidArgs, err
	}
	if len(positionals) < required {
		return ParsedCommand{}, ExitInvalidArgs, fmt.Errorf("%s requires %d argument(s)", name, required)
	}
	return ParsedCommand{Name: name, Args: positionals, Options: options}, ExitSuccess, nil
}

func parseOptionsWithExact(name string, args []string, required int, spec optionSpec) (ParsedCommand, ExitCode, error) {
	command, exit, err := parseOptionsWithRequired(name, args, required, spec)
	if err != nil {
		return ParsedCommand{}, exit, err
	}
	if len(command.Args) != required {
		return ParsedCommand{}, ExitInvalidArgs, fmt.Errorf("%s requires exactly %d argument(s)", name, required)
	}
	return command, ExitSuccess, nil
}

func parseQueryOptions(name string, args []string, spec optionSpec) (ParsedCommand, ExitCode, error) {
	positionals, options, err := splitOptions(args, spec)
	if err != nil {
		return ParsedCommand{}, ExitInvalidArgs, err
	}
	if len(positionals) == 0 {
		return ParsedCommand{}, ExitInvalidArgs, fmt.Errorf("%s requires 1 argument(s)", name)
	}
	return ParsedCommand{Name: name, Args: []string{strings.Join(positionals, " ")}, Options: options}, ExitSuccess, nil
}

func parseOptions(name string, args []string, spec optionSpec) (ParsedCommand, ExitCode, error) {
	positionals, options, err := splitOptions(args, spec)
	if err != nil {
		return ParsedCommand{}, ExitInvalidArgs, err
	}
	return ParsedCommand{Name: name, Args: positionals, Options: options}, ExitSuccess, nil
}

func parseDevices(args []string) (ParsedCommand, ExitCode, error) {
	args, options, err := splitGlobalStyleFlags(args)
	if err != nil {
		return ParsedCommand{}, ExitInvalidArgs, err
	}
	if len(args) == 0 {
		return ParsedCommand{}, ExitInvalidArgs, fmt.Errorf("devices requires subcommand: list, refresh, or default")
	}

	switch args[0] {
	case "list", "refresh":
		if len(args) != 1 {
			return ParsedCommand{}, ExitInvalidArgs, fmt.Errorf("devices %s accepts no positional arguments", args[0])
		}
		return ParsedCommand{Name: "devices " + args[0], Args: nil, Options: options}, ExitSuccess, nil
	case "default":
		if len(args) < 2 {
			return ParsedCommand{}, ExitInvalidArgs, fmt.Errorf("devices default requires get or set")
		}
		switch args[1] {
		case "get":
			if len(args) != 2 {
				return ParsedCommand{}, ExitInvalidArgs, fmt.Errorf("devices default get accepts no positional arguments")
			}
			return ParsedCommand{Name: "devices default get", Args: nil, Options: options}, ExitSuccess, nil
		case "set":
			deviceName := strings.TrimSpace(strings.Join(args[2:], " "))
			if deviceName == "" {
				return ParsedCommand{}, ExitInvalidArgs, fmt.Errorf("devices default set requires a device name")
			}
			return ParsedCommand{Name: "devices default set", Args: []string{deviceName}, Options: options}, ExitSuccess, nil
		default:
			return ParsedCommand{}, ExitInvalidArgs, fmt.Errorf("unknown devices default subcommand %q", args[1])
		}
	default:
		return ParsedCommand{}, ExitInvalidArgs, fmt.Errorf("unknown devices subcommand %q", args[0])
	}
}

func parseControl(name string, args []string) (ParsedCommand, ExitCode, error) {
	args, options, err := splitGlobalStyleFlags(args)
	if err != nil {
		return ParsedCommand{}, ExitInvalidArgs, err
	}
	required := 0
	if name == "seek" || name == "volume" {
		required = 1
	}
	if len(args) < required {
		return ParsedCommand{}, ExitInvalidArgs, fmt.Errorf("%s requires %d argument(s)", name, required)
	}
	if len(args) != required {
		return ParsedCommand{}, ExitInvalidArgs, fmt.Errorf("%s requires exactly %d argument(s)", name, required)
	}
	return ParsedCommand{Name: name, Args: args, Options: options}, ExitSuccess, nil
}

func parsePlay(args []string) (ParsedCommand, ExitCode, error) {
	args, options, err := splitGlobalStyleFlags(args)
	if err != nil {
		return ParsedCommand{}, ExitInvalidArgs, err
	}
	if len(args) == 0 {
		return ParsedCommand{Name: "playback play", Args: nil, Options: options}, ExitSuccess, nil
	}
	command, exit, err := parseQueryOptions("play", args, optionSpec{
		flags: map[string]string{"--lang": "lang", "-l": "lang", "--device": "device", "-d": "device", "--subtitle-delay": "subtitle_delay"},
		bools: map[string]string{"--vlc": "vlc"},
	})
	if err != nil {
		return ParsedCommand{}, exit, err
	}
	for key, value := range options {
		command.Options[key] = value
	}
	return command, ExitSuccess, nil
}

func splitOptions(args []string, spec optionSpec) ([]string, map[string]any, error) {
	positionals := []string{}
	options := map[string]any{}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if key, ok := globalStyleBoolOption(arg); ok {
			options[key] = true
			continue
		}
		if key, ok := spec.bools[arg]; ok {
			options[key] = true
			continue
		}
		if key, ok := spec.flags[arg]; ok {
			value, next, err := optionValue(args, i, arg)
			if err != nil {
				return nil, nil, err
			}
			options[key] = value
			i = next
			continue
		}
		if strings.HasPrefix(arg, "-") {
			return nil, nil, fmt.Errorf("unknown option %q", arg)
		}
		positionals = append(positionals, arg)
	}

	return positionals, options, nil
}

func splitGlobalStyleFlags(args []string) ([]string, map[string]any, error) {
	filtered := make([]string, 0, len(args))
	options := map[string]any{}
	for _, arg := range args {
		if key, ok := globalStyleBoolOption(arg); ok {
			options[key] = true
			continue
		}
		filtered = append(filtered, arg)
	}
	return filtered, options, nil
}

func globalStyleBoolOption(arg string) (string, bool) {
	switch arg {
	case "--json", "-j":
		return "json", true
	case "--quiet", "-q":
		return "quiet", true
	default:
		return "", false
	}
}

func optionValue(args []string, index int, name string) (string, int, error) {
	if index+1 >= len(args) || strings.TrimSpace(args[index+1]) == "" {
		return "", index, fmt.Errorf("%s requires a value", name)
	}
	return args[index+1], index + 1, nil
}
