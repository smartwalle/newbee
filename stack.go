package newbee

import (
	"bytes"
	"fmt"
)

type stackError struct {
	value interface{}
	stack []byte
}

func newStackError(v interface{}, stack []byte) *stackError {
	if line := bytes.IndexByte(stack[:], '\n'); line >= 0 {
		stack = stack[line+1:]
	}
	return &stackError{value: v, stack: stack}
}

func (this *stackError) Error() string {
	return fmt.Sprintf("%v\n%s", this.value, this.stack)
}
