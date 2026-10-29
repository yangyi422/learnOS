package auth

import "context"

type Principal struct {
	UserID   uint
	Username string
	Role     string
	// Scopes is nil for a full session and populated for scoped API tokens, so a
	// token holder can never reach endpoints its scope does not name.
	Scopes []string
}

// Allows reports whether the principal may act on one scope. A session
// principal (nil scopes) allows everything it is otherwise authorised for.
func (p Principal) Allows(scope string) bool {
	if p.Scopes == nil {
		return true
	}
	for _, candidate := range p.Scopes {
		if candidate == scope {
			return true
		}
	}
	return false
}

type contextKey struct{}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, contextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(contextKey{}).(Principal)
	return principal, ok
}
