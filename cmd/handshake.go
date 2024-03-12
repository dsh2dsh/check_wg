package cmd

import (
	"errors"
	"fmt"
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
			monitoringResponse("latest handshake", args, handshakeResponse).
				OutputAndExit()
		},
	}
)

func init() {
	handshakeCmd.Flags().DurationVarP(&handshakeWarn, "warn", "w", 5*time.Minute,
		"warning threshold")
	handshakeCmd.Flags().DurationVarP(&handshakeCrit, "crit", "c", 15*time.Minute,
		"critical threshold")
}

func handshakeResponse(dump *wg.Dump, resp *monitoringplugin.Response) error {
	peer := dump.OldestHandshake()
	if peer == nil {
		return errors.New("no valid peer found")
	}

	d := time.Since(peer.LatestHandshake).Truncate(time.Second)
	point := monitoringplugin.NewPerformanceDataPoint(
		"latest handshake", d.Seconds()).
		SetUnit("s").
		SetThresholds(monitoringplugin.NewThresholds(
			0, handshakeWarn.Seconds(), 0, handshakeCrit.Seconds()))

	if err := resp.AddPerformanceDataPoint(point); err != nil {
		return fmt.Errorf("add performance point %v: %w",
			peer.LatestHandshake, err)
	}

	resp.UpdateStatus(resp.GetStatusCode(), "peer: "+peer.Name())
	resp.UpdateStatus(resp.GetStatusCode(), "latest handshake: "+d.String()+" ago")

	if resp.GetStatusCode() != monitoringplugin.OK {
		var s string
		if resp.GetStatusCode() == monitoringplugin.WARNING {
			s = handshakeWarn.String()
		} else {
			s = handshakeCrit.String()
		}
		resp.UpdateStatus(resp.GetStatusCode(), "threshold: "+s)
	}
	return nil
}
