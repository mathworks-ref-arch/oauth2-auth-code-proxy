#!/bin/bash
set -e

cd "$(git rev-parse --show-toplevel)"

echo "Running pre-push quality checks..."

# 1. goimports formatting
echo "  - goimports..."
BAD=$(goimports -l . 2>&1 || true)
if [ -n "$BAD" ]; then
  echo "$BAD"
  echo '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"goimports check failed. Run: goimports -w ."}}'
  exit 0
fi
echo "    OK"

# 2. golangci-lint
echo "  - golangci-lint..."
if ! golangci-lint run 2>&1; then
  echo '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"golangci-lint found issues. Run: golangci-lint run"}}'
  exit 0
fi
echo "    OK"

# 3. gocyclo complexity
echo "  - gocyclo..."
CYCLO=$(gocyclo -over 9 ./internal/ 2>&1 || true)
if [ -n "$CYCLO" ]; then
  echo "$CYCLO"
  echo '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"gocyclo complexity exceeds threshold (>9). Refactor the flagged functions."}}'
  exit 0
fi
echo "    OK"

# 4. go test
echo "  - go test..."
if ! go test -count=1 ./... 2>&1; then
  echo '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"Tests failed. Run: go test -count=1 ./..."}}'
  exit 0
fi
echo "    OK"

echo "All pre-push checks passed."
exit 0
