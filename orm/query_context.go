package orm

import "context"

type fullScanKey struct{}

// WithFullScan explicitly allows a query without index conditions.
func WithFullScan(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, fullScanKey{}, true)
}

// FullScanAllowed reports whether ctx explicitly allows an unfiltered query.
func FullScanAllowed(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	allowed, _ := ctx.Value(fullScanKey{}).(bool)
	return allowed
}
