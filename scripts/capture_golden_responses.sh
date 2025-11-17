#!/bin/bash

# Script to capture golden JSON responses from Laravel monolith
# Usage: ./scripts/capture_golden_responses.sh

set -e

# Configuration
LARAVEL_URL="${LARAVEL_URL:-http://localhost:8000}"
OUTPUT_DIR="tests/golden/testdata"
TEST_USER="${TEST_USER:-test_golden_user}"
TEST_PASSWORD="${TEST_PASSWORD:-password}"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== MetaRGB Golden Response Capture ===${NC}"
echo ""

# Ensure output directory exists
mkdir -p "$OUTPUT_DIR"

# Function to pretty print JSON
save_json() {
    local file=$1
    local data=$2
    echo "$data" | jq '.' > "$OUTPUT_DIR/$file"
    echo -e "${GREEN}✓${NC} Saved: $file"
}

# Step 1: Authenticate
echo -e "${BLUE}[1/8]${NC} Authenticating..."
LOGIN_RESPONSE=$(curl -s -X POST "$LARAVEL_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"$TEST_USER\",\"password\":\"$TEST_PASSWORD\"}")

TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.token // .data.token // empty')

if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
    echo -e "${RED}✗${NC} Authentication failed"
    echo "Response: $LOGIN_RESPONSE"
    exit 1
fi

echo -e "${GREEN}✓${NC} Authenticated successfully"
echo ""

# Step 2: Capture /api/auth/me
echo -e "${BLUE}[2/8]${NC} Capturing /api/auth/me..."
AUTH_ME=$(curl -s -X POST "$LARAVEL_URL/api/auth/me" \
    -H "Authorization: Bearer $TOKEN")
save_json "auth_get_me.json" "$AUTH_ME"

# Step 3: Capture /api/user/wallet
echo -e "${BLUE}[3/8]${NC} Capturing /api/user/wallet..."
WALLET=$(curl -s -X GET "$LARAVEL_URL/api/user/wallet" \
    -H "Authorization: Bearer $TOKEN")
save_json "user_wallet.json" "$WALLET"

# Step 4: Capture /api/features (with bbox)
echo -e "${BLUE}[4/8]${NC} Capturing /api/features..."
FEATURES=$(curl -s -X GET "$LARAVEL_URL/api/features?bbox=35.0,51.0,36.0,52.0" \
    -H "Authorization: Bearer $TOKEN")
save_json "features_list.json" "$FEATURES"

# Step 5: Capture /api/features/{id}
echo -e "${BLUE}[5/8]${NC} Capturing /api/features/{id}..."
# Extract first feature ID from features list
FEATURE_ID=$(echo "$FEATURES" | jq -r '.data[0].id // empty')
if [ -n "$FEATURE_ID" ] && [ "$FEATURE_ID" != "null" ]; then
    FEATURE_DETAIL=$(curl -s -X GET "$LARAVEL_URL/api/features/$FEATURE_ID" \
        -H "Authorization: Bearer $TOKEN")
    save_json "feature_detail.json" "$FEATURE_DETAIL"
else
    echo -e "${RED}✗${NC} No features found, skipping feature detail"
fi

# Step 6: Capture /api/user/transactions
echo -e "${BLUE}[6/8]${NC} Capturing /api/user/transactions..."
TRANSACTIONS=$(curl -s -X GET "$LARAVEL_URL/api/user/transactions?page=1&per_page=10" \
    -H "Authorization: Bearer $TOKEN")
save_json "transactions_list.json" "$TRANSACTIONS"

# Step 7: Capture /api/users/{id}/levels
echo -e "${BLUE}[7/8]${NC} Capturing /api/users/{id}/levels..."
USER_ID=$(echo "$AUTH_ME" | jq -r '.data.id // empty')
if [ -n "$USER_ID" ] && [ "$USER_ID" != "null" ]; then
    LEVELS=$(curl -s -X GET "$LARAVEL_URL/api/users/$USER_ID/levels" \
        -H "Authorization: Bearer $TOKEN")
    save_json "user_level.json" "$LEVELS"
else
    echo -e "${RED}✗${NC} Could not extract user ID, skipping levels"
fi

# Step 8: Capture /api/notifications
echo -e "${BLUE}[8/8]${NC} Capturing /api/notifications..."
NOTIFICATIONS=$(curl -s -X GET "$LARAVEL_URL/api/notifications?page=1&per_page=10" \
    -H "Authorization: Bearer $TOKEN")
save_json "notifications_list.json" "$NOTIFICATIONS"

echo ""
echo -e "${GREEN}=== Capture Complete ===${NC}"
echo ""
echo "Golden files saved to: $OUTPUT_DIR"
echo ""
echo "Next steps:"
echo "  1. Review captured files for sensitive data"
echo "  2. Validate JSON structure: cd tests/golden && go test -v"
echo "  3. Commit golden files to version control"
echo ""

# Validate all JSON files
echo -e "${BLUE}Validating JSON files...${NC}"
INVALID_COUNT=0
for file in "$OUTPUT_DIR"/*.json; do
    if [ -f "$file" ]; then
        if ! jq empty "$file" 2>/dev/null; then
            echo -e "${RED}✗${NC} Invalid JSON: $(basename $file)"
            INVALID_COUNT=$((INVALID_COUNT + 1))
        fi
    fi
done

if [ $INVALID_COUNT -eq 0 ]; then
    echo -e "${GREEN}✓${NC} All JSON files are valid"
else
    echo -e "${RED}✗${NC} Found $INVALID_COUNT invalid JSON file(s)"
    exit 1
fi

