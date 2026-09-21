package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/auth"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/db"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/service"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/testutil"
)

func TestUserService_Create(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewFakeUserStore()
	svc := service.NewUserService(store)

	id, err := svc.Create(ctx, service.CreateUserInput{Name: "Ana", Email: " Ana@Farmacia.COM ", Password: "senha-forte-1", Role: auth.RoleReviewer})
	require.NoError(t, err)

	saved := store.User(id)
	assert.Equal(t, "ana@farmacia.com", saved.Email, "email é guardado normalizado")
	assert.NotEqual(t, "senha-forte-1", saved.PasswordHash, "senha nunca é guardada em texto")
	assert.True(t, auth.CheckPassword(saved.PasswordHash, "senha-forte-1"))

	tests := []struct {
		name string
		in   service.CreateUserInput
		want error
	}{
		{"email repetido", service.CreateUserInput{Name: "X", Email: "ana@farmacia.com", Password: "senha-forte-1", Role: auth.RoleEditor}, service.ErrDuplicateEmail},
		{"email inválido", service.CreateUserInput{Name: "X", Email: "nao-e-email", Password: "senha-forte-1", Role: auth.RoleEditor}, service.ErrInvalidEmail},
		{"sem nome", service.CreateUserInput{Email: "b@x.com", Password: "senha-forte-1", Role: auth.RoleEditor}, service.ErrInvalidUserName},
		{"papel inválido", service.CreateUserInput{Name: "X", Email: "b@x.com", Password: "senha-forte-1", Role: "root"}, service.ErrInvalidRole},
		{"senha curta", service.CreateUserInput{Name: "X", Email: "b@x.com", Password: "123", Role: auth.RoleEditor}, auth.ErrWeakPassword},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Create(ctx, tt.in)
			assert.ErrorIs(t, err, tt.want)
		})
	}
}

func TestUserService_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("não deixa rebaixar nem desativar o último admin", func(t *testing.T) {
		store := testutil.NewFakeUserStore()
		admin := store.SeedUser(db.User{Name: "Admin", Email: "admin@x.com", Role: "admin", Active: true})
		svc := service.NewUserService(store)

		err := svc.Update(ctx, admin.ID, service.UpdateUserInput{Name: "Admin", Email: "admin@x.com", Role: auth.RoleEditor, Active: true})
		assert.ErrorIs(t, err, service.ErrLastAdmin)

		err = svc.Update(ctx, admin.ID, service.UpdateUserInput{Name: "Admin", Email: "admin@x.com", Role: auth.RoleAdmin, Active: false})
		assert.ErrorIs(t, err, service.ErrLastAdmin)
	})

	t.Run("com outro admin ativo, pode rebaixar", func(t *testing.T) {
		store := testutil.NewFakeUserStore()
		a1 := store.SeedUser(db.User{Name: "A1", Email: "a1@x.com", Role: "admin", Active: true})
		store.SeedUser(db.User{Name: "A2", Email: "a2@x.com", Role: "admin", Active: true})
		svc := service.NewUserService(store)

		err := svc.Update(ctx, a1.ID, service.UpdateUserInput{Name: "A1", Email: "a1@x.com", Role: auth.RoleReviewer, Active: true})
		require.NoError(t, err)
		assert.Equal(t, "reviewer", store.User(a1.ID).Role)
	})

	t.Run("desativar derruba as sessões do usuário", func(t *testing.T) {
		store := testutil.NewFakeUserStore()
		hash, _ := auth.HashPassword("senha-forte-1")
		u := store.SeedUser(db.User{Name: "Ed", Email: "ed@x.com", PasswordHash: hash, Role: "editor", Active: true})
		authSvc := auth.NewService(store, time.Hour)
		token, _, _, err := authSvc.Login(ctx, "ed@x.com", "senha-forte-1")
		require.NoError(t, err)

		svc := service.NewUserService(store)
		require.NoError(t, svc.Update(ctx, u.ID, service.UpdateUserInput{Name: "Ed", Email: "ed@x.com", Role: auth.RoleEditor, Active: false}))

		_, err = authSvc.Authenticate(ctx, token)
		assert.ErrorIs(t, err, auth.ErrUnauthenticated)
	})

	t.Run("id inexistente", func(t *testing.T) {
		svc := service.NewUserService(testutil.NewFakeUserStore())
		err := svc.Update(ctx, 999, service.UpdateUserInput{Name: "X", Email: "x@x.com", Role: auth.RoleEditor, Active: true})
		assert.ErrorIs(t, err, service.ErrUserNotFound)
	})
}

func TestUserService_ChangeOwnPassword(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewFakeUserStore()
	hash, _ := auth.HashPassword("senha-antiga-1")
	u := store.SeedUser(db.User{Name: "Ed", Email: "ed@x.com", PasswordHash: hash, Role: "editor", Active: true})
	authSvc := auth.NewService(store, time.Hour)
	atual, _, _, _ := authSvc.Login(ctx, "ed@x.com", "senha-antiga-1")
	outra, _, _, _ := authSvc.Login(ctx, "ed@x.com", "senha-antiga-1")
	svc := service.NewUserService(store)

	err := svc.ChangeOwnPassword(ctx, u.ID, atual, "errada-123", "senha-nova-123")
	assert.ErrorIs(t, err, service.ErrWrongPassword)

	require.NoError(t, svc.ChangeOwnPassword(ctx, u.ID, atual, "senha-antiga-1", "senha-nova-123"))
	assert.True(t, auth.CheckPassword(store.User(u.ID).PasswordHash, "senha-nova-123"))

	_, err = authSvc.Authenticate(ctx, atual)
	assert.NoError(t, err, "a sessão em uso continua valendo")
	_, err = authSvc.Authenticate(ctx, outra)
	assert.ErrorIs(t, err, auth.ErrUnauthenticated, "as outras sessões são encerradas")
}

func TestUserService_EnsureAdmin(t *testing.T) {
	ctx := context.Background()
	store := testutil.NewFakeUserStore()
	svc := service.NewUserService(store)

	created, err := svc.EnsureAdmin(ctx, "Admin", "admin@x.com", "senha-forte-1")
	require.NoError(t, err)
	assert.True(t, created)

	created, err = svc.EnsureAdmin(ctx, "Outro", "outro@x.com", "senha-forte-1")
	require.NoError(t, err)
	assert.False(t, created, "com usuários no banco, não cria de novo")
}
