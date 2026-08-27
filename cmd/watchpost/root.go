package main

import (
	"context"
	"fmt"

	"github.com/branden-thompson/watchpost/platform/invariant"

	"github.com/spf13/cobra"

	"github.com/branden-thompson/watchpost/app"
	"github.com/branden-thompson/watchpost/modes/report"
	"github.com/branden-thompson/watchpost/pkg/schema"
	"github.com/branden-thompson/watchpost/platform/httpx"
	"github.com/branden-thompson/watchpost/platform/snapshot"
	"github.com/branden-thompson/watchpost/platform/term"
)

// newRootCmd builds the cobra tree (T-L: everything under `watchpost`).
func newRootCmd() *cobra.Command { return newRootCmdWith(app.ReportOnceWithStats) }

// newRootCmdWith builds the tree over the given report fetch.
func newRootCmdWith(reportOnce reportFunc) *cobra.Command {
	var ascii bool
	root := &cobra.Command{
		Use:           "watchpost",
		Short:         "Terminal-native live weather station",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunDashboard(version, app.Options{ASCII: ascii})
		},
	}
	// --ascii swaps the row marks and legend for ASCII forms (R-12a; the B6
	// promise, wired by the quality pass Q3 — A11-10). Persistent so
	// `watchpost setup --ascii` reads the same way.
	root.PersistentFlags().BoolVar(&ascii, "ascii", false, "draw the row marks and legend with ASCII characters")
	root.AddCommand(newReportCmd(reportOnce))
	root.AddCommand(&cobra.Command{
		Use:   "setup",
		Short: "Open the dashboard with the Setup window: your default location and an optional NASA FIRMS key (also [s] in the dashboard)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunDashboard(version, app.Options{OpenSetup: true, ASCII: ascii}) // UAT 100: setup is a window over the dashboard, not a separate screen
		},
	})
	root.AddCommand(&cobra.Command{
		Use:   "schema",
		Short: "Print the machine-mode JSON Schema (draft 2020-12)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out, err := schema.Generate()
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(out))
			return nil
		},
	})
	if err := invariant.Check(root.Version != "", "version must be wired into the cobra tree"); err != nil {
		root.Version = "0.0.0-unknown"
	}
	return root
}

// reportFunc is the report command's fetch — app.ReportOnceWithStats in
// the product, a fake in tests, so the command's output paths (plain,
// --json, --verbose, exit codes) are tested without the network (quality
// pass Q2, L3-F25). A parameter, not a package variable (P10-06).
type reportFunc func(context.Context, string) (*snapshot.Snapshot, httpx.RequestStats, error)

func newReportCmd(reportOnce reportFunc) *cobra.Command {
	var asJSON, verbose bool
	cmd := &cobra.Command{
		Use:   "report <city|zip|lat,lon>",
		Short: "One-shot weather report (machine mode: --json; plain text: --report-only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			snap, stats, err := reportOnce(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if err := invariant.Check(snap != nil, "ReportOnce returned no snapshot and no error — report this bug"); err != nil {
				return err
			}
			if asJSON {
				out, err := report.RenderJSON(snap)
				if err != nil {
					return err
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(out)) // stdout write errors surface via cobra exit
			} else {
				// Plain text is the default report surface (and the
				// screen-reader surface, R-12d); width resolved ONCE (T-C').
				_, _ = fmt.Fprint(cmd.OutOrStdout(), report.RenderPlain(snap, term.Width())) // stdout write errors surface via cobra exit
				if verbose {
					_, _ = fmt.Fprint(cmd.OutOrStdout(), report.RenderRequests(stats)) // quality pass Q0: request counters per host
				}
			}
			if code := report.ExitCode(snap); code != 0 {
				// Partial data still prints; the exit code carries the caveat
				// (typed error mapped by main — B1 red-team #8: defer os.Exit
				// skipped cleanup and was untestable).
				return exitCodeError{code: code}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable JSON (see 'watchpost schema')")
	cmd.Flags().Bool("report-only", false, "plain text, the default — explicit for scripts")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "append the request counters per host (plain text only)")
	return cmd
}

// exitCodeError carries a non-zero exit code with a silent message: the
// report already printed; only the process code changes (§10.2).
type exitCodeError struct{ code int }

func (e exitCodeError) Error() string { return "" }
func (e exitCodeError) ExitCode() int { return e.code }
