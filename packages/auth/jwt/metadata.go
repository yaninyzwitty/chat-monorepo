package jwt

import (
	"context"

	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	headerAuthorize = "authorization"
)

func AuthFromMD(ctx context.Context, expectedScheme string) (string, error) {
	vals := metadata.ValueFromIncomingContext(ctx, headerAuthorize)
	if len(vals) == 0 {
		return "", status.Error(codes.Unauthenticated, "Request unauthenticated with "+expectedScheme)
	}
	scheme, token, found := strings.Cut(vals[0], " ")
	if !found {
		return "", status.Error(codes.Unauthenticated, "Bad authorization string")
	}
	if !strings.EqualFold(scheme, expectedScheme) {
		return "", status.Error(codes.Unauthenticated, "Request unauthenticated with "+expectedScheme)
	}
	return token, nil
}
