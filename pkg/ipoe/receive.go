package ipoe

import "context"

type IPOEReciver interface {
	Listen(ctx context.Context)
}
