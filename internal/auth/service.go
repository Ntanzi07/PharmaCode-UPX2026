package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/db"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUnauthenticated    = errors.New("authentication required")
)

// Querier é o pedaço do sqlc que o login usa (facilita o fake nos testes).
type Querier interface {
	GetUserByEmail(ctx context.Context, email string) (db.GetUserByEmailRow, error)
	CreateSession(ctx context.Context, arg db.CreateSessionParams) error
	GetSessionUser(ctx context.Context, tokenHash []byte) (db.GetSessionUserRow, error)
	DeleteSession(ctx context.Context, tokenHash []byte) error
	DeleteExpiredSessions(ctx context.Context) error
}

type Service struct {
	q   Querier
	ttl time.Duration
	now func() time.Time
}

func NewService(q Querier, sessionTTL time.Duration) *Service {
	return &Service{q: q, ttl: sessionTTL, now: time.Now}
}

// NormalizeEmail deixa o email no formato em que ele é guardado no banco.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// HashToken é o que vai para o banco: o SHA-256 do token do cookie.
func HashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Login confere email e senha e cria uma sessão. Devolve o token que vai no cookie.
func (s *Service) Login(ctx context.Context, email, password string) (string, User, time.Time, error) {
	row, err := s.q.GetUserByEmail(ctx, NormalizeEmail(email))
	if errors.Is(err, pgx.ErrNoRows) {
		CheckPassword(string(dummyHash), password) // mesmo custo de quando o email existe
		return "", User{}, time.Time{}, ErrInvalidCredentials
	}
	if err != nil {
		return "", User{}, time.Time{}, err
	}
	if !CheckPassword(row.PasswordHash, password) || !row.Active {
		return "", User{}, time.Time{}, ErrInvalidCredentials
	}

	token, err := newToken()
	if err != nil {
		return "", User{}, time.Time{}, err
	}
	expires := s.now().Add(s.ttl)
	if err := s.q.CreateSession(ctx, db.CreateSessionParams{
		TokenHash: HashToken(token),
		UserID:    row.ID,
		ExpiresAt: pgtype.Timestamptz{Time: expires, Valid: true},
	}); err != nil {
		return "", User{}, time.Time{}, err
	}

	// Faxina: aproveita o login para apagar sessões vencidas.
	if err := s.q.DeleteExpiredSessions(ctx); err != nil {
		log.Printf("failed to delete expired sessions: %v", err)
	}

	return token, User{ID: row.ID, Email: row.Email, Name: row.Name, Role: Role(row.Role)}, expires, nil
}

// Authenticate devolve o dono da sessão, se ela existir, não tiver vencido e o usuário estiver ativo.
func (s *Service) Authenticate(ctx context.Context, token string) (User, error) {
	if token == "" {
		return User{}, ErrUnauthenticated
	}
	row, err := s.q.GetSessionUser(ctx, HashToken(token))
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUnauthenticated
	}
	if err != nil {
		return User{}, err
	}
	return User{ID: row.ID, Email: row.Email, Name: row.Name, Role: Role(row.Role)}, nil
}

// Logout apaga a sessão.
func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.q.DeleteSession(ctx, HashToken(token))
}
