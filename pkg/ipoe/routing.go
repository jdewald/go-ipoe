package ipoe

import (
	"fmt"
	"strings"
)

type RoutingTable interface {
	Route(dest string) (string, error)
}

type SameEmailRouting struct {
	DestEmail string
}

func (ser *SameEmailRouting) Route(dest string) (string, error) {
	tokes := strings.SplitN(ser.DestEmail, "@", 2)
	return fmt.Sprintf("%s+%s@%s", tokes[0], dest, tokes[1]), nil

}
