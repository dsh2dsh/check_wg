package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var rootCmd = cobra.Command{
	Use:   "check_wg",
	Short: "Icinga2 health check of wireguard peers, using output of wg(8).",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Don't show usage on app errors.
		// https://github.com/spf13/cobra/issues/340#issuecomment-378726225
		cmd.SilenceUsage = true
	},
}

func init() {
	rootCmd.AddCommand(&handshakeCmd)
}

func Execute() error {
	return rootCmd.Execute() //nolint:wrapcheck // main() doesn't need it
}

func withWgCmd(args []string, fn func(r io.Reader) error) error {
	r, cmd, err := startWgCmd(args)
	if err != nil {
		return err
	}

	if err := fn(r); err != nil {
		return err
	}

	if cmd != nil {
		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("wait for %v: %w", args, err)
		}
	}
	return nil
}

func startWgCmd(args []string) (io.Reader, *exec.Cmd, error) {
	if len(args) == 0 {
		return os.Stdin, nil, nil
	}

	var cmdArgs []string
	if len(args) > 1 {
		cmdArgs = args[1:]
	}

	cmd := exec.Command(args[0], cmdArgs...)
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	const errMsg = "exec %v: %w"
	if err != nil {
		return nil, nil, fmt.Errorf(errMsg, args, err)
	} else if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf(errMsg, args, err)
	}
	return stdout, cmd, nil
}
