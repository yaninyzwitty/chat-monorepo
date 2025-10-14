package handler_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/gocql/gocql"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/cassandra"
	database "github.com/yaninyzwitty/chat/packages/db"
)

var connectionHost = ""

// var parrallel = true

func TestMain(m *testing.M) {
	ctx := context.Background()

	root := filepath.Join("..") // go back to module root (packages/user)
	testdataPath := filepath.Join(root, "testdata", "init.sh")
	cassandraContainer, err := cassandra.Run(ctx, "cassandra:4.1.3", cassandra.WithInitScripts(filepath.Dir(testdataPath), "init.sh"))

	defer func() {
		if err := testcontainers.TerminateContainer(cassandraContainer); err != nil {
			slog.Error("failed to terminate container", "error", err)
			os.Exit(1)
		}

	}()

	if err != nil {
		slog.Error("failed to load container", "error", err)
		os.Exit(1)
	}

	connectionHost, err = cassandraContainer.ConnectionHost(ctx)
	if err != nil {
		slog.Error("failed to get connection host", "error", err)
		os.Exit(1)
	}

	res := m.Run()
	os.Exit(res)

}

func getConn() (*gocql.Session, error) {
	return database.ConnectLocal(connectionHost)
}

// TODO- remove unused methods

// func cleanup() {
// 	session, err := database.ConnectLocal(connectionHost)
// 	if err != nil {
// 		return
// 	}
// 	defer session.Close()

// 	session.Query("DELETE FROM init_sh_keyspace.test_table WHERE id = 1")
// }

// func checkParallel(t *testing.T) {
// 	if parrallel {
// 		t.Parallel()
// 	}
// }
