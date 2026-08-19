package runtime

import (
	"context"

	"github.com/tofchaliss/themis/benchmarks/internal/model"
)

type Runtime interface {
	Name() string

	Run(
		ctx context.Context,
		req model.Request,
	) (*model.Response, error)
}
