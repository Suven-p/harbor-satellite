package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/container-registry/harbor-satellite/ground-control/gcctl/apiclient/generated/client/system"
	"github.com/container-registry/harbor-satellite/ground-control/gcctl/apiclient/generated/models"
	"github.com/container-registry/harbor-satellite/ground-control/gcctl/internal/apiclient"
)

// healthCmd represents the health command
var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check whether Ground Control is healthy",
	Long: `Issues a GET /health against Ground Control and prints the status.

The server URL is taken from --server if set; otherwise it falls back to the
server saved in the gcctl config (populated by 'gcctl login'). Exits non-zero
if the service reports unhealthy or cannot be reached.
`,
	RunE: runHealth,
}

func init() {
	rootCmd.AddCommand(healthCmd)

	healthCmd.Flags().String("server", "", "Ground Control server URL (overrides the saved config)")
}

func runHealth(cmd *cobra.Command, _ []string) error {
	serverURL, _ := cmd.Flags().GetString("server")
	if serverURL == "" {
		serverURL = appConfig.Server
	}

	if serverURL == "" {
		return errors.New("server URL must be provided via --server or saved in config")
	}

	gc, err := apiclient.New(serverURL)
	if err != nil {
		return err
	}

	ok, err := gc.System.GetHealth(system.NewGetHealthParamsWithContext(cmd.Context()))
	if err != nil {
		if unhealthy, ok := errors.AsType[*system.GetHealthServiceUnavailable](err); ok {
			printHealth(cmd, unhealthy.Payload)
			return errors.New("service is unhealthy")
		}
		return err
	}
	printHealth(cmd, ok.Payload)
	return nil
}

func printHealth(cmd *cobra.Command, payload *models.HealthResponse) {
	status := "unknown"
	if payload != nil && payload.Status != nil {
		status = *payload.Status
	}
	fmt.Fprintf(cmd.OutOrStdout(), "status: %s\n", status)
}
