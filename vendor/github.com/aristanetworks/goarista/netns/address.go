// Copyright (C) 2016  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package netns

import (
	"fmt"
	"strings"
)

// ParseAddress takes in an address string, parsing out the address
// and an optional VRF name. It returns the namespace corresponding to the VRF.
// The expected form is [<vrf-name>/]address:port. However, ParseAddress
// will not actually check to see if the VRF name or address are valid.
// Presumably, when those values are used later, they will fail if they
// are malformed
func ParseAddress(address string) (nsName string, addr string, err error) {
	split := strings.Split(address, "/")
	if l := len(split); l == 1 {
		addr = split[0]
	} else if l == 2 {
		nsName = VRFToNetNS(split[0])
		addr = split[1]
	} else {
		err = fmt.Errorf("Could not parse out a <vrf-name>/address for %s", address)
	}
	return
}

// VRFToNetNS converts a VRF name to network namespace's name corresponding to that VRF.
func VRFToNetNS(vrf string) string {
	if vrf != "" && vrf != "default" && !strings.HasPrefix(vrf, "ns-") {
		vrf = "ns-" + vrf
	}
	return vrf
}
