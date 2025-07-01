package cmd

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type node struct {
	symbol       string
	fileName     string
	lineToString map[int]string
	declaration  bool
	parent       *node
	children     []*node
}

func parseSql(content string, nodes []*node) error {
	return nil
}

var (
	findCmd = &cobra.Command{
		Use:   "find",
		Short: "TODO",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 && len(args) != 1 {
				_ = cmd.Help()
				return errors.New("Too many arguments")
			}

			dir, err := os.Getwd()
			if err != nil {
				return err
			}

			if len(args) != 0 {
				dir = args[1]
			}

			nodes := []*node{}

			// Walk the directory recursively
			err = filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
				if err != nil {
					return err
				}
				// Check if it's a regular file and ends with .sql
				if !info.IsDir() && strings.HasSuffix(info.Name(), ".sql") {
					contentBytes, err := os.ReadFile(path)
					if err != nil {
						return err
					}
					content := string(contentBytes)
					if strings.Contains(content, "[code]") {
						err := parseSql(content, nodes)
						if err != nil {
							return err // TODO: multiple errors
						}
					}
				}
				return nil
			})

			if err != nil {
				return err
			}

			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(findCmd)
}
