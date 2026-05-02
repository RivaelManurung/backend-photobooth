#!/bin/bash

# Comprehensive Test Runner for Photo Booth Backend
# This script runs all tests and generates coverage reports

echo "========================================="
echo "Photo Booth Backend - Test Suite Runner"
echo "========================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Set test environment
export GIN_MODE=test
export DB_HOST=localhost
export DB_PORT=5433
export DB_NAME=photobooth_test
export DB_USER=postgres
export DB_PASSWORD=postgres

echo "📋 Test Configuration:"
echo "  - Environment: TEST"
echo "  - Database: In-Memory SQLite"
echo "  - Coverage: Enabled"
echo ""

# Run tests with coverage
echo "🧪 Running all test suites..."
echo ""

# Run each test suite individually for better reporting
test_files=(
    "auth_test.go"
    "integration_test.go"
    "template_test.go"
    "photo_test.go"
    "admin_test.go"
    "payment_test.go"
    "promo_test.go"
    "session_test.go"
)

failed_tests=()
passed_tests=()

for test_file in "${test_files[@]}"; do
    echo "Running: $test_file"
    if go test -v -coverprofile=coverage_${test_file}.out ./$test_file 2>&1 | tee test_${test_file}.log; then
        echo -e "${GREEN}✓ $test_file PASSED${NC}"
        passed_tests+=("$test_file")
    else
        echo -e "${RED}✗ $test_file FAILED${NC}"
        failed_tests+=("$test_file")
    fi
    echo ""
done

# Run all tests together for overall coverage
echo "📊 Generating overall coverage report..."
go test -v -coverprofile=coverage.out -covermode=atomic ./... 2>&1 | tee test_all.log

# Generate HTML coverage report
if [ -f coverage.out ]; then
    go tool cover -html=coverage.out -o coverage.html
    echo -e "${GREEN}✓ Coverage report generated: coverage.html${NC}"
    
    # Display coverage summary
    echo ""
    echo "📈 Coverage Summary:"
    go tool cover -func=coverage.out | tail -n 1
fi

# Summary
echo ""
echo "========================================="
echo "Test Summary"
echo "========================================="
echo -e "${GREEN}Passed: ${#passed_tests[@]}${NC}"
echo -e "${RED}Failed: ${#failed_tests[@]}${NC}"
echo ""

if [ ${#failed_tests[@]} -eq 0 ]; then
    echo -e "${GREEN}🎉 All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ Some tests failed:${NC}"
    for test in "${failed_tests[@]}"; do
        echo "  - $test"
    done
    exit 1
fi
