package cmd

import (
	"context"
	"database/sql"
	"fmt"
	mssql "github.com/microsoft/go-mssqldb"
	"github.com/microsoft/go-mssqldb/azuread"
	"golang.org/x/net/proxy"
	"io/ioutil"
	"os"
	"path"
	"strings"

	_ "github.com/microsoft/go-mssqldb/azuread"
	"github.com/microsoft/go-mssqldb/msdsn"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type DatabaseConfig struct {
	Connection       string `yaml:"connection"`
	Dsn              msdsn.Config
	UsePasswordLogin bool
}

func OpenSocks5Sql(dsn string) (*sql.DB, error) {
	var err error
	var connector *mssql.Connector

	if strings.HasPrefix(dsn, "azuresql://") {
		connector, err = azuread.NewConnector(dsn)
		if err != nil {
			return nil, err
		}
	} else if strings.HasPrefix(dsn, "sqlserver://") {
		//dbi, err := sql.Open("sqlserver", dsn)
		connector, err = mssql.NewConnector(dsn)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("expected URI-style dsn; sqlserver:// for password login or azuresql:// for AD login")
	}

	socksProxyAddress := os.Getenv("SQL_SOCKS")
	if socksProxyAddress != "" {
		dialer, err := proxy.SOCKS5("tcp", socksProxyAddress, nil, nil)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("Could not connect with SOCKS5 to %s", socksProxyAddress))
		}
		connector.Dialer = dialer.(proxy.ContextDialer)
	}

	return sql.OpenDB(connector), nil
}

func (dbcfg DatabaseConfig) Open(ctx context.Context, logger logrus.FieldLogger) (*sql.DB, error) {
	return OpenSocks5Sql(dbcfg.Connection)
}

type Config struct {
	Databases   map[string]DatabaseConfig `yaml:"databases"`
	ServiceName string                    `yaml:"servicename"`
}

func LoadConfig() (Config, error) {
	var result Config

	configFilename := path.Join(directory, "sqlcode.yaml")
	if _, err := os.Stat(configFilename); os.IsNotExist(err) {
		return Config{}, errors.New("No sqlcode.yaml found in current directory")
	}

	yamlFile, err := ioutil.ReadFile(configFilename)
	if err != nil {
		return Config{}, err
	}
	err = yaml.Unmarshal(yamlFile, &result)
	if err != nil {
		return Config{}, err
	}

	for key, dbcfg := range result.Databases {
		if err != nil {
			return Config{}, err
		}
		result.Databases[key] = dbcfg
	}
	return result, nil
}
