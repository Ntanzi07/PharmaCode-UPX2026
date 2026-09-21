package auth

import "context"

// User é o usuário autenticado da requisição. Nunca carrega o hash da senha.
type User struct {
	ID    int64  `json:"id" example:"1"`
	Email string `json:"email" example:"ana@farmacia.com"`
	Name  string `json:"name" example:"Ana Souza"`
	Role  Role   `json:"role" example:"reviewer"`
}

type ctxKey struct{}

// WithUser devolve um contexto com o usuário autenticado.
func WithUser(ctx context.Context, u User) context.Context {
	return context.WithValue(ctx, ctxKey{}, u)
}

// UserFrom pega o usuário que o middleware colocou no contexto.
func UserFrom(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(ctxKey{}).(User)
	return u, ok
}
