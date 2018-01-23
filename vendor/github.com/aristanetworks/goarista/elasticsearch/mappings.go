// Copyright (C) 2016  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package elasticsearch

import (
	"strings"
)

// EscapeFieldName escapes field names for Elasticsearch
func EscapeFieldName(name string) string {
	return strings.Replace(name, ".", "_", -1)
}
