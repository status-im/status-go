package destructive

import (
	"github.com/vishvananda/netlink"
)

// NetworkConnectionController removes and restores network connection.
type NetworkConnectionController struct {
	defRoute *netlink.Route
}

// Enable removes default route.
func (t *NetworkConnectionController) Enable() error {
	link, err := netlink.LinkByName("eth0")
	if err != nil {
		return err
	}
	// order is determentistic, but we can remove all routes if necessary
	routes, err := netlink.RouteList(link, 4)
	if err != nil {
		return err
	}
	t.defRoute = &routes[0]
	return netlink.RouteDel(&routes[0])
}

// Disable removes default route.
func (t *NetworkConnectionController) Disable() error {
	if t.defRoute != nil {
		return netlink.RouteAdd(t.defRoute)
	}
	return nil
}
