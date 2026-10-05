package contract

import "encoding/json"

const Version = 1
const MaxMessageBytes = 1 << 20

type Request struct {
	ID     string          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	ID     string `json:"id"`
	Result any    `json:"result,omitempty"`
	Error  *Error `json:"error,omitempty"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Event struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

type Ready struct {
	Protocol int `json:"protocol"`
}

type Status struct {
	Version   string `json:"version"`
	Execution string `json:"execution"`
}
