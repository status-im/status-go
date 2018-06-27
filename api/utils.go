package api

// RunAsync runs the specified function asynchronously.
func RunAsync(f func() error) <-chan error {
	resp := make(chan error, 1)
	go func() {
		err := f()
		resp <- err
		close(resp)
	}()
	return resp
}
