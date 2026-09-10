package service

import (
	"context"

	"github.com/jayant132/seki/internal/auth"
	"github.com/jayant132/seki/internal/domain"
	"github.com/jayant132/seki/internal/repository"
)

type AuthService struct {
	users  *repository.UserRepository
	issuer *auth.Issuer
}

func NewAuthService(users *repository.UserRepository, issuer *auth.Issuer) *AuthService {
	return &AuthService{users: users, issuer: issuer}
}

func (s *AuthService) Register(ctx context.Context, email, password string) (*domain.User, string, error) {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, "", err
	}
	u, err := s.users.Create(ctx, email, hash, domain.RoleCustomer)
	if err != nil {
		return nil, "", err
	}
	token, err := s.issuer.Issue(u.ID, string(u.Role))
	if err != nil {
		return nil, "", err
	}
	return u, token, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*domain.User, string, error) {
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, "", domain.ErrInvalidCreds
		}
		return nil, "", err
	}
	if !auth.CheckPassword(u.PasswordHash, password) {
		return nil, "", domain.ErrInvalidCreds
	}
	token, err := s.issuer.Issue(u.ID, string(u.Role))
	if err != nil {
		return nil, "", err
	}
	return u, token, nil
}
