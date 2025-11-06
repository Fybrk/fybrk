#!/bin/bash

echo "=== COMPREHENSIVE FYBRK SYNC TEST ==="
echo "Testing all sync scenarios..."
echo

# Cleanup
pkill -f fybrk 2>/dev/null || true
rm -rf test-device1 test-device2 test-results.txt

# Test results tracking
TESTS_PASSED=0
TESTS_FAILED=0

test_result() {
    if [ $1 -eq 0 ]; then
        echo "✅ $2"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo "❌ $2"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

# Setup test directories
mkdir -p test-device1 test-device2

echo "=== TEST 1: Basic Initialization ==="
cd test-device1
echo "test content" > initial.txt
../bin/fybrk . > ../device1.log 2>&1 &
DEVICE1_PID=$!
cd ..

sleep 2

# Check if fybrk started properly
if ps -p $DEVICE1_PID > /dev/null; then
    test_result 0 "Device 1 started successfully"
else
    test_result 1 "Device 1 failed to start"
fi

# Check if .fybrk directory was created
if [ -d "test-device1/.fybrk" ]; then
    test_result 0 ".fybrk directory created"
else
    test_result 1 ".fybrk directory missing"
fi

# Check if database was created
if [ -f "test-device1/.fybrk/metadata.db" ]; then
    test_result 0 "Database created"
else
    test_result 1 "Database missing"
fi

# Check if key was generated
if [ -f "test-device1/.fybrk/key" ]; then
    test_result 0 "Encryption key generated"
else
    test_result 1 "Encryption key missing"
fi

echo
echo "=== TEST 2: File Watching ==="

# Create new file
echo "new file content" > test-device1/newfile.txt
sleep 1

# Modify existing file
echo "modified content" >> test-device1/initial.txt
sleep 1

# Check logs for file events
if grep -q "File event.*newfile.txt" device1.log; then
    test_result 0 "New file creation detected"
else
    test_result 1 "New file creation not detected"
fi

if grep -q "File event.*initial.txt" device1.log; then
    test_result 0 "File modification detected"
else
    test_result 1 "File modification not detected"
fi

echo
echo "=== TEST 3: Server Functionality ==="

# Check if server is listening
SERVER_PORT=$(grep "Server listening on port" device1.log | grep -o '[0-9]*')
if [ ! -z "$SERVER_PORT" ]; then
    test_result 0 "WebSocket server started on port $SERVER_PORT"
    
    # Test if port is actually listening
    if nc -z localhost $SERVER_PORT 2>/dev/null; then
        test_result 0 "Server port is accessible"
    else
        test_result 1 "Server port not accessible"
    fi
else
    test_result 1 "Server port not found in logs"
fi

echo
echo "=== TEST 4: Second Device ==="

cd test-device2
echo "device2 content" > device2.txt
../bin/fybrk . > ../device2.log 2>&1 &
DEVICE2_PID=$!
cd ..

sleep 2

if ps -p $DEVICE2_PID > /dev/null; then
    test_result 0 "Device 2 started successfully"
else
    test_result 1 "Device 2 failed to start"
fi

echo
echo "=== TEST 5: File Operations ==="

# Test various file operations
echo "testing create" > test-device1/create-test.txt
sleep 1

echo "testing modify" >> test-device1/initial.txt
sleep 1

mkdir -p test-device1/subdir
echo "subdirectory file" > test-device1/subdir/subfile.txt
sleep 1

rm test-device1/newfile.txt
sleep 1

# Check if all operations were detected
OPERATIONS_DETECTED=0

if grep -q "create-test.txt" device1.log; then
    OPERATIONS_DETECTED=$((OPERATIONS_DETECTED + 1))
fi

if grep -q "subfile.txt" device1.log; then
    OPERATIONS_DETECTED=$((OPERATIONS_DETECTED + 1))
fi

if [ $OPERATIONS_DETECTED -ge 2 ]; then
    test_result 0 "Multiple file operations detected ($OPERATIONS_DETECTED)"
else
    test_result 1 "File operations not properly detected ($OPERATIONS_DETECTED)"
fi

echo
echo "=== TEST 6: Database Integrity ==="

# Check if database has records
DB_PATH="test-device1/.fybrk/metadata.db"
if command -v sqlite3 >/dev/null 2>&1; then
    FILE_COUNT=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM files;" 2>/dev/null || echo "0")
    if [ "$FILE_COUNT" -gt 0 ]; then
        test_result 0 "Database contains $FILE_COUNT file records"
    else
        test_result 1 "Database has no file records"
    fi
else
    echo "⚠️  sqlite3 not available, skipping database check"
fi

echo
echo "=== TEST 7: Pair URL Generation ==="

if grep -q "fybrk://pair" device1.log; then
    test_result 0 "Pair URL generated"
    PAIR_URL=$(grep "fybrk://pair" device1.log | head -1 | cut -d' ' -f3)
    echo "   URL: $PAIR_URL"
else
    test_result 1 "Pair URL not generated"
fi

echo
echo "=== TEST 8: Error Handling ==="

# Test invalid directory
../bin/fybrk /nonexistent/directory > error-test.log 2>&1 &
ERROR_PID=$!
sleep 1

if ! ps -p $ERROR_PID > /dev/null 2>&1; then
    test_result 0 "Graceful error handling for invalid directory"
else
    kill $ERROR_PID 2>/dev/null
    test_result 1 "Did not handle invalid directory properly"
fi

echo
echo "=== TEST 9: Resource Cleanup ==="

# Test graceful shutdown
kill $DEVICE1_PID $DEVICE2_PID 2>/dev/null
sleep 2

if ! ps -p $DEVICE1_PID > /dev/null 2>&1 && ! ps -p $DEVICE2_PID > /dev/null 2>&1; then
    test_result 0 "Processes terminated cleanly"
else
    test_result 1 "Processes did not terminate cleanly"
    pkill -9 -f fybrk 2>/dev/null || true
fi

echo
echo "=== TEST SUMMARY ==="
echo "Tests Passed: $TESTS_PASSED"
echo "Tests Failed: $TESTS_FAILED"
echo "Total Tests: $((TESTS_PASSED + TESTS_FAILED))"

if [ $TESTS_FAILED -eq 0 ]; then
    echo "🎉 ALL TESTS PASSED!"
    exit 0
else
    echo "💥 SOME TESTS FAILED!"
    exit 1
fi
