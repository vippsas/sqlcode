package sqltest

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	mssql "github.com/microsoft/go-mssqldb"
	"github.com/microsoft/go-mssqldb/msdsn"
)

type StdoutLogger struct {
}

func (s StdoutLogger) Printf(format string, v ...interface{}) {
	fmt.Printf(format, v...)
}

func (s StdoutLogger) Println(v ...interface{}) {
	fmt.Println(v...)
}

var _ mssql.Logger = StdoutLogger{}

type Fixture struct {
	DB      *sql.DB
	DBName  string
	adminDB *sql.DB
}

func NewFixture() *Fixture {
	var fixture Fixture

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	dsn := os.Getenv("SQLSERVER_DSN")
	if dsn == "" {
		panic("Must set SQLSERVER_DSN to run tests")
	}
	dsn = dsn + "&log=3"

	mssql.SetLogger(StdoutLogger{})

	var err error

	fixture.adminDB, err = sql.Open("sqlserver", dsn)
	if err != nil {
		panic(err)
	}
	fixture.DBName = strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")

	_, err = fixture.adminDB.ExecContext(ctx, fmt.Sprintf(`create database [%s]`, fixture.DBName))
	if err != nil {
		panic(err)
	}
	// These settings are just to get "worst-case" for our tests, since snapshot could interfer
	_, err = fixture.adminDB.ExecContext(ctx, fmt.Sprintf(`alter database [%s] set allow_snapshot_isolation on`, fixture.DBName))
	if err != nil {
		panic(err)
	}
	_, err = fixture.adminDB.ExecContext(ctx, fmt.Sprintf(`alter database [%s] set read_committed_snapshot on`, fixture.DBName))
	if err != nil {
		panic(err)
	}

	pdsn, err := msdsn.Parse(dsn)
	if err != nil {
		panic(err)
	}
	pdsn.Database = fixture.DBName

	fixture.DB, err = sql.Open("sqlserver", pdsn.URL().String())
	if err != nil {
		panic(err)
	}

	return &fixture
}

func (f *Fixture) Teardown() {
	if f.adminDB == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	_ = f.DB.Close()
	f.DB = nil
	_, _ = f.adminDB.ExecContext(ctx, fmt.Sprintf(`drop database [%s]`, f.DBName))
	_ = f.adminDB.Close()
	f.adminDB = nil
}

func (f *Fixture) RunMigrations() {
	migrationSql, err := ioutil.ReadFile("migrations/from0001/0001.changefeed.sql")
	if err != nil {
		panic(err)
	}
	parts := strings.Split(string(migrationSql), "\ngo\n")
	for _, p := range parts {
		_, err = f.DB.Exec(p)
		if err != nil {
			fmt.Println(p)
			panic(err)
		}
	}
}

func (f *Fixture) RunMigrationFile(filename string) {
	migrationSql, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	parts := strings.Split(string(migrationSql), "\ngo\n")
	for _, p := range parts {
		_, err = f.DB.Exec(p)
		if err != nil {
			fmt.Println(p)
			panic(err)
		}
	}
}
