// Copyright (C) 2016  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package main

import "fmt"

type textDumper struct{}

func newTextDumper() OpenTSDBConn {
	return textDumper{}
}

func (t textDumper) Put(d *DataPoint) error {
	var tags string
	if len(d.Tags) != 0 {
		for tag, value := range d.Tags {
			tags += " " + tag + "=" + value
		}
	}
	fmt.Printf("put %s %d %#v%s\n", d.Metric, d.Timestamp, d.Value, tags)
	return nil
}
