//go:build e2e

package e2e_test

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/tdenkov123/file-metadata-service/internal/app"
	"github.com/tdenkov123/file-metadata-service/internal/config"
)

func startApp(t *testing.T, cfg *config.Config) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))

	application, err := app.New(ctx, cfg, logger)
	require.NoError(t, err)

	go func() {
		_ = application.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		application.Close()
	})

	addr := fmt.Sprintf("127.0.0.1:%d", cfg.GRPCPort)
	require.Eventually(t, func() bool {
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return false
		}
		defer conn.Close()
		health := grpc_health_v1.NewHealthClient(conn)
		resp, err := health.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
		return err == nil && resp.GetStatus() == grpc_health_v1.HealthCheckResponse_SERVING
	}, 30*time.Second, 500*time.Millisecond)
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func itoa(v int) string {
	return strconv.Itoa(v)
}
