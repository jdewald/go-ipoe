package ipoe

import (
	"context"

	"github.com/songgao/water"
)

type IPOEReciver interface {
	Listen(ctx context.Context, intf *water.Interface)
}
