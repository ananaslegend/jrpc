package jrpc

import (
	"fmt"

	"github.com/goccy/go-json"
)

var (
	errorParsingJSONString = []byte(`{"jsonrpc": "2.0", "error": {"code": -32700, "message": "Parse error"}, "id": null}`)
	errorInvalidRequest    = []byte(`{"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid Request"}, "id": null}`)
)

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func (e *Error) Error() string {
	str := fmt.Sprintf(`{"code": %d, "message": "%s"`, e.Code, e.Message)

	data := func() string {
		if e.Data != nil {
			jsonData, err := json.Marshal(e.Data)
			if err != nil {
				jsonData = []byte("error during marshaling error data: " + err.Error())
			}

			return fmt.Sprintf(`, "data": %s`, string(jsonData))
		}
		return ""
	}()

	str += data

	str += "}"

	return str
}

func ParseError(msg ...string) *Error {
	err := &Error{Code: -32700, Message: "Parse error"}

	if len(msg) != 0 {
		err.Message = msg[0]
	}

	return err
}

func InvalidRequestError(msg ...string) *Error {
	err := &Error{Code: -32600, Message: "Invalid Request"}

	if len(msg) != 0 {
		err.Message = msg[0]
	}

	return err
}

func MethodNotFoundError() *Error {
	return &Error{Code: -32601, Message: "Method not found"}
}

func InvalidParamsError(msg ...string) *Error {
	err := &Error{Code: -32602, Message: "Invalid params"}

	if len(msg) != 0 {
		err.Message = msg[0]
	}

	return err
}

func InternalError(msg ...string) *Error {
	err := &Error{Code: -32603, Message: "Internal error"}

	if len(msg) != 0 {
		err.Message = msg[0]
	}

	return err
}
