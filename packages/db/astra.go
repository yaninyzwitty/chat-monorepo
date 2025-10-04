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

func ConnectLocal(host string, port int) *gocql.Session {
	if host == "" {
		host = LOCAL_HOST
	}
	cluster := gocql.NewCluster(host)
	cluster.Port = LOCAL_PORT
	cluster.Keyspace = "system" // system is alreasdy created will ease in testing
	cluster.Consistency = gocql.Quorum
	cluster.Timeout = 30 * time.Second
	cluster.ConnectTimeout = 30 * time.Second

	session, err := cluster.CreateSession()
	util.Fail(err, "failed to create local session")
	return session
}
