package database

import (
	"context"
	"time"

	gocqlastra "github.com/datastax/gocql-astra"
	"github.com/gocql/gocql"
	"github.com/yaninyzwitty/chat/pkg/config"
	"github.com/yaninyzwitty/chat/pkg/util"
)

func DbConnect(ctx context.Context, cfg *config.Config, token string) *gocql.Session {
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
}
