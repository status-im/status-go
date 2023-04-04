package pairing

import (
	"fmt"

	"github.com/schollz/peerdiscovery"
)

func Search() {
	discoveries, _ := peerdiscovery.Discover(peerdiscovery.Settings{Limit: 1, AllowSelf: true})
	for _, d := range discoveries {
		fmt.Printf("discovered '%s'\n", d.Address)
	}
}
