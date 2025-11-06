package sqltest

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	mssql "github.com/denisenkom/go-mssqldb"
	"github.com/denisenkom/go-mssqldb/msdsn"
	"github.com/gofrs/uuid"
	pgsql "github.com/lib/pq"
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
	Driver  driver.Driver
}

func (f *Fixture) Quote(value string) string {
	var ms mssql.Driver
	var pg pgsql.Driver

	if f.Driver == &ms {
		return fmt.Sprintf("[%s]", value)
	}
	if f.Driver == &pg {
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

	driver := os.Getenv("SQLSERVER_DRIVER")
	if len(driver) == 0 {
		panic("Must set SQLSERVER_DRIVER to run tests")
	}

	switch driver {
	case "sqlserver":
		// set the logging level
		dsn = dsn + "&log=3"
		mssql.SetLogger(StdoutLogger{})
	case "postgres":
		break
	}

	var err error

	fixture.adminDB, err = sql.Open(driver, dsn)
	if err != nil {
		panic(err)
	}
	// store a reference to the type of sql driver
	fixture.Driver = fixture.adminDB.Driver()

	fixture.DBName = strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	dbname := fixture.Quote(fixture.DBName)
	_, err = fixture.adminDB.ExecContext(ctx, fmt.Sprintf(`create database %s`, dbname))
	if err != nil {
		fmt.Printf("Failed to create the database: %s for the %s driver\n", dbname, driver)
		panic(err)
	}

	if driver == "sqlserver" {
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

		fixture.DB, err = sql.Open(driver, pdsn.URL().String())
		if err != nil {
			panic(err)
		}
	}

	if driver == "postgres" {
		// TODO
		fixture.DB, err = sql.Open(driver, strings.ReplaceAll(dsn, "/master", "/"+fixture.DBName))
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
