package cli

type ExitCode int

const (
	ExitSuccess ExitCode = iota
	ExitError
	ExitInvalidArgs
	ExitNetwork
	ExitDeviceNotFound
	ExitNoResults
	ExitPlaybackFailed
)

type ErrorCode string

const (
	ErrorInvalidArgs    ErrorCode = "INVALID_ARGS"
	ErrorNetwork        ErrorCode = "NETWORK_ERROR"
	ErrorDeviceNotFound ErrorCode = "DEVICE_NOT_FOUND"
	ErrorNoResults      ErrorCode = "NO_RESULTS"
	ErrorPlaybackFailed ErrorCode = "PLAYBACK_FAILED"
	ErrorUnsupported    ErrorCode = "UNSUPPORTED_COMMAND"
	ErrorInternal       ErrorCode = "INTERNAL_ERROR"
)

func codeForExit(exit ExitCode) ErrorCode {
	switch exit {
	case ExitInvalidArgs:
		return ErrorInvalidArgs
	case ExitNetwork:
		return ErrorNetwork
	case ExitDeviceNotFound:
		return ErrorDeviceNotFound
	case ExitNoResults:
		return ErrorNoResults
	case ExitPlaybackFailed:
		return ErrorPlaybackFailed
	default:
		return ErrorInternal
	}
}
