package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey struct{}

var ownerIDKey = contextKey{}

type Config struct {
	Enabled    bool
	HMACSecret string
	Issuer     string
	Audience   string
}

type Claims struct {
	jwt.RegisteredClaims
	OwnerID string `json:"owner_id"`
}

func OwnerIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ownerIDKey).(string)
	return v, ok && v != ""
}

func WithOwnerID(ctx context.Context, ownerID string) context.Context {
	return context.WithValue(ctx, ownerIDKey, ownerID)
}

func UnaryServerInterceptor(cfg Config) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !cfg.Enabled || isPublicMethod(info.FullMethod) {
			return handler(ctx, req)
		}

		ownerID, err := extractOwnerFromMetadata(ctx, cfg)
		if err != nil {
			return nil, err
		}
		return handler(WithOwnerID(ctx, ownerID), req)
	}
}

func isPublicMethod(fullMethod string) bool {
	return strings.HasPrefix(fullMethod, "/grpc.health.v1.Health/")
}

func extractOwnerFromMetadata(ctx context.Context, cfg Config) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}

	token := bearerToken(md.Get("authorization"))
	if token == "" {
		return "", status.Error(codes.Unauthenticated, "missing bearer token")
	}

	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(cfg.HMACSecret), nil
	}, jwt.WithIssuer(cfg.Issuer), jwt.WithAudience(cfg.Audience))
	if err != nil || !parsed.Valid {
		return "", status.Error(codes.Unauthenticated, "invalid token")
	}

	ownerID := claims.OwnerID
	if ownerID == "" {
		ownerID = claims.Subject
	}
	if ownerID == "" {
		return "", status.Error(codes.Unauthenticated, "token missing owner identity")
	}
	return ownerID, nil
}

func bearerToken(values []string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if strings.HasPrefix(strings.ToLower(v), "bearer ") {
			return strings.TrimSpace(v[7:])
		}
	}
	return ""
}

func ResolveOwnerID(ctx context.Context, requestOwnerID string) (string, error) {
	if authOwner, ok := OwnerIDFromContext(ctx); ok {
		if requestOwnerID != "" && requestOwnerID != authOwner {
			return "", errors.New("owner_id mismatch")
		}
		return authOwner, nil
	}
	return requestOwnerID, nil
}
