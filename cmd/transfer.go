package cmd

import (
	"fmt"

	"github.com/inexio/go-monitoringplugin"
	"github.com/spf13/cobra"

	"github.com/dsh2dsh/check_wg/wg"
)

var transferCmd = cobra.Command{
	Use:   "transfer [flags] PEER [wg show wg0 dump]",
	Short: "Outputs transfer stats",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		peerName := args[0]
		var peerArgs []string
		if len(args) > 1 {
			peerArgs = args[1:]
		}
		monitoringResponse("bytes transferred", peerArgs,
			func(dump *wg.Dump, resp *monitoringplugin.Response) error {
				return transferResponse(dump, peerName, resp)
			}).
			OutputAndExit()
	},
}

func transferResponse(dump *wg.Dump, name string,
	resp *monitoringplugin.Response,
) error {
	peer := dump.Peer(name)
	if peer == nil {
		return fmt.Errorf("peer not found: %s", name)
	}
	resp.UpdateStatus(monitoringplugin.OK, fmt.Sprintf("peer=%v", peer.Name()))

	points := [...]struct {
		Label   string
		Octests uint64
	}{
		{Label: "rx", Octests: peer.Rx},
		{Label: "tx", Octests: peer.Tx},
	}

	for i := range points {
		pd := &points[i]
		point := monitoringplugin.NewPerformanceDataPoint(pd.Label, pd.Octests).
			SetUnit("b")
		if err := resp.AddPerformanceDataPoint(point); err != nil {
			return fmt.Errorf("failed add performance data %s=%v: %w",
				pd.Label, pd.Octests, err)
		}
	}
	return nil
}
