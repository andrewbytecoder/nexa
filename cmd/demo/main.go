package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var httpMethod string
var childHttpMethod string

var rootCmd = &cobra.Command{
	Use: "app",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		fmt.Println("✅ rootCmd.PersistentPreRun executed!")
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Running root command")
		fmt.Println("HTTP Method:", httpMethod)
	},
}

var childCmd = &cobra.Command{
	Use: "child",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Running child command")
		fmt.Println("HTTP childHttpMethod:", childHttpMethod)
		fmt.Println("HTTP Method:", httpMethod)
	},
}

func init() {
	rootCmd.AddCommand(childCmd)
	rootCmd.Flags().StringVarP(&httpMethod, "request", "X", "GET", "HTTP method to use")
	childCmd.Flags().StringVarP(&childHttpMethod, "request", "X", "GET", "HTTP method to use")

}

func main() {
	rootCmd.Execute()
}
