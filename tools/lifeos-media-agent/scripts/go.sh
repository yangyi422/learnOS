#!/bin/sh
set -eu
if [ -n "${MEDIA_AGENT_GO:-}" ]; then
  exec "$MEDIA_AGENT_GO" "$@"
fi
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
for candidate in "$script_dir/../../../.tmp/lifeos-toolchain/go/bin/go" "$HOME/Library/Application Support/LifeOSMediaAgent/toolchain/go/bin/go" /opt/homebrew/bin/go /usr/local/go/bin/go; do
  if [ -x "$candidate" ]; then
    exec "$candidate" "$@"
  fi
done
exec go "$@"
