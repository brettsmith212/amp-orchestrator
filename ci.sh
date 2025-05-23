#!/bin/bash

set -euo pipefail

# CI Script for Amp Orchestrator
# This script is called by the post-receive hook

# Get the repository and branch information
REPO_DIR="$1"
REF_NAME="$2"
COMMIT_HASH="$3"

echo "Running CI for $REF_NAME ($COMMIT_HASH)"

# Store the original working directory
ORIGINAL_DIR="$(pwd)"

# Create status directory if it doesn't exist  
# Use the directory relative to where the script is called from
STATUS_DIR="$ORIGINAL_DIR/ci-status"
mkdir -p "$STATUS_DIR"

# Create a temporary working directory
WORK_DIR=$(mktemp -d)
echo "Using working directory: $WORK_DIR"

# Cleanup function to run on exit
cleanup() {
  echo "Cleaning up $WORK_DIR"
  rm -rf "$WORK_DIR"
}
trap cleanup EXIT

# Clone the repository
git clone "$REPO_DIR" "$WORK_DIR/repo"
cd "$WORK_DIR/repo"
git checkout "$COMMIT_HASH"

# Run tests
echo "Running tests..."
STATUS="PASS"
OUTPUT=""

if [ -f "go.mod" ]; then
  # Run Go tests
  if ! OUTPUT=$(go test ./... 2>&1); then
    STATUS="FAIL"
  fi
else
  # No tests found
  OUTPUT="No tests to run"
fi

# Create status JSON file properly escaped
jq -n \
  --arg ref "$REF_NAME" \
  --arg commit "$COMMIT_HASH" \
  --arg status "$STATUS" \
  --arg timestamp "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
  --arg output "$OUTPUT" \
  '{
    ref: $ref,
    commit: $commit,
    status: $status,
    timestamp: $timestamp,
    output: $output
  }' > "$STATUS_DIR/$COMMIT_HASH.json"

echo "CI completed with status: $STATUS"
echo "Status saved to $STATUS_DIR/$COMMIT_HASH.json"

exit 0