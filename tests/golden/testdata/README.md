## Golden JSON Test Data

This directory contains "golden" JSON responses captured from the Laravel monolith. These serve as the source of truth for API compatibility testing.

### Directory Structure

```
testdata/
├── README.md                 # This file
├── auth_get_me.json         # /api/auth/me response
├── user_wallet.json         # /api/user/wallet response
├── features_list.json       # /api/features response
├── feature_detail.json      # /api/features/{id} response
├── transactions_list.json   # /api/user/transactions response
├── user_level.json          # /api/users/{id}/levels response
├── notifications_list.json  # /api/notifications response
└── actual/                  # Directory for actual microservice responses
    └── (generated at runtime)
```

### Capturing Golden Responses

To capture golden responses from the Laravel monolith:

```bash
# 1. Ensure Laravel app is running
cd /path/to/laravel-app
php artisan serve

# 2. Run the capture script
./scripts/capture_golden_responses.sh

# This will:
# - Authenticate with test credentials
# - Call each endpoint
# - Save responses to testdata/*.json
# - Pretty-format JSON for readability
```

### Manual Capture

You can also manually capture responses:

```bash
# Get auth token first
TOKEN=$(curl -s -X POST http://localhost:8000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"test_user","password":"password"}' | jq -r '.token')

# Capture /api/auth/me
curl -s -X POST http://localhost:8000/api/auth/me \
  -H "Authorization: Bearer $TOKEN" \
  | jq '.' > testdata/auth_get_me.json

# Capture /api/user/wallet
curl -s -X GET http://localhost:8000/api/user/wallet \
  -H "Authorization: Bearer $TOKEN" \
  | jq '.' > testdata/user_wallet.json

# Capture /api/features
curl -s -X GET http://localhost:8000/api/features \
  -H "Authorization: Bearer $TOKEN" \
  | jq '.' > testdata/features_list.json

# ... and so on for each endpoint
```

### Important Notes

1. **Deterministic Data**: Golden responses should use consistent test data. Before capturing:
   - Reset test database to known state
   - Use fixed timestamps (not "2 hours ago" or relative times)
   - Use consistent user IDs, feature IDs, etc.

2. **Sensitive Data**: Remove or mask sensitive data before committing:
   ```bash
   # Replace actual tokens with placeholders
   sed -i 's/"token": ".*"/"token": "REDACTED"/g' testdata/*.json
   ```

3. **Field Order**: JSON field order should be preserved. Use `jq` with `-S` for sorted keys if needed:
   ```bash
   jq -S '.' < unsorted.json > sorted.json
   ```

4. **Jalali Dates**: Ensure Jalali date formats are captured correctly:
   ```json
   {
     "created_at": "1402/10/15 14:30",
     "updated_at": "1402/10/16 09:15"
   }
   ```

5. **Number Formats**: Verify string-formatted numbers are preserved:
   ```json
   {
     "wallet": {
       "psc": "12345.6789012345",
       "rgb": "0.0000000000"
     },
     "feature": {
       "price_psc": "1000",
       "price_irr": "50000000"
     }
   }
   ```

### Test Data Requirements

For comprehensive golden tests, ensure test database includes:

- Users with various states (verified, unverified, suspended)
- Wallets with different balances
- Features with various properties (for sale, not for sale, owned, unowned)
- Transactions of all types (purchase, sell, deposit, withdrawal)
- Notifications (read, unread, various types)
- Dynasty and family structures
- Level progression data
- Support tickets and responses

### Updating Golden Files

When Laravel API intentionally changes (new features, fixes):

1. Document the change in CHANGELOG.md
2. Re-capture affected golden files
3. Update tests if field names or structures changed
4. Ensure microservices match new golden files

### Validation

Before committing golden files, validate they match expected schema:

```bash
# Run golden tests
cd tests/golden
go test -v

# Check for invalid JSON
for file in testdata/*.json; do
  jq empty "$file" || echo "Invalid JSON: $file"
done
```

### Example Golden Response

```json
{
  "success": true,
  "message": "عملیات با موفقیت انجام شد",
  "data": {
    "id": 123,
    "username": "test_user",
    "email": "test@example.com",
    "wallet": {
      "psc": "10000.0000000000",
      "rgb": "500.5000000000"
    },
    "created_at": "1402/08/15 10:30",
    "last_seen": "1402/10/25 15:45"
  }
}
```

### Troubleshooting

**Golden test fails but looks identical:**
- Check whitespace differences
- Verify JSON field ordering
- Look for invisible characters (BOM, zero-width spaces)

**Date/time mismatches:**
- Ensure test uses fixed timestamps
- Check timezone settings (Laravel vs microservice)
- Verify Jalali calendar implementation matches

**Number format mismatches:**
- Check DECIMAL precision settings
- Verify string vs number type for prices/IDs
- Ensure rounding behavior matches
