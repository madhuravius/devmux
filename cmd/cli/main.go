package main

import (
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"

	"devmux/internal/cli"
)

func main() {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		Level:           log.InfoLevel,
		ReportTimestamp: true,
		ReportCaller:    false,
	})

	rootCmd := &cobra.Command{
		Use:   "devmux",
		Short: "Docker development environment manager",
		Long:  `A CLI tool to help manage Docker-based development environments.`,
	}

	healthCmd := &cobra.Command{
		Use:   "health",
		Short: "Check Docker daemon and container health",
		Run: func(cmd *cobra.Command, args []string) {
			app := cli.NewHealthApp(nil)
			if err := app.Run(); err != nil {
				logger.Error("Error running health check", "error", err)
				os.Exit(1)
			}
		},
	}

	rootCmd.AddCommand(healthCmd)

	logger.Info("Starting application")

	if err := rootCmd.Execute(); err != nil {
		logger.Error("Command execution failed", "error", err)
		os.Exit(1)
	}

	logger.Info("Application completed")
}
