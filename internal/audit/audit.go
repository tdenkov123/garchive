package audit

import (
	"context"
	"log/slog"

	"google.golang.org/grpc/peer"
)

type Logger struct {
	log *slog.Logger
}

func New(log *slog.Logger) *Logger {
	if log == nil {
		log = slog.Default()
	}
	return &Logger{log: log.With("component", "audit")}
}

func (l *Logger) Log(ctx context.Context, event string, attrs ...any) {
	args := append([]any{"event", event, "client_ip", clientIP(ctx)}, attrs...)
	l.log.InfoContext(ctx, event, args...)
}

func clientIP(ctx context.Context) string {
	if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
		return p.Addr.String()
	}
	return "unknown"
}
