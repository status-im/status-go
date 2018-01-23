// Copyright (C) 2016  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package main

// DataPoint for OpenTSDB to store.
type DataPoint struct {
	// Metric name.
	Metric string `json:"metric"`

	// UNIX timestamp with millisecond resolution.
	Timestamp uint64 `json:"timestamp"`

	// Value of the data point (integer or floating point).
	Value interface{} `json:"value"`

	// Tags.  The host is automatically populated by the OpenTSDBConn.
	Tags map[string]string `json:"tags"`
}

// OpenTSDBConn is a managed connection to an OpenTSDB instance (or cluster).
type OpenTSDBConn interface {
	Put(d *DataPoint) error
}
