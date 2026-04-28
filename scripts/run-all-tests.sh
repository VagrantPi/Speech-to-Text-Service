#!/bin/bash
set -e

echo "========================================"
echo "Running all tests..."
echo "========================================"

# Run packages tests
echo ""
echo ">>> packages/"
echo "========================================"
for pkg in packages/*/; do
    if [ -d "$pkg" ] && [ -f "$pkg/go.mod" ]; then
        echo "--- $pkg"
        (cd "$pkg" && go test ./... 2>&1 | tail -3) || true
    fi
done

# Run apps tests
echo ""
echo ">>> apps/"
echo "========================================"
for app in apps/*/; do
    if [ -d "$app" ] && [ -f "$app/go.mod" ]; then
        echo "--- $app"
        (cd "$app" && go test ./... 2>&1 | tail -3) || true
    fi
done

echo ""
echo "========================================"
echo "All tests complete!"
echo "========================================"