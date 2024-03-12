[![Go](https://github.com/dsh2dsh/check_wg/actions/workflows/go.yml/badge.svg)](https://github.com/dsh2dsh/check_wg/actions/workflows/go.yml)

-------------------------------------------------------------------------------

# Icinga2 health check of wireguard peers, using output of wg(8).

FreeBSD port [here](https://github.com/dsh2dsh/freebsd-ports/tree/master/net-mgmt/check_wg)

## Usage

```
Usage:
  check_wg [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  handshake   check oldest latest handshake
  help        Help about any command
  transfer    Outputs transfer stats

Flags:
  -h, --help   help for check_wg

Use "check_wg [command] --help" for more information about a command.
```

```
$ check_wg handshake -h
It executes given wg(8) command and reads its output or stdin, if no
command was given at all.

It analizes latest handshake of every peer and outputs warning or critical
status if any of them is greater of given threshold.

Usage:
  check_wg handshake [-w 5m] [-c 15m] [wg show wg0 dump] [flags]

Flags:
  -c, --crit duration   critical threshold (default 15m0s)
  -h, --help            help for handshake
  -w, --warn duration   warning threshold (default 5m0s)

$ check_wg handshake wg show wg0 dump
OK: latest handshake
peer: 10.0.0.3/32
latest handshake: 5m36s ago | 'latest handshake'=265s;300;900;;

$ check_wg handshake wg show wg0 dump
CRITICAL: latest handshake is outside of CRITICAL threshold
peer: 10.0.0.4/32
latest handshake: 188016s ago
threshold: 180000s | 'latest handshake'=188016s;300;180000;;
```

```
$ check_wg transfer -h
Outputs transfer stats

Usage:
  check_wg transfer [flags] PEER [wg show wg0 dump]

Flags:
  -h, --help   help for transfer

$ check_wg transfer 10.0.0.5/32 wg show wg0 dump
OK: bytes transferred
peer=10.0.0.5/32 | 'rx'=5319303179b 'tx'=81508002220b
```

## Icinga2 configuration examples

```
object CheckCommand "check_wg_handshake" {
  command = [ PluginDir + "/check_wg" ]

  arguments = {
    "--handshake" = {
      value = "handshake"
      order = -1
      skip_key = true
    }

    "-w" = {
      value = "$wg_handshake_warn$"
    }
    "-c" = {
      value = "$wg_handshake_crit$"
    }

    "--" = {
      value = "--"
      order = 1
      skip_key = true
    }
    "--wg-bin" = {
      value = "/usr/bin/wg"
      order = 2
      skip_key = true
    }
    "--show" = {
      value = "show"
      order = 3
      skip_key = true
    }
    "--ifname" = {
      value = "$wg_ifname$"
      order = 4
      skip_key = true
    }
    "--dump" = {
      value = "dump"
      order = 5
      skip_key = true
    }
  }

  vars.wg_ifname = "wg0"
  vars.wg_handshake_warn = "30m"
  vars.wg_handshake_crit = "1h"
}
```

```
object CheckCommand "check_wg_transfer" {
  command = [ PluginDir + "/check_wg" ]

  arguments = {
    "--transfer" = {
      value = "transfer"
      order = -1
      skip_key = true
    }
    "--peer" = {
      value = "$wg_peer$"
      required = true
      skip_key = true
    }
    "--" = {
      value = "--"
      order = 1
      skip_key = true
    }
    "--wg-bin" = {
      value = "/usr/bin/wg"
      order = 2
      skip_key = true
    }
    "--show" = {
      value = "show"
      order = 3
      skip_key = true
    }
    "--ifname" = {
      value = "$wg_ifname$"
      order = 4
      required = true
      skip_key = true
    }
    "--dump" = {
      value = "dump"
      order = 5
      skip_key = true
    }
  }

  vars.wg_ifname = "wg0"
}
```

```
apply Service "wg_handshake_" for (ifname in host.vars.wg_ifaces) {
  import "generic-service"

  check_command = "check_wg_handshake"
  command_endpoint = host.vars.agent_endpoint

  vars.wg_iface = ifname

  assign where host.vars.wg_ifaces
}
```

```
apply Service "wg_transfer_" for (name => cfg in host.vars.wg_peers) {
  import "generic-service"

  check_command = "check_wg_transfer"
  command_endpoint = host.vars.agent_endpoint

  vars.wg_iface = cfg.ifname
  vars.wg_peer = cfg.peer

  assign where host.vars.wg_peers
}
```

```
object Host "server" {
  vars.wg_ifaces = [ "wg0" ]
  vars.wg_peers["peer1"] = {
    ifname = "wg0"
    peer = "10.0.0.2/32"
  }
  vars.wg_peers["peer2"] = {
    ifname = "wg0"
    peer = "10.0.0.3/32"
  }
}
```
