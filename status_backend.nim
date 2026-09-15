# A thin Nim host for the status-backend HTTP server: all server logic lives on
# the Go side (StatusBackendRunServer, wrapping cmd/status-backend/server).
# Build it from a checkout with `nim libstatus statusgo.nims` followed by
# `nim c status_backend.nim`.
import std/[os, strutils]
import ./status_go

proc usage() =
  echo "usage: status_backend [--address=<host:port>]"
  echo "  --address  host:port to listen on (default 127.0.0.1:0 = random port)"

when isMainModule:
  var address = "127.0.0.1:0"
  for param in commandLineParams():
    if param.startsWith("--address="):
      address = param.split('=', maxsplit = 1)[1]
    elif param in ["-h", "--help"]:
      usage()
      quit QuitSuccess
    else:
      usage()
      quit "unknown argument: " & param, QuitFailure
  let err = statusBackendRunServer(address)  # blocks while serving
  if err.len > 0:
    quit "status-backend failed: " & err, QuitFailure
