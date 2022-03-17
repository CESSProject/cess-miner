package rpc

import "fmt"

// Error wraps RPC errors, which contain an error code in addition to the message.
type Error interface {
	Error() string    // returns the message
	ErrorCode() int32 // returns the code
}

const (
	defaultErrorCode = -1 - iota
	ParseErrorCode
	MethodNotFoundErrorCode
)

type parseError struct{ message string }

func (e *parseError) ErrorCode() int32 { return ParseErrorCode }

func (e *parseError) Error() string { return e.message }

type methodNotFoundError struct{ method string }

func (e *methodNotFoundError) ErrorCode() int { return MethodNotFoundErrorCode }

func (e *methodNotFoundError) Error() string {
	return fmt.Sprintf("the method %s does not exist/is not available", e.method)
}