package push

import "fmt"

type PushError string

func (e PushError) TransportErrorCode() string {
	return string(e)
}

func (e PushError) Error() string {
	return e.TransportErrorCode()
}

var (
	ErrorTransportProblem   = fmt.Errorf("error [%w]", PushError("ServiceUnavailable"))
	ErrorRequest            = fmt.Errorf("error [%w]", PushError("InvalidRequest"))
	ErrorInvalidKey         = fmt.Errorf("error [%w]", PushError("InvalidKey"))
	ErrorServiceUnavailable = fmt.Errorf("error [%w]", PushError("ServiceUnavailable"))
	ErrorTokenRemoved       = fmt.Errorf("error [%w]", PushError("TokenRemoved.STATUS_URL.ACTION_REMOVE"))
	ErrorRateLimit          = fmt.Errorf("error [%w]", PushError("RateLimit"))
	ErrorRefreshToken       = fmt.Errorf("error [%w]", PushError("RefreshToken"))
	ErrorPerissionDenied    = fmt.Errorf("error [%w]", PushError("PermissionDenied"))
)
