# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

# Minimal Superset configuration for acceptance testing.
# This file is mounted into the Superset container at /app/superset_config.py.

import os

# Secret key — use a stable value so sessions survive container restarts within one test run.
SECRET_KEY = os.environ.get("SUPERSET_SECRET_KEY", "test-secret-key-do-not-use-in-prod")

# Metadata database (PostgreSQL, provided by the compose stack).
SQLALCHEMY_DATABASE_URI = os.environ.get(
    "DATABASE_URL",
    "postgresql+psycopg2://superset:superset@db:5432/superset",
)

# Disable CSRF for API endpoints so the Terraform provider can call mutating APIs
# without the extra CSRF-token round-trip failing under test conditions.
# Note: we still mock/use CSRF in the provider itself; this just ensures the test
# Superset instance never returns 400/403 on that account.
WTF_CSRF_ENABLED = True

# Allow all origins — tests run on localhost with dynamic ports.
TALISMAN_ENABLED = False

# Allow "unsafe" DB connections such as superset:// (the APSW meta-database
# dialect) and sqlite://. Superset blocks these by default in production;
# for acceptance testing we need them enabled.
PREVENT_UNSAFE_DB_CONNECTIONS = False

# Feature flags useful for testing.
FEATURE_FLAGS = {
    "EMBEDDED_SUPERSET": True,
    "ROW_LEVEL_SECURITY": True,
}

# Keep logs terse during test runs.
LOG_LEVEL = "WARNING"
