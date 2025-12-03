package sqltest

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	mssql "github.com/denisenkom/go-mssqldb"
	"github.com/denisenkom/go-mssqldb/msdsn"
	"github.com/gofrs/uuid"
	_ "github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type SqlDriverType int

const (
	SqlDriverDenisen SqlDriverType = iota
	SqlDriverPgx
)

var sqlDrivers = map[SqlDriverType]string{
	SqlDriverDenisen: "sqlserver",
	SqlDriverPgx:     "pgx",
}

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
	Driver  SqlDriverType
}

func (f *Fixture) IsSqlServer() bool {
	return f.Driver == SqlDriverDenisen
}

func (f *Fixture) IsPostgresql() bool {
	return f.Driver == SqlDriverPgx
}

// SQL specific quoting syntax
func (f *Fixture) Quote(value string) string {
	if f.IsSqlServer() {
		return fmt.Sprintf("[%s]", value)
	}
	if f.IsPostgresql() {
		return fmt.Sprintf(`"%s"`, value)
	}
	return value
}

func NewFixture() *Fixture {
	var fixture Fixture

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	dsn := os.Getenv("SQLSERVER_DSN")
	if len(dsn) == 0 {
		panic("Must set SQLSERVER_DSN to run tests")
	}

	if strings.Contains(dsn, "sqlserver") {
		// set the logging level
		// To enable specific logging levels, you sum the values of the desired flags
		// 1: Log errors
		// 2: Log messages
		// 4: Log rows affected
		// 8: Trace SQL statements
		// 16: Log statement parameters
		// 32: Log transaction begin/end
		dsn = dsn + "&log=63"
		mssql.SetLogger(StdoutLogger{})
		fixture.Driver = SqlDriverDenisen
	}
	if strings.Contains(dsn, "postgresql") {
		fixture.Driver = SqlDriverPgx
		// https://www.postgresql.org/docs/current/runtime-config-client.html#GUC-CLIENT-MIN-MESSAGES
		dsn = dsn + "&options=-c%20client_min_messages%3DDEBUG5"
	}

	var err error
	fixture.adminDB, err = sql.Open(sqlDrivers[fixture.Driver], dsn)
	if err != nil {
		panic(err)
	}

	fixture.DBName = strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	dbname := fixture.Quote(fixture.DBName)
	qs := fmt.Sprintf(`create database %s`, dbname)
	_, err = fixture.adminDB.ExecContext(ctx, qs)
	if err != nil {
		fmt.Printf("Failed to create the (%s) database: %s: %e\n", sqlDrivers[fixture.Driver], dbname, err)
		panic(err)
	}

	if fixture.IsSqlServer() {
		// These settings are just to get "worst-case" for our tests, since snapshot could interfer
		_, err = fixture.adminDB.ExecContext(ctx, fmt.Sprintf(`alter database %s set allow_snapshot_isolation on`, dbname))
		if err != nil {
			panic(err)
		}
		_, err = fixture.adminDB.ExecContext(ctx, fmt.Sprintf(`alter database %s set read_committed_snapshot on`, dbname))
		if err != nil {
			panic(err)
		}

		pdsn, _, err := msdsn.Parse(dsn)
		if err != nil {
			panic(err)
		}
		pdsn.Database = fixture.DBName

		fixture.DB, err = sql.Open(sqlDrivers[fixture.Driver], pdsn.URL().String())
		if err != nil {
			panic(err)
		}
	}

	if fixture.IsPostgresql() {
		// TODO use pgx config parser
		fixture.DB, err = sql.Open(sqlDrivers[fixture.Driver], strings.ReplaceAll(dsn, "/master", "/"+fixture.DBName))
		if err != nil {
			panic(err)
		}
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
	_, err := f.adminDB.ExecContext(ctx, fmt.Sprintf(`drop database %s`, f.Quote(f.DBName)))
	if err != nil {
		fmt.Printf("Failed to drop (%s) database %s: %e", sqlDrivers[f.Driver], f.DBName, err)
	}
	_ = f.adminDB.Close()
	f.adminDB = nil
}

func (f *Fixture) RunMigrationFile(filename string) {
	migrationSql, err := os.ReadFile(filename)
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
