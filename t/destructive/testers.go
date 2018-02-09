package destructive

import (
	"github.com/vishvananda/netlink"
)

// LinkUpDownTester removes and restores default route.
type LinkUpDownTester struct {
	defRoute *netlink.Route
}

// Setup removes default route.
func (t *LinkUpDownTester) Setup() error {
	link, err := netlink.LinkByName("eth0")
	if err != nil {
		return err
	}
	// order is determentistic, but we can remove all routes if necessary
	routes, err := netlink.RouteList(link, netlink.FAMILY_V4)
	if err != nil {
		return err
	}
	t.defRoute = &routes[0]
	return netlink.RouteDel(&routes[0])
}

// TearDown removes default route.
func (t *LinkUpDownTester) TearDown() error {
	if t.defRoute != nil {
		return netlink.RouteAdd(t.defRoute)
	}
	return nil
}
