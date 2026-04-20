package capability

import "context"

// ctxKey is an unexported context key type used to attach verified Claims to a
// request context. Keeping the type unexported guarantees that no code outside
// this package can read or write the value under a colliding key.
type ctxKey struct{}

// WithClaims returns a new context carrying the supplied Claims. The
// requireCapability middleware calls this after a successful Verify so
// downstream handlers can access claims.Perms / claims.SID / claims.GrantID
// without re-parsing the token.
func WithClaims(ctx context.Context, c Claims) context.Context {
	return context.WithValue(ctx, ctxKey{}, c)
}

// ClaimsFromContext returns the Claims previously attached via WithClaims and
// true, or the zero Claims value and false when no claims are present. The
// zero-value path lets callers detect a context that never passed through the
// capability middleware.
func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	c, ok := ctx.Value(ctxKey{}).(Claims)
	return c, ok
}
