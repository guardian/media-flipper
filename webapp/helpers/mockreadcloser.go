package helpers

import "io"

type MockReadCloserState struct {
	WasClosed bool
	WasRead   bool
}

type MockReadCloser struct {
	State      *MockReadCloserState
	DataToRead []byte
}

func NewMockReadCloser() MockReadCloser {
	return MockReadCloser{
		State: &MockReadCloserState{},
	}
}

func (c MockReadCloser) Close() error {
	c.State.WasClosed = true
	return nil
}

func (c MockReadCloser) Read(p []byte) (n int, err error) {
	if c.State.WasRead {
		return 0, io.EOF
	}
	if c.State.WasClosed {
		return 0, io.ErrClosedPipe
	}
	c.State.WasRead = true
	copy(p, c.DataToRead)
	return len(c.DataToRead), nil
}
