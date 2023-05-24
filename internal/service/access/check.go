package access

import (
	"context"
	"errors"
	"fmt"
	"github.com/Arkosh744/auth-service-api/internal/pkg/token"
	"google.golang.org/grpc/metadata"
	"strings"
)

const authPrefix = "Bearer "

var accessibleRoles map[string]string

func (s *service) CheckAccess(ctx context.Context, endpointAddress string) (bool, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return false, errors.New("metadata is not provided")
	}

	authHeader, ok := md["authorization"]
	if !ok || len(authHeader) == 0 {
		return false, errors.New("authorization header is not provided")
	}

	if !strings.HasPrefix(authHeader[0], authPrefix) {
		return false, errors.New("invalid authorization header format")
	}

	accessToken := strings.TrimPrefix(authHeader[0], authPrefix)

	claims, err := token.VerifyToken(accessToken, s.authConfig.AccessTokenSecretKey())
	if err != nil {
		return false, errors.New("access token is invalid")
	}

	accessibleMap, err := s.accessibleRoles(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get accessible roles: %v", err)
	}

	role, ok := accessibleMap[endpointAddress]
	if !ok {
		return true, nil
	}

	if role == claims.Role {
		return true, nil
	}

	return false, errors.New("access denied")
}

func (s *service) accessibleRoles(ctx context.Context) (map[string]string, error) {
	if accessibleRoles == nil {
		accessibleRoles = make(map[string]string)

		accessInfo, err := s.repo.GetList(ctx)
		if err != nil {
			return nil, err
		}

		for _, info := range accessInfo {
			accessibleRoles[info.EndpointAddress] = info.Role
		}
	}

	return accessibleRoles, nil
}