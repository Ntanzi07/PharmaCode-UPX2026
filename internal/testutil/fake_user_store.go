package testutil

import (
	"bytes"
	"context"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/db"
)

type fakeSession struct {
	tokenHash []byte
	userID    int64
	expiresAt time.Time
}

// FakeUserStore guarda usuários e sessões em memória. Implementa tanto o
// auth.Querier quanto o service.UserQuerier, então login, middleware e
// gestão de usuários enxergam os mesmos dados nos testes.
type FakeUserStore struct {
	mu       sync.Mutex
	nextID   int64
	users    map[int64]db.User
	sessions []fakeSession

	Now func() time.Time
}

func NewFakeUserStore() *FakeUserStore {
	return &FakeUserStore{nextID: 1, users: map[int64]db.User{}, Now: time.Now}
}

// SeedUser grava um usuário já com hash de senha (use auth.HashPassword).
func (f *FakeUserStore) SeedUser(u db.User) db.User {
	f.mu.Lock()
	defer f.mu.Unlock()
	if u.ID == 0 {
		u.ID = f.nextID
	}
	if u.ID >= f.nextID {
		f.nextID = u.ID + 1
	}
	f.users[u.ID] = u
	return u
}

// User devolve o usuário como está no "banco" (para asserts).
func (f *FakeUserStore) User(id int64) db.User {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.users[id]
}

// SessionCount conta as sessões de um usuário.
func (f *FakeUserStore) SessionCount(userID int64) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, s := range f.sessions {
		if s.userID == userID {
			n++
		}
	}
	return n
}

func (f *FakeUserStore) filterSessions(keep func(fakeSession) bool) {
	out := f.sessions[:0]
	for _, s := range f.sessions {
		if keep(s) {
			out = append(out, s)
		}
	}
	f.sessions = out
}

// ---- auth.Querier ----

func (f *FakeUserStore) GetUserByEmail(ctx context.Context, email string) (db.GetUserByEmailRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range f.users {
		if u.Email == email {
			return db.GetUserByEmailRow{ID: u.ID, Email: u.Email, Name: u.Name, PasswordHash: u.PasswordHash, Role: u.Role, Active: u.Active}, nil
		}
	}
	return db.GetUserByEmailRow{}, pgx.ErrNoRows
}

func (f *FakeUserStore) CreateSession(ctx context.Context, arg db.CreateSessionParams) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sessions = append(f.sessions, fakeSession{tokenHash: arg.TokenHash, userID: arg.UserID, expiresAt: arg.ExpiresAt.Time})
	return nil
}

func (f *FakeUserStore) GetSessionUser(ctx context.Context, tokenHash []byte) (db.GetSessionUserRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.sessions {
		if bytes.Equal(s.tokenHash, tokenHash) && s.expiresAt.After(f.Now()) {
			u := f.users[s.userID]
			if !u.Active {
				break
			}
			return db.GetSessionUserRow{ID: u.ID, Email: u.Email, Name: u.Name, Role: u.Role}, nil
		}
	}
	return db.GetSessionUserRow{}, pgx.ErrNoRows
}

func (f *FakeUserStore) DeleteSession(ctx context.Context, tokenHash []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.filterSessions(func(s fakeSession) bool { return !bytes.Equal(s.tokenHash, tokenHash) })
	return nil
}

func (f *FakeUserStore) DeleteExpiredSessions(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := f.Now()
	f.filterSessions(func(s fakeSession) bool { return s.expiresAt.After(now) })
	return nil
}

// ---- service.UserQuerier ----

func (f *FakeUserStore) CreateUser(ctx context.Context, arg db.CreateUserParams) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range f.users {
		if u.Email == arg.Email {
			return 0, PgError(CodeUniqueViolation)
		}
	}
	id := f.nextID
	f.nextID++
	f.users[id] = db.User{ID: id, Email: arg.Email, Name: arg.Name, PasswordHash: arg.PasswordHash, Role: arg.Role, Active: true}
	return id, nil
}

func (f *FakeUserStore) GetUserByID(ctx context.Context, id int64) (db.GetUserByIDRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.users[id]
	if !ok {
		return db.GetUserByIDRow{}, pgx.ErrNoRows
	}
	return db.GetUserByIDRow{ID: u.ID, Email: u.Email, Name: u.Name, Role: u.Role, Active: u.Active}, nil
}

func (f *FakeUserStore) ListUsers(ctx context.Context) ([]db.ListUsersRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []db.ListUsersRow
	for _, u := range f.users {
		out = append(out, db.ListUsersRow{ID: u.ID, Email: u.Email, Name: u.Name, Role: u.Role, Active: u.Active})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (f *FakeUserStore) UpdateUser(ctx context.Context, arg db.UpdateUserParams) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.users[arg.ID]
	if !ok {
		return 0, nil
	}
	for id, other := range f.users {
		if id != arg.ID && other.Email == arg.Email {
			return 0, PgError(CodeUniqueViolation)
		}
	}
	u.Email, u.Name, u.Role, u.Active = arg.Email, arg.Name, arg.Role, arg.Active
	u.UpdatedAt = pgtype.Timestamptz{Time: f.Now(), Valid: true}
	f.users[arg.ID] = u
	return 1, nil
}

func (f *FakeUserStore) UpdateUserPassword(ctx context.Context, arg db.UpdateUserPasswordParams) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.users[arg.ID]
	if !ok {
		return 0, nil
	}
	u.PasswordHash = arg.PasswordHash
	f.users[arg.ID] = u
	return 1, nil
}

func (f *FakeUserStore) GetUserPasswordHash(ctx context.Context, id int64) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.users[id]
	if !ok || !u.Active {
		return "", pgx.ErrNoRows
	}
	return u.PasswordHash, nil
}

func (f *FakeUserStore) CountUsers(ctx context.Context) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return int64(len(f.users)), nil
}

func (f *FakeUserStore) CountActiveAdmins(ctx context.Context) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var n int64
	for _, u := range f.users {
		if u.Role == "admin" && u.Active {
			n++
		}
	}
	return n, nil
}

func (f *FakeUserStore) DeleteUserSessions(ctx context.Context, userID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.filterSessions(func(s fakeSession) bool { return s.userID != userID })
	return nil
}

func (f *FakeUserStore) DeleteUserSessionsExcept(ctx context.Context, arg db.DeleteUserSessionsExceptParams) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.filterSessions(func(s fakeSession) bool {
		return s.userID != arg.UserID || bytes.Equal(s.tokenHash, arg.KeepTokenHash)
	})
	return nil
}
