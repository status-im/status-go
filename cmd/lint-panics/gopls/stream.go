package gopls

import "io"

// IOStream combines stdin and stdout into one interface.
type IOStream struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

// Write writes data to stdin.
func (c *IOStream) Write(p []byte) (n int, err error) {
	return c.stdin.Write(p)
}

// Read reads data from stdout.
func (c *IOStream) Read(p []byte) (n int, err error) {
	return c.stdout.Read(p)
}

// Close closes both stdin and stdout.
func (c *IOStream) Close() error {
	err1 := c.stdin.Close()
	err2 := c.stdout.Close()
	if err1 != nil {
		return err1
	}
	return err2
}
