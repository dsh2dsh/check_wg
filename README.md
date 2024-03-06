[![Go](https://github.com/dsh2dsh/check_wg/actions/workflows/go.yml/badge.svg)](https://github.com/dsh2dsh/check_wg/actions/workflows/go.yml)

-------------------------------------------------------------------------------

# Icinga2 health check of wireguard peers, using output of wg(8).

## Usage

```
Usage:
  check_wg [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  handshake   check oldest latest handshake
  help        Help about any command

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
OK: oldest latest handshake is OK / peer=10.0.0.3/32 | 'latest-handshake'=265s;~:300;~:900;;

$ check_wg handshake wg show wg0 dump
CRITICAL: latest-handshake is outside of CRITICAL threshold / peer=10.0.0.4/32 | 'latest-handshake'=188016s;~:300;~:180000;;
```
