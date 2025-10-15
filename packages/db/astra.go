package database

import (
	"fmt"
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
	cluster.Keyspace = "init_sh_keyspace" // ✅ Set correct keyspace
	cluster.Consistency = gocql.Quorum
	cluster.Timeout = 30 * time.Second
	cluster.ConnectTimeout = 30 * time.Second

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %v", err)
	}

	queries := []string{
		`CREATE KEYSPACE IF NOT EXISTS init_sh_keyspace 
		 WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}`,
		// ✅ No `USE` needed, session already bound to keyspace
		`CREATE TABLE IF NOT EXISTS init_sh_keyspace.users (
			id UUID PRIMARY KEY,
			name TEXT,
			alias_name TEXT,
			created_at TIMESTAMP,
			updated_at TIMESTAMP,
			email TEXT,
			password TEXT
		)`,
	}

	for _, query := range queries {
		if err := session.Query(query).Exec(); err != nil {
			return nil, fmt.Errorf("failed to execute query %q: %v", query, err)
		}
	}

	return session, nil
}
