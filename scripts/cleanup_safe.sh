#!/bin/bash
set -e

echo "=== PAW Blockchain Safe Cleanup ==="
echo "Removing test executables and temporary files..."

# Count files before
echo "Files to remove:"
ls -lh *.exe 2>/dev/null | wc -l || echo "0"
ls -lh */*.exe 2>/dev/null | wc -l || echo "0"

# Remove test executables (not in , just filesystem)
rm -f *.exe */*.exe *.test.exe 2>/dev/null || true

# Remove empty file
rm -f list.json 2>/dev/null || true

echo "Cleanup complete!"
echo "Test executables removed from filesystem"
echo "Note: Files already committed need ' rm' in separate step"
