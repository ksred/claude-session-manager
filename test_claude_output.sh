#!/bin/bash

echo "Testing Claude CLI output..."
echo "Running: claude"
echo "=========================="

# Run claude and capture the first 500 bytes of output
# Use timeout to avoid hanging
timeout 5s claude 2>&1 | head -c 500 | od -c

echo ""
echo "=========================="
echo "Hex dump of output:"
timeout 5s claude 2>&1 | head -c 500 | xxd

echo ""
echo "=========================="
echo "Raw output (first 20 lines):"
timeout 5s claude 2>&1 | head -20