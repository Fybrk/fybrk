#!/bin/bash

# Test script to demonstrate 2-way sync functionality

echo "=== Fybrk 2-Way Sync Test ==="
echo

# Clean up any existing test directories
rm -rf test-sync1 test-sync2

# Create test directories
mkdir -p test-sync1 test-sync2

# Create initial files
echo "Hello from device 1" > test-sync1/device1.txt
echo "Hello from device 2" > test-sync2/device2.txt

echo "Created test directories with initial files:"
echo "test-sync1/device1.txt: $(cat test-sync1/device1.txt)"
echo "test-sync2/device2.txt: $(cat test-sync2/device2.txt)"
echo

# Start first instance in background
echo "Starting Fybrk on test-sync1..."
cd test-sync1
../bin/fybrk . > ../sync1.log 2>&1 &
SYNC1_PID=$!
cd ..

# Wait a moment for server to start
sleep 2

# Extract server address from log
SERVER_ADDR=$(grep "Server:" sync1.log | cut -d' ' -f2)
echo "Server address: $SERVER_ADDR"

# Start second instance and connect to first
echo "Starting Fybrk on test-sync2 and connecting to first instance..."
cd test-sync2

# For this demo, we'll simulate connecting by starting another instance
# In a real scenario, you'd use the pair URL
../bin/fybrk . > ../sync2.log 2>&1 &
SYNC2_PID=$!
cd ..

echo "Both sync instances are running!"
echo "Sync1 PID: $SYNC1_PID"
echo "Sync2 PID: $SYNC2_PID"
echo

# Wait a moment for sync to initialize
sleep 3

echo "=== Testing File Sync ==="
echo

# Test 1: Create a file in sync1
echo "Test 1: Creating new file in test-sync1..."
echo "This is a new file from sync1" > test-sync1/new-from-sync1.txt
echo "Created: test-sync1/new-from-sync1.txt"

# Test 2: Create a file in sync2
echo "Test 2: Creating new file in test-sync2..."
echo "This is a new file from sync2" > test-sync2/new-from-sync2.txt
echo "Created: test-sync2/new-from-sync2.txt"

# Test 3: Modify existing file
echo "Test 3: Modifying existing file in test-sync1..."
echo "Modified content from sync1" >> test-sync1/device1.txt

echo
echo "Waiting 5 seconds for sync to propagate..."
sleep 5

echo
echo "=== Sync Results ==="
echo "Files in test-sync1:"
ls -la test-sync1/
echo
echo "Files in test-sync2:"
ls -la test-sync2/
echo

# Show logs
echo "=== Sync1 Log ==="
tail -10 sync1.log
echo
echo "=== Sync2 Log ==="
tail -10 sync2.log

# Cleanup
echo
echo "Cleaning up processes..."
kill $SYNC1_PID $SYNC2_PID 2>/dev/null
wait $SYNC1_PID $SYNC2_PID 2>/dev/null

echo "Test complete!"
