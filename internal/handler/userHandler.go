package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/auth"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/db"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/service"
)

type UserHandler struct {
	service *service.UserService
}

func NewUserHandler(s *service.UserService) *UserHandler {
	return &UserHandler{service: s}
}

type createUserRequest struct {
	Name     string    `json:"name" example:"Ana Souza"`
	Email    string    `json:"email" example:"ana@farmacia.com"`
	Password string    `json:"password" example:"uma-senha-forte"`
	Role     auth.Role `json:"role" example:"reviewer" enums:"editor,reviewer,admin"`
}

type updateUserRequest struct {
	Name   string    `json:"name" example:"Ana Souza"`
	Email  string    `json:"email" example:"ana@farmacia.com"`
	Role   auth.Role `json:"role" example:"reviewer" enums:"editor,reviewer,admin"`
	Active bool      `json:"active" example:"true"`
}

type setPasswordRequest struct {
	Password string `json:"password" example:"nova-senha-forte"`
}

// userErrorStatus traduz erros do UserService para HTTP.
func userErrorStatus(err error) (int, bool) {
	switch {
	case errors.Is(err, service.ErrUserNotFound):
		return http.StatusNotFound, true
	case errors.Is(err, service.ErrDuplicateEmail), errors.Is(err, service.ErrLastAdmin):
		return http.StatusConflict, true
	case errors.Is(err, service.ErrInvalidEmail), errors.Is(err, service.ErrInvalidUserName),
		errors.Is(err, service.ErrInvalidRole), errors.Is(err, auth.ErrWeakPassword):
		return http.StatusBadRequest, true
	}
	return 0, false
}

func writeUserError(w http.ResponseWriter, err error, action string) {
	if status, ok := userErrorStatus(err); ok {
		http.Error(w, err.Error(), status)
		return
	}
	log.Printf("failed to %s: %v", action, err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

// ListUsers godoc
// @Summary      Lista usuários do painel (admin)
// @Tags         users
// @Produce      json
// @Success      200  {array}   db.ListUsersRow
// @Failure      401  {string}  string  "authentication required"
// @Failure      403  {string}  string  "insufficient permissions"
// @Router       /users [get]
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.List(r.Context())
	if err != nil {
		writeUserError(w, err, "list users")
		return
	}
	if users == nil {
		users = []db.ListUsersRow{}
	}
	writeJSON(w, http.StatusOK, users)
}

// CreateUser godoc
// @Summary      Cria um usuário do painel (admin)
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body      createUserRequest  true  "Dados do usuário (senha de 8 a 72 caracteres)"
// @Success      201   {object}  idResponse
// @Failure      400   {string}  string  "dados inválidos"
// @Failure      409   {string}  string  "email already registered"
// @Router       /users [post]
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	id, err := h.service.Create(r.Context(), service.CreateUserInput{
		Name:     strings.TrimSpace(req.Name),
		Email:    req.Email,
		Password: req.Password,
		Role:     req.Role,
	})
	if err != nil {
		writeUserError(w, err, "create user")
		return
	}
	writeJSON(w, http.StatusCreated, idResponse{ID: id})
}

// UpdateUser godoc
// @Summary      Atualiza nome, email, papel e status (admin)
// @Description  Desativar ou trocar o papel encerra as sessões do usuário. O último admin ativo não pode ser rebaixado nem desativado.
// @Tags         users
// @Accept       json
// @Param        id    path  int                true  "ID do usuário"
// @Param        body  body  updateUserRequest  true  "Dados do usuário"
// @Success      204
// @Failure      400  {string}  string  "dados inválidos"
// @Failure      404  {string}  string  "user not found"
// @Failure      409  {string}  string  "email already registered / cannot remove the last active admin"
// @Router       /users/{id} [put]
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	err = h.service.Update(r.Context(), id, service.UpdateUserInput{
		Name:   strings.TrimSpace(req.Name),
		Email:  req.Email,
		Role:   req.Role,
		Active: req.Active,
	})
	if err != nil {
		writeUserError(w, err, "update user")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// SetUserPassword godoc
// @Summary      Define uma senha nova para o usuário (admin)
// @Description  Encerra todas as sessões do usuário.
// @Tags         users
// @Accept       json
// @Param        id    path  int                 true  "ID do usuário"
// @Param        body  body  setPasswordRequest  true  "Senha nova (8 a 72 caracteres)"
// @Success      204
// @Failure      400  {string}  string  "senha fraca"
// @Failure      404  {string}  string  "user not found"
// @Router       /users/{id}/password [put]
func (h *UserHandler) SetUserPassword(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var req setPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := h.service.SetPassword(r.Context(), id, req.Password); err != nil {
		writeUserError(w, err, "set user password")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
