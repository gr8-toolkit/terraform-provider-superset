#!/bin/bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0
#
# Bootstraps a fresh Superset metadata database and creates the admin user.
# Runs once as an init container / one-shot service in docker-compose.

set -euo pipefail

echo "==> Waiting for the metadata database to be ready..."
until python3 -c "
import socket, sys
try:
    s = socket.create_connection(('db', 5432), timeout=2)
    s.close()
    sys.exit(0)
except OSError:
    sys.exit(1)
"; do
  sleep 2
done

echo "==> Upgrading metadata DB schema..."
superset db upgrade

echo "==> Creating admin user..."
superset fab create-admin \
  --username admin \
  --firstname Admin \
  --lastname User \
  --email admin@localhost \
  --password admin \
  2>/dev/null || true   # ignore "already exists" errors

echo "==> Initialising Superset..."
superset init

echo "==> Bootstrap complete."
