#!/bin/bash

set -e

modules=". testutils"

echo "Updating dependencies for all modules in workspace"

go work use -r .

# Pass 1: tidy all modules
echo "Running go mod tidy for all modules (pass 1)..."
for module in ${modules}; do
  if [ -f "$module/go.mod" ]; then
    echo "Tidying $module"
    (cd "$module" && go mod tidy)
  fi
done

# Sync: propagate highest require versions across all modules
echo "Syncing Go workspace..."
go work sync

# Pass 2: re-tidy after sync may have bumped versions
echo "Running go mod tidy for all modules (pass 2)..."
for module in ${modules}; do
  if [ -f "$module/go.mod" ]; then
    echo "Tidying $module"
    (cd "$module" && go mod tidy)
  fi
done

# Verify all modules
echo "Verifying all modules..."
for module in ${modules}; do
  if [ -f "$module/go.mod" ]; then
    echo "Verifying $module"
    (cd "$module" && go mod verify)
  fi
done

# Create unified vendor directory
echo "Creating unified vendor directory..."
go work vendor -v
