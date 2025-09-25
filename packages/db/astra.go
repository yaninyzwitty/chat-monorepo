package database

import (
	"context"
	"time"

	gocqlastra "github.com/datastax/gocql-astra"
	"github.com/docker/go-connections/nat"
	"github.com/gocql/gocql"
	"github.com/yaninyzwitty/chat/packages/shared/config"
	"github.com/yaninyzwitty/chat/packages/shared/util"
)

func DbConnect(ctx context.Context, cfg *config.Config, token string, hostAndPort string, connectionPort nat.Port) *gocql.Session {
	if token != "" {
		cluster, err := gocqlastra.NewClusterFromBundle(
			cfg.DatabaseConfig.Path,
			cfg.DatabaseConfig.Username,
			token,
			30*time.Second, // bundle read timeout
		)
		util.Fail(err, "unable to load bundle")

		// set session timeout
		cluster.Timeout = time.Duration(cfg.DatabaseConfig.Timeout) * time.Second

		// create session
		session, err := gocql.NewSession(*cluster)
		util.Fail(err, "unable to connect to session %v", err)
		return session
	} else {
		// add this block to facilitate testing
		cluster := gocql.NewCluster(hostAndPort)
		cluster.Port = connectionPort.Int()
		cluster.Keyspace = "system" // default keyspace exists
		cluster.Consistency = gocql.Quorum
		cluster.Timeout = 30 * time.Second
		cluster.ConnectTimeout = 30 * time.Second

		// make the session
		session, err := cluster.CreateSession()
		util.Fail(err, "failed to create session")
		return session

	}

}
