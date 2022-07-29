package problem

import (
	"io"
)

type source interface {
	// ReadCloser returns a reader for the source code.
	ReadCloser() (io.ReadCloser, error)
}
