#!/usr/bin/env bash
#
# Minimal cleura login pattern for any CI system.
#
# Provide via your CI's secret store (as environment variables):
#   CLEURA_USERNAME      — service account (must not have SMS 2FA)
#   CLEURA_API_PASSWORD  — password; read by `cleura login`, never stored
#   CLEURA_PROJECT_ID    — project for project-scoped commands

set -euo pipefail

# On shared runners, keep this job's credentials out of the shared HOME.
export CLEURA_CONFIG="$PWD/cleura-config.yaml"

cleura login -u "$CLEURA_USERNAME" --profile ci \
  --cloud public --region sto1 --project-id "$CLEURA_PROJECT_ID"

# Revoke the job's token on exit, success or failure.
trap 'cleura logout --profile ci -q' EXIT

cleura whoami
cleura gardener shoot list
