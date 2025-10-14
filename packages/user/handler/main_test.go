package handler_test

import (
	"context"
	"log/slog"
	"os"
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

	wd, _ := os.Getwd()

	cassandraContainer, err := cassandra.Run(
		ctx,
		"cassandra:4.1.3",
		cassandra.WithInitScripts(wd, "init.sh"),
	)

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
