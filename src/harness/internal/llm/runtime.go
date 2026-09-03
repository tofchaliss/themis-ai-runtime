package llm

import (
	"context"
)

type Runtime interface {
	Name() string

	Run(
		ctx context.Context,
		req Request,
	) (*Response, error)
}
