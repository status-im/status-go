package gopls

import "io"

// CombinedReadWriteCloser combines stdin and stdout into one interface.
type CombinedReadWriteCloser struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

// Write writes data to stdin.
func (c *CombinedReadWriteCloser) Write(p []byte) (n int, err error) {
	return c.stdin.Write(p)
}

// Read reads data from stdout.
func (c *CombinedReadWriteCloser) Read(p []byte) (n int, err error) {
	return c.stdout.Read(p)
}

// Close closes both stdin and stdout.
func (c *CombinedReadWriteCloser) Close() error {
	err1 := c.stdin.Close()
	err2 := c.stdout.Close()
	if err1 != nil {
		return err1
	}
	return err2
}
