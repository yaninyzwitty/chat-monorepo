package handler_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/gocql/gocql"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/cassandra"
)

var (
	session   *gocql.Session
	parallel  = false
	cqlScript = "../../db/testdata/init.cql"
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Start Cassandra container
	cassandraContainer, err := cassandra.Run(ctx, "cassandra:4.1.3",
		cassandra.WithInitScripts(cqlScript),
	)
	if err != nil {
		slog.Error("failed to create cassandra instance", "error", err)
		os.Exit(1)
	}

	// Cleanup on exit
	defer func() {
		if err := testcontainers.TerminateContainer(cassandraContainer); err != nil {
			slog.Error("failed to terminate container", "error", err)
		}
	}()

	// Get connection info
	connectionHost, err := cassandraContainer.ConnectionHost(ctx)
	if err != nil {
		slog.Error("failed to get connection host", "error", err)
		os.Exit(1)
	}

	port, err := cassandraContainer.MappedPort(ctx, "9042/tcp")
	if err != nil {
		slog.Error("failed to get mapped port", "error", err)
		os.Exit(1)
	}

	// Connect to Cassandra via gocql
	cluster := gocql.NewCluster(connectionHost)
	cluster.Port = port.Int()
	cluster.Keyspace = "chat" // make sure this matches your init.cql
	cluster.Consistency = gocql.Quorum
	cluster.Timeout = 30 * time.Second
	cluster.ConnectTimeout = 30 * time.Second

	session, err = cluster.CreateSession()
	if err != nil {
		slog.Error("failed to connect to cassandra", "error", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Close Cassandra session
	session.Close()

	os.Exit(code)
}

func Cleanup() {
	// TODO -- check of
	if err := session.Query("TRUNCATE chat.users").Exec(); err != nil {
		slog.Warn("cleanup failed", "error", err)
	}
}

func CheckParallel(t *testing.T) {
	if parallel {
		t.Parallel()
	}
}

func GetConnection(ctx context.Context) (*gocql.Session, error) {
	return session, nil
}
