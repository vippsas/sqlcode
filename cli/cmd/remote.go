package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var (
	remoteCmd = &cobra.Command{
		Use:   "remote",
		Short: "Lists databases listed in sqlcode.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := LoadConfig()
			if err != nil {
				return err
			}
			for k := range cfg.Databases {
				fmt.Println(k)
			}
			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(remoteCmd)
}
