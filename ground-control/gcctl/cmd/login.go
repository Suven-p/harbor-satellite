package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/container-registry/harbor-satellite/ground-control/gcctl/internal/auth"
	"github.com/container-registry/harbor-satellite/ground-control/gcctl/internal/config"
)

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate against Ground Control and save the token",
	Long: `Authenticate against Ground Control and save the token for future use.
Credentials are read from --username/--password if provided, otherwise prompted.

Example:
  gcctl login --server http://localhost:8080 --username alice
`,
	RunE: runLogin,
}

func init() {
	rootCmd.AddCommand(loginCmd)

	loginCmd.Flags().String("server", "", "Ground Control server URL (e.g. http://localhost:8080)")
	loginCmd.Flags().String("username", "", "Username for authentication (prompted if not provided)")
	loginCmd.Flags().String("password", "", "Password for authentication (prompted if not provided)")
	_ = loginCmd.MarkFlagRequired("server")
}

func runLogin(cmd *cobra.Command, _ []string) error {
	server, _ := cmd.Flags().GetString("server")
	username, _ := cmd.Flags().GetString("username")
	password, _ := cmd.Flags().GetString("password")

	if username == "" {
		got, err := promptUsername(cmd.InOrStdin(), cmd.OutOrStdout())
		if err != nil {
			return err
		}
		username = got
	}
	if password == "" {
		got, err := promptPassword(cmd.OutOrStdout())
		if err != nil {
			return err
		}
		password = got
	}

	result, err := auth.Login(cmd.Context(), server, username, password)
	if err != nil {
		return err
	}

	appConfig = config.NewConfig(server, result.Token, result.ExpiresAt.String())
	if err := appConfigStore.Save(appConfig); err != nil {
		return fmt.Errorf("save token to %s: %w", appConfigStore.Path(), err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Logged in as %s and token has been saved\n", username)
	return nil
}

func promptUsername(in io.Reader, out io.Writer) (string, error) {
	fmt.Fprint(out, "Username: ")
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read username: %w", err)
	}
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return "", errors.New("username cannot be empty")
	}
	return line, nil
}

func promptPassword(out io.Writer) (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return "", errors.New("password not provided and stdin is not a terminal; pass --password")
	}
	fmt.Fprint(out, "Password: ")
	pw, err := term.ReadPassword(fd)
	fmt.Fprintln(out)
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	if len(pw) == 0 {
		return "", errors.New("password cannot be empty")
	}
	return string(pw), nil
}
