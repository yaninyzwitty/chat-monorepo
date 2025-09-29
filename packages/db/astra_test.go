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
	require.NoError(t, err)

	// check the container state
	state, err := cassandraContainer.State(ctx)
	require.NoError(t, err)
	require.True(t, state.Running) // ensure its in running state

	// get connection info
	connectionHost, err := cassandraContainer.Host(ctx)
	require.NoError(t, err)

	connectionPort, err := cassandraContainer.MappedPort(ctx, "9042")
	require.NoError(t, err)

	// connect directly
	session := ConnectLocal(connectionHost, connectionPort.Int())
	defer session.Close()

	// query
	var releaseVersion string
	err = session.Query(`SELECT release_version FROM system.local`).Scan(&releaseVersion)
	require.NoError(t, err)

	fmt.Println("Cassandra release version:", releaseVersion)

}
