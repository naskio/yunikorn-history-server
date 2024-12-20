package health

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/G-Research/unicorn-history-server/internal/config"
	"github.com/G-Research/unicorn-history-server/internal/database/postgres"
	"github.com/G-Research/unicorn-history-server/internal/yunikorn"
	testconfig "github.com/G-Research/unicorn-history-server/test/config"
)

func TestService_Readiness_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode.")
	}

	ctx := context.Background()

	now := time.Now()
	version := "1.0.0"

	t.Run("status is unhealthy when one component is unhealthy", func(t *testing.T) {
		invalidYunikornConfig := config.YunikornConfig{
			Host:   "invalid-host",
			Port:   2212,
			Secure: false,
		}
		yunikornClient := yunikorn.NewRESTClient(&invalidYunikornConfig)
		postgresPool, err := postgres.NewConnectionPool(ctx, testconfig.GetTestPostgresConfig())
		if err != nil {
			t.Fatalf("error creating postgres connection pool: %v", err)
		}
		components := []Component{
			NewYunikornComponent(yunikornClient),
			NewPostgresComponent(postgresPool),
		}
		service := Service{
			startedAt:  now,
			version:    version,
			components: components,
		}
		status := service.Readiness(context.Background())
		expectErrorPrefix := `Get "http://invalid-host:2212/ws/v1/scheduler/healthcheck": dial tcp: lookup invalid-host`
		assert.False(t, status.Healthy)
		assert.Equal(t, 2, len(status.ComponentStatuses))
		assertStatus(t, status.ComponentStatuses, "yunikorn", false, expectErrorPrefix)
		assert.Equal(t, now, status.StartedAt)
		assert.Equal(t, version, status.Version)
	})

	t.Run("status is healthy when all components are healthy", func(t *testing.T) {
		yunikornClient := yunikorn.NewRESTClient(testconfig.GetTestYunikornConfig())
		postgresPool, err := postgres.NewConnectionPool(ctx, testconfig.GetTestPostgresConfig())
		if err != nil {
			t.Fatalf("error creating postgres connection pool: %v", err)
		}
		components := []Component{
			NewYunikornComponent(yunikornClient),
			NewPostgresComponent(postgresPool),
		}
		service := Service{
			startedAt:  now,
			version:    version,
			components: components,
		}
		status := service.Readiness(context.Background())
		assert.True(t, status.Healthy)
		for _, componentStatus := range status.ComponentStatuses {
			assert.True(t, componentStatus.Healthy)
		}
		assert.Equal(t, now, status.StartedAt)
		assert.Equal(t, version, status.Version)
	})
}

func assertStatus(t *testing.T, statuses []*ComponentStatus, identifier string, expectedHealthy bool, expectedErrorPrefix string) {
	for _, status := range statuses {
		if status.Identifier == identifier {
			assert.Equal(t, expectedHealthy, status.Healthy)
			assert.Contains(t, status.Error, expectedErrorPrefix)
			return
		}
	}
	t.Fatalf("component with identifier %s not found", identifier)
}
