// Chat_Service/ws/hub_utils.go
package ws

import "context"

func ctxBackground() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), 2*1000_000_000) // 2s
	return ctx
}
