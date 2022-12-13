package cmdcfg

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var (
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version of libgenerator",
		Long:  `version prints out the version of the libgenerator binary.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("libgenerator version %s\n", Version)
			return nil
		},
	}
)
