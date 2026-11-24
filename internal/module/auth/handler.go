package auth

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/pyrofloor/cloud-backend/internal/httpx"
)

const (
	maxLoginRequestBytes = 16 << 10
)

type LoginUseCase interface {
	Login(ctx context.Context, in LoginInput) (*LoginResult, error)
	LoginOwner(ctx context.Context, in LoginInput) (*LoginResult, error)
	LoginManager(ctx context.Context, in LoginInput) (*LoginResult, error)
	LoginResident(ctx context.Context, in LoginInput) (*LoginResult, error)
	Register(
		ctx context.Context,
		in RegisterInput,
	) error
}

type AuthCookieSetter interface {
	SetAuthCookies(w http.ResponseWriter, pair *TokenPair) error
}

type AuthHandler struct {
	service LoginUseCase
	cookies AuthCookieSetter
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerRequest struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phone_number"`
	Password    string `json:"password"`
}

type loginResponse struct {
	User userResponse `json:"user"`
}

type userResponse struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Phone     string `json:"phone,omitempty"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at,omitempty"`
}

func NewAuthHandler(
	service LoginUseCase,
	cookies AuthCookieSetter,
) (*AuthHandler, error) {
	if service == nil || cookies == nil {
		return nil, errors.New("auth service and cookie setter are required")
	}

	return &AuthHandler{
		service: service,
		cookies: cookies,
	}, nil
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var request registerRequest
	if err := httpx.DecodeJSON(w, r, maxLoginRequestBytes, &request); err != nil {
		httpx.WriteError(w, err)
		return
	}

	err := h.service.Register(r.Context(), RegisterInput{
		Name:     request.Name,
		Email:    request.Email,
		Phone:    request.PhoneNumber,
		Password: request.Password,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := httpx.WriteSuccess(w, http.StatusCreated, struct{}{}); err != nil {
		log.Printf("write registration response: %v", err)
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	h.loginWith(w, r, h.service.Login)
}

func (h *AuthHandler) LoginOwner(w http.ResponseWriter, r *http.Request) {
	h.loginWith(w, r, h.service.LoginOwner)
}

func (h *AuthHandler) LoginManager(w http.ResponseWriter, r *http.Request) {
	h.loginWith(w, r, h.service.LoginManager)
}

func (h *AuthHandler) LoginGuard(w http.ResponseWriter, r *http.Request) {
	h.loginWith(w, r, h.service.LoginManager)
}

func (h *AuthHandler) LoginResident(w http.ResponseWriter, r *http.Request) {
	h.loginWith(w, r, h.service.LoginResident)
}

func (h *AuthHandler) loginWith(
	w http.ResponseWriter,
	r *http.Request,
	login func(context.Context, LoginInput) (*LoginResult, error),
) {
	var request loginRequest
	if err := httpx.DecodeJSON(w, r, maxLoginRequestBytes, &request); err != nil {
		httpx.WriteError(w, err)
		return
	}

	result, err := login(r.Context(), LoginInput{
		Email:    request.Email,
		Password: request.Password,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	if result == nil || result.User == nil || result.Tokens == nil {
		httpx.WriteError(w, errors.New("auth login returned an incomplete result"))
		return
	}

	if err := h.cookies.SetAuthCookies(w, result.Tokens); err != nil {
		httpx.WriteError(w, err)
		return
	}

	if err := httpx.WriteSuccess(w, http.StatusOK, loginResponse{
		User: userResponse{
			ID:        result.User.IDNguoiDung,
			Role:      result.Role,
			Name:      result.User.TenNguoiDung,
			Email:     result.User.Email,
			Phone:     result.User.SoDienThoai,
			Status:    string(result.User.TrangThai),
			CreatedAt: result.User.NgayTao,
		},
	}); err != nil {
		log.Printf("write login response: %v", err)
	}
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	ClearAuthCookies(w)
	if err := httpx.WriteSuccess(w, http.StatusOK, map[string]string{"message": "logged out"}); err != nil {
		log.Printf("write logout response: %v", err)
	}
}
