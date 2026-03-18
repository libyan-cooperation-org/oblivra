package oql

import "context"

func cancelled(ctx context.Context, i int) bool {
	if i&1023 != 0 {
		return false
	}
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
