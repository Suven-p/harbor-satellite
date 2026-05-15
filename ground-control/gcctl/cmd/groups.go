package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/spf13/cobra"

	"github.com/container-registry/harbor-satellite/ground-control/gcctl/apiclient/generated/client/groups"
	"github.com/container-registry/harbor-satellite/ground-control/gcctl/apiclient/generated/models"
	"github.com/container-registry/harbor-satellite/ground-control/gcctl/internal/apiclient"
)

// groupsCmd is the parent command for group operations.
// New subcommands (list, get, delete, etc.) are added here alongside `sync`.
var groupsCmd = &cobra.Command{
	Use:   "groups",
	Short: "Manage Ground Control groups",
}

// groupsSyncCmd represents the `groups sync` command.
var groupsSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Create or update a group from a state artifact",
	Long: `Sends a state artifact to Ground Control's POST /api/groups/sync.
The request body is read from --file (use "-" to read from stdin).

Example:
  gcctl groups sync --file state.json
  cat state.json | gcctl groups sync -f -
`,
	RunE: runGroupsSync,
}

func init() {
	rootCmd.AddCommand(groupsCmd)
	groupsCmd.AddCommand(groupsSyncCmd)

	groupsSyncCmd.Flags().StringP("file", "f", "-", `Path to the state artifact JSON ("-" reads from stdin)`)
}

func runGroupsSync(cmd *cobra.Command, _ []string) error {
	if appConfig == nil || !appConfig.IsLoggedIn() {
		return errors.New("not logged in: run 'gcctl login' first")
	}

	path, _ := cmd.Flags().GetString("file")
	state, err := readStateArtifact(path, cmd.InOrStdin())
	if err != nil {
		return err
	}

	gc, err := apiclient.New(appConfig.Server)
	if err != nil {
		return err
	}

	params := groups.NewSyncGroupParamsWithContext(cmd.Context()).WithState(state)
	authInfo := httptransport.BearerToken(appConfig.Token)

	ok, err := gc.Groups.SyncGroup(params, authInfo)
	if err != nil {
		return translateSyncGroupError(err)
	}

	return writeJSON(cmd.OutOrStdout(), ok.Payload)
}

func readStateArtifact(path string, stdin io.Reader) (*models.StateArtifact, error) {
	var r io.Reader
	if path == "-" {
		r = stdin
	} else {
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open state file: %w", err)
		}
		defer f.Close()
		r = f
	}

	var state models.StateArtifact
	if err := json.NewDecoder(r).Decode(&state); err != nil {
		return nil, fmt.Errorf("decode state artifact: %w", err)
	}
	return &state, nil
}

func translateSyncGroupError(err error) error {
	if _, ok := errors.AsType[*groups.SyncGroupUnauthorized](err); ok {
		return errors.New("unauthorized: token rejected; run 'gcctl login' again")
	}
	if badRequest, ok := errors.AsType[*groups.SyncGroupBadRequest](err); ok {
		return fmt.Errorf("invalid request: %s", apiErrorMessage(badRequest.Payload))
	}
	if badGateway, ok := errors.AsType[*groups.SyncGroupBadGateway](err); ok {
		return fmt.Errorf("upstream Harbor error: %s", apiErrorMessage(badGateway.Payload))
	}
	if serverErr, ok := errors.AsType[*groups.SyncGroupInternalServerError](err); ok {
		return fmt.Errorf("ground control returned 500: %s", apiErrorMessage(serverErr.Payload))
	}
	return err
}

func apiErrorMessage(e *models.ErrorResponse) string {
	if e == nil || e.Error == nil {
		return "no error message in response"
	}
	return *e.Error
}

func writeJSON(out io.Writer, v any) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
