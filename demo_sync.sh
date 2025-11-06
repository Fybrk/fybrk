#!/bin/bash

echo "=== Fybrk Real-Time Sync Demo ==="
echo

# Clean up
rm -rf demo-sync
mkdir demo-sync
cd demo-sync

# Create initial file
echo "Initial content" > test.txt

echo "Starting Fybrk sync..."
echo "Watch this directory in another terminal: watch -n 1 'ls -la && echo && cat test.txt 2>/dev/null'"
echo

# Start fybrk
./bin/fybrk . &
FYBRK_PID=$!

# Wait for startup
sleep 2

echo "Fybrk is now monitoring this directory for changes."
echo "Try these commands in another terminal:"
echo "  cd $(pwd)"
echo "  echo 'Hello World' > test.txt"
echo "  echo 'More content' >> test.txt"
echo "  touch newfile.txt"
echo "  rm test.txt"
echo
echo "You should see file events in the log below."
echo "Press Ctrl+C to stop."
echo

# Show live log
tail -f ../demo-sync/.fybrk/sync.log 2>/dev/null || wait $FYBRK_PID
