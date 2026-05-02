#!/bin/bash

# Photo Booth Backend - Setup Verification Script
# This script checks if all components are properly configured

echo "🔍 Photo Booth Backend - Setup Verification"
echo "==========================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
PASSED=0
FAILED=0

# Function to check
check() {
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $1"
        ((PASSED++))
    else
        echo -e "${RED}✗${NC} $1"
        ((FAILED++))
    fi
}

# 1. Check Go installation
echo "1. Checking Go installation..."
go version > /dev/null 2>&1
check "Go is installed"

# 2. Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
if [ "$(printf '%s\n' "1.21" "$GO_VERSION" | sort -V | head -n1)" = "1.21" ]; then
    echo -e "${GREEN}✓${NC} Go version $GO_VERSION (>= 1.21)"
    ((PASSED++))
else
    echo -e "${RED}✗${NC} Go version $GO_VERSION (< 1.21 required)"
    ((FAILED++))
fi

# 3. Check PostgreSQL
echo ""
echo "2. Checking PostgreSQL..."
psql --version > /dev/null 2>&1
check "PostgreSQL is installed"

# 4. Check if database exists
psql -U postgres -lqt | cut -d \| -f 1 | grep -qw photobooth
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓${NC} Database 'photobooth' exists"
    ((PASSED++))
else
    echo -e "${YELLOW}⚠${NC} Database 'photobooth' not found (will be created)"
fi

# 5. Check Redis (optional)
echo ""
echo "3. Checking Redis (optional)..."
redis-cli ping > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓${NC} Redis is running"
    ((PASSED++))
else
    echo -e "${YELLOW}⚠${NC} Redis not running (optional for caching)"
fi

# 6. Check .env file
echo ""
echo "4. Checking configuration..."
if [ -f ".env" ]; then
    echo -e "${GREEN}✓${NC} .env file exists"
    ((PASSED++))
    
    # Check required variables
    if grep -q "GOPAY_MERCHANT_ID" .env && [ "$(grep GOPAY_MERCHANT_ID .env | cut -d '=' -f2)" != "" ]; then
        echo -e "${GREEN}✓${NC} GOPAY_MERCHANT_ID is set"
        ((PASSED++))
    else
        echo -e "${RED}✗${NC} GOPAY_MERCHANT_ID not set"
        ((FAILED++))
    fi
    
    if grep -q "GOPAY_SECRET_KEY" .env && [ "$(grep GOPAY_SECRET_KEY .env | cut -d '=' -f2)" != "" ]; then
        echo -e "${GREEN}✓${NC} GOPAY_SECRET_KEY is set"
        ((PASSED++))
    else
        echo -e "${RED}✗${NC} GOPAY_SECRET_KEY not set"
        ((FAILED++))
    fi
    
    if grep -q "JWT_SECRET" .env && [ "$(grep JWT_SECRET .env | cut -d '=' -f2)" != "your-secret-key-change-in-production" ]; then
        echo -e "${GREEN}✓${NC} JWT_SECRET is customized"
        ((PASSED++))
    else
        echo -e "${YELLOW}⚠${NC} JWT_SECRET should be changed for production"
    fi
else
    echo -e "${RED}✗${NC} .env file not found"
    echo -e "${YELLOW}→${NC} Run: cp .env.example .env"
    ((FAILED++))
fi

# 7. Check required directories
echo ""
echo "5. Checking directories..."
DIRS=("uploads" "uploads/photos" "uploads/templates" "uploads/processed" "uploads/thumbnails" "uploads/strips" "uploads/qris")
for dir in "${DIRS[@]}"; do
    if [ -d "$dir" ]; then
        echo -e "${GREEN}✓${NC} Directory $dir exists"
        ((PASSED++))
    else
        echo -e "${RED}✗${NC} Directory $dir not found"
        echo -e "${YELLOW}→${NC} Run: mkdir -p $dir"
        ((FAILED++))
    fi
done

# 8. Check Go dependencies
echo ""
echo "6. Checking Go dependencies..."
if [ -f "go.mod" ]; then
    echo -e "${GREEN}✓${NC} go.mod exists"
    ((PASSED++))
    
    # Check critical dependencies
    DEPS=("github.com/gin-gonic/gin" "gorm.io/gorm" "github.com/skip2/go-qrcode" "github.com/gorilla/websocket")
    for dep in "${DEPS[@]}"; do
        if grep -q "$dep" go.mod; then
            echo -e "${GREEN}✓${NC} Dependency $dep found"
            ((PASSED++))
        else
            echo -e "${RED}✗${NC} Dependency $dep missing"
            ((FAILED++))
        fi
    done
else
    echo -e "${RED}✗${NC} go.mod not found"
    ((FAILED++))
fi

# 9. Check main files
echo ""
echo "7. Checking source files..."
FILES=("main.go" "config/config.go" "database/database.go" "models/qris_payment.go" "services/gopay_qris.go" "handlers/gopay_handler.go")
for file in "${FILES[@]}"; do
    if [ -f "$file" ]; then
        echo -e "${GREEN}✓${NC} File $file exists"
        ((PASSED++))
    else
        echo -e "${RED}✗${NC} File $file not found"
        ((FAILED++))
    fi
done

# 10. Try to build
echo ""
echo "8. Testing build..."
go build -o photobooth_test > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓${NC} Build successful"
    ((PASSED++))
    rm -f photobooth_test
else
    echo -e "${RED}✗${NC} Build failed"
    echo -e "${YELLOW}→${NC} Run: go mod download && go build"
    ((FAILED++))
fi

# Summary
echo ""
echo "==========================================="
echo "📊 Verification Summary"
echo "==========================================="
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ All checks passed! Backend is ready.${NC}"
    echo ""
    echo "Next steps:"
    echo "1. Update GoPay credentials in .env"
    echo "2. Run: go run main.go"
    echo "3. Test: curl http://localhost:8080/health"
    exit 0
else
    echo -e "${RED}❌ Some checks failed. Please fix the issues above.${NC}"
    echo ""
    echo "Quick fixes:"
    echo "1. Install missing dependencies: go mod download"
    echo "2. Create directories: mkdir -p uploads/{photos,templates,processed,thumbnails,strips,qris}"
    echo "3. Copy env file: cp .env.example .env"
    echo "4. Create database: createdb photobooth"
    exit 1
fi
