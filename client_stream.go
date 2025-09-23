package http

import (
	"context"

	"go.unistack.org/micro/v4/client"
)

// TODO: Add stream support in the future.

func (c *Client) fnStream(context.Context, client.Request, ...client.CallOption) (client.Stream, error) {
	panic("not implemented")
}
