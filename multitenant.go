package goat

import (
	"database/sql"
	"log"
)

// Tenant describes a tenant database configuration
type Tenant struct {
	URL             string   `yaml:"url,omitempty"`
	MaxConnections  int      `yaml:"max-connections,omitempty"`
	IdleConnections int      `yaml:"idle-connections,omitempty"`
	Domains         []string `yaml:"domains,omitempty"`
}

// Multitenant describes the multitenant configuration
type Multitenant struct {
	Tenants []Tenant `yaml:"tenants,omitempty"`
}

// MDB host multitenant database connection and related domain mapping
type MDB struct {
	conns   []*sql.DB
	domains map[string]*sql.DB
}

// OpenTenants open all tenant database connections
func OpenTenants(tenants []Tenant) (*MDB, error) {
	mdb := &MDB{}
	mdb.conns = make([]*sql.DB, 0, len(tenants))
	mdb.domains = make(map[string]*sql.DB)
	for _, t := range tenants {
		db, err := openSQL(EvaluateEnv(t.URL), t.MaxConnections, t.IdleConnections)
		if err != nil {
			return mdb, err
		}

		mdb.conns = append(mdb.conns, db)
		for _, d := range t.Domains {
			mdb.domains[d] = db
		}
	}
	return mdb, nil
}

func openSQL(dburl string, maxconn, maxidle int) (*sql.DB, error) {
	db, err := sql.Open("postgres", dburl)
	if err == nil {
		if maxconn > 0 {
			db.SetMaxOpenConns(maxconn)
		}

		if maxidle > 0 {
			db.SetMaxIdleConns(maxidle)
		}

		err = db.Ping()
		if err != nil {
			log.Println(err)
		}
	}
	return db, err
}

// Close terminates any connection stored in the multitenant database structure
func (mdb *MDB) Close() error {
	for _, db := range mdb.conns {
		err := db.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// Get find and return the db connection for the provider domain
func (mdb *MDB) Get(domain string) *sql.DB {
	db, ok := mdb.domains[domain]
	if !ok {
		db = mdb.domains["default"]
	}
	return db
}
