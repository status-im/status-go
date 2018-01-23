// Copyright (C) 2016  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package main

type telnetClient struct {
	addr string
}

func newTelnetClient(addr string) OpenTSDBConn {
	return &telnetClient{
		addr: addr,
	}
}

func (c *telnetClient) Put(d *DataPoint) error {
	panic("TODO")
}
