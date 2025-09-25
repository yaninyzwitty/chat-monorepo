package database

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/cassandra"
)

func TestCassandra(t *testing.T) {
	ctx := context.Background()
	cassandraContainer, err := cassandra.Run(context.Background(), "cassandra:4.1.3")
	// TODO-check how it works with this
	// cassandraContainer, err := cassandra.Run(ctx,
	// 	"cassandra:4.1.3",
	// 	cassandra.WithInitScripts(filepath.Join("testdata", "init.cql")),
	// 	cassandra.WithConfigFile(filepath.Join("testdata", "config.yaml")),
	// )

	require.NoError(t, err)

	// check container state
	state, err := cassandraContainer.State(ctx)
	require.NoError(t, err)
	require.True(t, state.Running)

	// get connection info
	connectionHost, err := cassandraContainer.Host(ctx)
	require.NoError(t, err)

	connectionPort, err := cassandraContainer.MappedPort(ctx, "9042")
	require.NoError(t, err)

	hostAndPort := fmt.Sprintf("%s:%d", connectionHost, connectionPort.Int())
	fmt.Println("ScyllaDB is available at", hostAndPort)

	session := DbConnect(ctx, nil, "", hostAndPort, connectionPort)
	defer session.Close()
	// Query
	var releaseVersion string
	err = session.Query(`SELECT release_version FROM system.local`).Scan(&releaseVersion)
	require.NoError(t, err)

	fmt.Println("ScyllaDB release version:", releaseVersion)

}
