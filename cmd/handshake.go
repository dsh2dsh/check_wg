package cmd

import (
	"fmt"
	"io"
	"time"

	"github.com/inexio/go-monitoringplugin"
	"github.com/spf13/cobra"

	"github.com/dsh2dsh/check_wg/wg"
)

var (
	handshakeWarn, handshakeCrit time.Duration

	handshakeCmd = cobra.Command{
		Use:   "handshake [-w 5m] [-c 15m] [wg show wg0 dump]",
		Short: "check oldest latest handshake",
		Long: `It executes given wg(8) command and reads its output or stdin, if no
command was given at all.

It analizes latest handshake of every peer and outputs warning or critical
status if any of them is greater of given threshold.`,

		Run: func(cmd *cobra.Command, args []string) {
			CheckLatestHandshake(args)
		},
	}
)

func init() {
	handshakeCmd.Flags().DurationVarP(&handshakeWarn, "warn", "w", 5*time.Minute,
		"warning threshold")
	handshakeCmd.Flags().DurationVarP(&handshakeCrit, "crit", "c", 15*time.Minute,
		"critical threshold")
}

func CheckLatestHandshake(args []string) {
	resp := monitoringplugin.NewResponse("oldest latest handshake is OK")
	resp.SetOutputDelimiter(" / ")
	defer resp.OutputAndExit()

	peer, err := OldestHandshake(args)
	if resp.UpdateStatusOnError(err, monitoringplugin.WARNING, "", true) {
		return
	} else if !peer.Valid() {
		resp.UpdateStatus(monitoringplugin.WARNING, "no valid peer found")
		return
	}

	d := time.Since(peer.LatestHandshake).Truncate(time.Second).Seconds()
	point := monitoringplugin.NewPerformanceDataPoint("latest-handshake", d).
		SetUnit("s").
		SetThresholds(monitoringplugin.NewThresholds(
			nil, handshakeWarn.Seconds(), nil, handshakeCrit.Seconds()))

	if err := resp.AddPerformanceDataPoint(point); err != nil {
		resp.UpdateStatusOnError(
			fmt.Errorf("failed add performance data: %w", err),
			monitoringplugin.WARNING, "", true)
		return
	}
	resp.UpdateStatus(monitoringplugin.OK, fmt.Sprintf("peer=%v", peer.Name()))
}

func OldestHandshake(args []string) (wg.DumpPeer, error) {
	var peer wg.DumpPeer
	err := withWgCmd(args, func(r io.Reader) error {
		wgDump, err := wg.NewDump(r)
		if err != nil {
			if len(args) == 0 {
				return fmt.Errorf("with input from stdin: %w", err)
			}
			return fmt.Errorf("with input from %v: %w", args, err)
		}

		if oldestPeer := wgDump.OldestHandshake(); oldestPeer != nil {
			peer = *oldestPeer
		}
		return nil
	})
	return peer, err
}
