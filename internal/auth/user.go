package auth

import "context"

// User is the authenticated user of the request. It never carries the password hash.
type User struct {
	ID    int64  `json:"id" example:"1"`
	Email string `json:"email" example:"ana@farmacia.com"`
	Name  string `json:"name" example:"Ana Souza"`
	Role  Role   `json:"role" example:"reviewer"`
}

type ctxKey struct{}

// WithUser returns a context carrying the authenticated user.
func WithUser(ctx context.Context, u User) context.Context {
	return context.WithValue(ctx, ctxKey{}, u)
}

// UserFrom gets the user the middleware put in the context.
func UserFrom(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(ctxKey{}).(User)
	return u, ok
}
