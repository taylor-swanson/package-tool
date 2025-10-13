package modify

import "github.com/andrewkroh/go-fleetpkg"

type Modifier struct {
	Name        string
	Description string
	Run         func(pkg *fleetpkg.Integration) error
}
