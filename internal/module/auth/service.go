package auth

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/pyrofloor/cloud-backend/internal/module/nguoidung"
)

type TokenIssuer interface {
	IssuePair(userID, roleID string) (*TokenPair, error)
}

type ScopedTokenIssuer interface {
	IssuePairWithScope(userID, roleID string, scope TokenScope) (*TokenPair, error)
}

type NguoiDungRepository interface {
	CreateUser(
		ctx context.Context,
		name, email string,
		phone string,
		passwordHash string,
	) error

	FindByEmail(
		ctx context.Context,
		email string,
	) (*nguoidung.LoginRecord, error)
}

type ActorLoginRepository interface {
	FindOwnerByEmail(ctx context.Context, email string) (*nguoidung.LoginRecord, error)
	FindManagerByEmail(ctx context.Context, email string) (*nguoidung.LoginRecord, error)
	FindResidentByEmail(ctx context.Context, email string) (*nguoidung.LoginRecord, error)
}

type Service struct {
	repo   NguoiDungRepository
	tokens TokenIssuer
}

func NewService(repo NguoiDungRepository, tokens TokenIssuer) *Service {
	return &Service{
		repo:   repo,
		tokens: tokens,
	}
}

type RegisterInput struct {
	Name     string
	Email    string
	Phone    string
	Password string
}

type LoginInput struct {
	Email    string
	Password string
}

type LoginResult struct {
	User   *nguoidung.NguoiDung
	Role   string
	Tokens *TokenPair
}

func normalizeEmail(email string) (string, error) {
	email = strings.TrimSpace(email)

	if len(email) > 254 {
		return "", errors.New("email is too long")
	}

	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email ||
		!strings.Contains(email, "@") {
		return "", errors.New("invalid email")
	}

	return strings.ToLower(email), nil
}

func (s *Service) Register(
	ctx context.Context,
	in RegisterInput,
) error {
	name := strings.TrimSpace(in.Name)

	if name == "" || !utf8.ValidString(name) ||
		utf8.RuneCountInString(name) > 100 {
		return errors.New("invalid name")
	}

	email, err := normalizeEmail(in.Email)
	if err != nil {
		return err
	}

	if !utf8.ValidString(in.Password) ||
		len(in.Password) < 8 ||
		len(in.Password) > 50 {
		return errors.New(
			"password must be between 12 and 72 bytes",
		)
	}

	value := strings.TrimSpace(in.Phone)
	if value == "" {
		return errors.New("phone is too long")
	} else if len(value) > 10 {
		return errors.New("phone is too long")
	}

	passwordHash, err := HashPassword(in.Password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	return s.repo.CreateUser(
		ctx,
		name,
		email,
		in.Phone,
		passwordHash,
	)
}

func (s *Service) Login(
	ctx context.Context,
	in LoginInput,
) (*LoginResult, error) {
	return s.login(ctx, in, "", "", false)
}

func (s *Service) LoginOwner(ctx context.Context, in LoginInput) (*LoginResult, error) {
	return s.login(ctx, in, nguoidung.RoleCodeOwner, "owner", true)
}

func (s *Service) LoginManager(ctx context.Context, in LoginInput) (*LoginResult, error) {
	return s.login(ctx, in, nguoidung.RoleCodeManager, "manager", true)
}

func (s *Service) LoginResident(ctx context.Context, in LoginInput) (*LoginResult, error) {
	return s.login(ctx, in, nguoidung.RoleCodeResident, "resident", true)
}

func (s *Service) login(
	ctx context.Context,
	in LoginInput,
	expectedRole, actor string,
	actorLogin bool,
) (*LoginResult, error) {
	email, err := normalizeEmail(in.Email)
	if err != nil {
		// Trả cùng lỗi với thông tin đăng nhập không hợp lệ.
		return nil, ErrInvalidLogin
	}

	if in.Password == "" || len(in.Password) > MaxPasswordBytes {
		return nil, ErrInvalidLogin
	}

	var rec *nguoidung.LoginRecord
	if actorLogin {
		actorRepo, ok := s.repo.(ActorLoginRepository)
		if !ok {
			return nil, errors.New("actor login repository is not configured")
		}
		switch actor {
		case "owner":
			rec, err = actorRepo.FindOwnerByEmail(ctx, email)
		case "manager":
			rec, err = actorRepo.FindManagerByEmail(ctx, email)
		case "resident":
			rec, err = actorRepo.FindResidentByEmail(ctx, email)
		default:
			return nil, ErrInvalidLogin
		}
	} else {
		rec, err = s.repo.FindByEmail(ctx, email)
	}
	if errors.Is(err, nguoidung.ErrUserNotFound) {
		return nil, ErrInvalidLogin
	}
	if err != nil {
		return nil, err
	}

	ok, err := VerifyPassword(in.Password, rec.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("verify password hash: %w", err)
	}

	if !ok {
		return nil, ErrInvalidLogin
	}

	if rec.User.TrangThai != nguoidung.HoatDong {
		return nil, ErrAccountDeleted
	}

	roleMatches := rec.Role == expectedRole
	if actor == "manager" {
		roleMatches = rec.Role == nguoidung.RoleCodeManager
	}
	if actorLogin && !roleMatches {
		return nil, ErrInvalidLogin
	}

	scope := TokenScope{IDToaNha: rec.IDToaNha, IDKhuVuc: rec.IDKhuVuc}
	var tokens *TokenPair
	if scopedIssuer, ok := s.tokens.(ScopedTokenIssuer); ok {
		tokens, err = scopedIssuer.IssuePairWithScope(rec.User.IDNguoiDung, rec.Role, scope)
	} else {
		tokens, err = s.tokens.IssuePair(rec.User.IDNguoiDung, rec.Role)
	}
	if err != nil {
		return nil, fmt.Errorf("issue tokens: %w", err)
	}

	return &LoginResult{
		User:   &rec.User,
		Role:   rec.Role,
		Tokens: tokens,
	}, nil
}
