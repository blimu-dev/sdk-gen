#!/bin/bash
echo "Formatting Go code..."

# Format with gofmt (built-in Go formatter)
gofmt -w .

# Try to find and use goimports
if command -v goimports &> /dev/null; then
    goimports -w .
elif [ -f "$HOME/go/bin/goimports" ]; then
    "$HOME/go/bin/goimports" -w .
else
    echo "Warning: goimports not found, skipping import organization"
fi

echo "Go formatting completed!"
