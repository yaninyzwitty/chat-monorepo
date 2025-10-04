package database

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/gocql/gocql"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/cassandra"
)

func TestCassandra(t *testing.T) {
	ctx := context.Background()
	cassandraContainer, err := cassandra.Run(context.Background(), "cassandra:4.1.3")
	require.NoError(t, err)

	// Cleanup on exit
	defer func() {
		if err := testcontainers.TerminateContainer(cassandraContainer); err != nil {
			slog.Error("failed to terminate container", "error", err)
		}
	}()

	connectionHost, err := cassandraContainer.ConnectionHost(ctx)
	require.NoError(t, err)

	// Connect to Cassandra via gocql
	cluster := gocql.NewCluster(connectionHost)
	cluster.Keyspace = "system" // make sure this matches your init.cql
	cluster.Consistency = gocql.Quorum
	cluster.Timeout = 30 * time.Second
	cluster.ConnectTimeout = 30 * time.Second
	session, err := cluster.CreateSession()
	require.NoError(t, err)

	defer session.Close()

	// query
	var releaseVersion string
	err = session.Query(`SELECT release_version FROM system.local`).Scan(&releaseVersion)
	require.NoError(t, err)

	fmt.Println("Cassandra release version:", releaseVersion)

}
