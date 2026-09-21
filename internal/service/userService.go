package service

import (
	"context"
	"errors"
	"net/mail"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/auth"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/db"
)

type UserQuerier interface {
	CreateUser(ctx context.Context, arg db.CreateUserParams) (int64, error)
	GetUserByID(ctx context.Context, id int64) (db.GetUserByIDRow, error)
	ListUsers(ctx context.Context) ([]db.ListUsersRow, error)
	UpdateUser(ctx context.Context, arg db.UpdateUserParams) (int64, error)
	UpdateUserPassword(ctx context.Context, arg db.UpdateUserPasswordParams) (int64, error)
	GetUserPasswordHash(ctx context.Context, id int64) (string, error)
	CountUsers(ctx context.Context) (int64, error)
	CountActiveAdmins(ctx context.Context) (int64, error)
	DeleteUserSessions(ctx context.Context, userID int64) error
	DeleteUserSessionsExcept(ctx context.Context, arg db.DeleteUserSessionsExceptParams) error
}

type UserService struct {
	queries UserQuerier
}

func NewUserService(q UserQuerier) *UserService {
	return &UserService{queries: q}
}

type CreateUserInput struct {
	Name     string
	Email    string
	Password string
	Role     auth.Role
}

type UpdateUserInput struct {
	Name   string
	Email  string
	Role   auth.Role
	Active bool
}

func validateUser(name, email string, role auth.Role) (string, error) {
	if name == "" {
		return "", ErrInvalidUserName
	}
	email = auth.NormalizeEmail(email)
	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
		return "", ErrInvalidEmail
	}
	if !role.Valid() {
		return "", ErrInvalidRole
	}
	return email, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (s *UserService) Create(ctx context.Context, in CreateUserInput) (int64, error) {
	email, err := validateUser(in.Name, in.Email, in.Role)
	if err != nil {
		return 0, err
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return 0, err
	}
	id, err := s.queries.CreateUser(ctx, db.CreateUserParams{
		Email:        email,
		Name:         in.Name,
		PasswordHash: hash,
		Role:         string(in.Role),
	})
	if isUniqueViolation(err) {
		return 0, ErrDuplicateEmail
	}
	return id, err
}

func (s *UserService) List(ctx context.Context) ([]db.ListUsersRow, error) {
	return s.queries.ListUsers(ctx)
}

func (s *UserService) Get(ctx context.Context, id int64) (db.GetUserByIDRow, error) {
	row, err := s.queries.GetUserByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.GetUserByIDRow{}, ErrUserNotFound
	}
	return row, err
}

// Update altera dados, papel e status. Não deixa o sistema ficar sem nenhum
// admin ativo. Se o usuário foi desativado ou mudou de papel, derruba as
// sessões dele para a mudança valer na hora.
func (s *UserService) Update(ctx context.Context, id int64, in UpdateUserInput) error {
	email, err := validateUser(in.Name, in.Email, in.Role)
	if err != nil {
		return err
	}
	current, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	losingAdmin := current.Role == string(auth.RoleAdmin) && current.Active &&
		(in.Role != auth.RoleAdmin || !in.Active)
	if losingAdmin {
		admins, err := s.queries.CountActiveAdmins(ctx)
		if err != nil {
			return err
		}
		if admins <= 1 {
			return ErrLastAdmin
		}
	}

	rows, err := s.queries.UpdateUser(ctx, db.UpdateUserParams{
		ID:     id,
		Email:  email,
		Name:   in.Name,
		Role:   string(in.Role),
		Active: in.Active,
	})
	if isUniqueViolation(err) {
		return ErrDuplicateEmail
	}
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUserNotFound
	}

	if !in.Active || string(in.Role) != current.Role {
		return s.queries.DeleteUserSessions(ctx, id)
	}
	return nil
}

// SetPassword é a troca feita pelo admin: define a senha e derruba todas as sessões do usuário.
func (s *UserService) SetPassword(ctx context.Context, id int64, password string) error {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	rows, err := s.queries.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{ID: id, PasswordHash: hash})
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUserNotFound
	}
	return s.queries.DeleteUserSessions(ctx, id)
}

// ChangeOwnPassword é a troca feita pelo próprio usuário: exige a senha atual
// e mantém só a sessão em uso.
func (s *UserService) ChangeOwnPassword(ctx context.Context, id int64, sessionToken, current, next string) error {
	hash, err := s.queries.GetUserPasswordHash(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrUserNotFound
	}
	if err != nil {
		return err
	}
	if !auth.CheckPassword(hash, current) {
		return ErrWrongPassword
	}
	newHash, err := auth.HashPassword(next)
	if err != nil {
		return err
	}
	if _, err := s.queries.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{ID: id, PasswordHash: newHash}); err != nil {
		return err
	}
	return s.queries.DeleteUserSessionsExcept(ctx, db.DeleteUserSessionsExceptParams{
		UserID:        id,
		KeepTokenHash: auth.HashToken(sessionToken),
	})
}

// EnsureAdmin cria o primeiro admin quando o banco ainda não tem nenhum usuário.
// Devolve true se criou.
func (s *UserService) EnsureAdmin(ctx context.Context, name, email, password string) (bool, error) {
	n, err := s.queries.CountUsers(ctx)
	if err != nil || n > 0 {
		return false, err
	}
	if _, err := s.Create(ctx, CreateUserInput{Name: name, Email: email, Password: password, Role: auth.RoleAdmin}); err != nil {
		return false, err
	}
	return true, nil
}
