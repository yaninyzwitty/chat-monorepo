package database

import (
	"time"

	gocqlastra "github.com/datastax/gocql-astra"
	"github.com/gocql/gocql"
	"github.com/yaninyzwitty/chat/packages/shared/config"
	"github.com/yaninyzwitty/chat/packages/shared/util"
)

const (
	LOCAL_PORT = 9042
	LOCAL_HOST = "127.0.0.1"
)

func ConnectAstra(cfg *config.Config, token string) *gocql.Session {
	// Astra DB
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

func ConnectLocal(host string) (*gocql.Session, error) {
	cluster := gocql.NewCluster(host)
	cluster.Keyspace = "init_sh_keyspace"
	cluster.Consistency = gocql.Quorum
	cluster.Timeout = 30 * time.Second
	cluster.ConnectTimeout = 30 * time.Second

	return cluster.CreateSession()
	// slog another commit
}
