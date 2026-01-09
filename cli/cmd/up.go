package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/simukka/sqlcode/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	upCmd = &cobra.Command{
		Use:   "up <dbname>:<schemasuffix>",
		Short: "Uploads the SQL code to the SQL database configured in sqlcode.yaml",
		Long:  "Uploads the SQL code to an SQL database. The target is specified using a :-delimiter, with the database name first and schemasuffix last",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := logrus.StandardLogger()
			ctx := context.Background()

			if len(args) != 1 {
				_ = cmd.Help()
				return errors.New("Wrong number of arguments")
			}
			target := args[0]
			targetParts := strings.Split(target, ":")
			if len(targetParts) != 2 {
				if len(args) != 1 {
					_ = cmd.Help()
					return errors.New("Illegal target, should contain exactly one ':'")
				}
			}
			dbname := targetParts[0]
			schemasuffix := targetParts[1]

			config, err := LoadConfig()
			if err != nil {
				return err
			}

			dbconfig, ok := config.Databases[dbname]
			if !ok {
				return errors.New(fmt.Sprintf("database %s not present in configuration file", dbname))
			}

			dbc, err := dbconfig.Open(ctx, logger)
			if err != nil {
				return err
			}

			exists, err := sqlcode.Exists(ctx, dbc, schemasuffix)
			if err != nil {
				return err
			}
			if exists {
				fmt.Println(fmt.Sprintf("Schema [%s] already exists, removing", sqlcode.SchemaName(schemasuffix)))
				if err := sqlcode.Drop(ctx, dbc, schemasuffix); err != nil {
					return err
				}
			}

			deployable, err := dep(false)
			if err != nil {
				return err
			}
			deployable = deployable.WithSchemaSuffix(schemasuffix)
			err = deployable.DropAndUpload(ctx, dbc)
			if err != nil {
				return err
			}
			fmt.Println(fmt.Sprintf("Schema [%s] successfully uploaded", sqlcode.SchemaName(schemasuffix)))

			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(upCmd)
}
