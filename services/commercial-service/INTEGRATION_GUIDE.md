# Commercial Service - Integration Guide

This guide explains how to wire up the newly implemented commercial service components.

## Dependencies Wiring (main.go)

Update `cmd/server/main.go` to initialize all components:

```go
package main

import (
    "database/sql"
    "log"
    
    "metargb/commercial-service/internal/handler"
    "metargb/commercial-service/internal/parsian"
    "metargb/commercial-service/internal/repository"
    "metargb/commercial-service/internal/service"
)

func main() {
    // Database connection
    db, err := sql.Open("mysql", "dsn...")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Initialize Repositories
    walletRepo := repository.NewWalletRepository(db)
    transactionRepo := repository.NewTransactionRepository(db)
    orderRepo := repository.NewOrderRepository(db)
    paymentRepo := repository.NewPaymentRepository(db)
    variableRepo := repository.NewVariableRepository(db)
    userVariableRepo := repository.NewUserVariableRepository(db)
    referralRepo := repository.NewReferralRepository(db)
    firstOrderRepo := repository.NewFirstOrderRepository(db)

    // Initialize Parsian client
    parsianClient := parsian.NewClient(
        "merchant_id",
        "api_endpoint",
    )

    // Initialize Utilities
    jalaliConverter := service.NewJalaliConverter()

    // Initialize Policy Services
    orderPolicy := service.NewOrderPolicy(firstOrderRepo)

    // Initialize Business Services
    referralService := service.NewReferralService(
        referralRepo,
        variableRepo,
        userVariableRepo,
        walletRepo,
    )

    walletService := service.NewWalletService(walletRepo)

    transactionService := service.NewTransactionService(
        transactionRepo,
        jalaliConverter,
    )

    paymentService := service.NewPaymentService(
        orderRepo,
        transactionRepo,
        paymentRepo,
        walletRepo,
        firstOrderRepo,
        variableRepo,
        parsianClient,
        referralService,
        orderPolicy,
        jalaliConverter,
    )

    // Initialize Handlers
    walletHandler := handler.NewWalletHandler(walletService)
    transactionHandler := handler.NewTransactionHandler(transactionService)
    paymentHandler := handler.NewPaymentHandler(paymentService)

    // Start gRPC server
    // ... server initialization code
}
```

## Handler Updates

### WalletHandler

The wallet handler already returns the correct format thanks to the updated `WalletService.GetWallet()` method. No changes needed.

### TransactionHandler

Update to return `TransactionDTO` instead of raw `Transaction`:

```go
func (h *transactionHandler) ListTransactions(ctx context.Context, req *pb.ListTransactionsRequest) (*pb.ListTransactionsResponse, error) {
    // Convert filters
    filters := make(map[string]interface{})
    if req.Asset != "" {
        filters["asset"] = req.Asset
    }
    if req.Action != "" {
        filters["action"] = req.Action
    }
    // ... more filters

    // Get DTOs (already formatted with Jalali dates)
    dtos, err := h.service.ListTransactions(ctx, req.UserId, filters)
    if err != nil {
        return nil, err
    }

    // Convert DTOs to protobuf
    pbTransactions := make([]*pb.Transaction, len(dtos))
    for i, dto := range dtos {
        pbTransactions[i] = &pb.Transaction{
            Id:     dto.ID,
            Type:   dto.Type,
            Asset:  dto.Asset,
            Amount: dto.Amount,
            Action: dto.Action,
            Status: dto.Status,
            Date:   dto.Date, // Already in Jalali format
            Time:   dto.Time, // Already in Jalali format
        }
    }

    return &pb.ListTransactionsResponse{
        Transactions: pbTransactions,
    }, nil
}
```

### PaymentHandler

The payment handler callback now automatically processes referrals and bonuses:

```go
func (h *paymentHandler) HandleCallback(ctx context.Context, req *pb.CallbackRequest) (*pb.CallbackResponse, error) {
    // Service handles everything:
    // 1. First order bonus check
    // 2. Wallet updates
    // 3. Referral commission
    success, redirectURL, message, err := h.service.HandleCallback(
        ctx,
        req.OrderId,
        req.Status,
        req.Token,
    )

    if err != nil {
        return nil, err
    }

    return &pb.CallbackResponse{
        Success:     success,
        RedirectUrl: redirectURL,
        Message:     message,
    }, nil
}
```

## Database Schema Requirements

Ensure the following tables exist:

### Variables Table
```sql
CREATE TABLE `variables` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `key` VARCHAR(50) NOT NULL,
  `value` DOUBLE NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `variables_key_unique` (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Required rows:
INSERT INTO variables (`key`, value) VALUES
('psc', 1.0),
('red', 1.0),
('blue', 1.0),
('yellow', 1.0);
```

### User Variables Table
```sql
CREATE TABLE `user_variables` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `referral_profit` DOUBLE NOT NULL DEFAULT 0,
  `withdraw_profit` INT NOT NULL DEFAULT 7,
  `created_at` TIMESTAMP NULL DEFAULT NULL,
  `updated_at` TIMESTAMP NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `user_variables_user_id_index` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

### Referral Order Histories Table
```sql
CREATE TABLE `referral_order_histories` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL, -- referrer (receives commission)
  `referral_id` BIGINT UNSIGNED NOT NULL, -- referred user
  `amount` DOUBLE NOT NULL,
  `created_at` TIMESTAMP NULL DEFAULT NULL,
  `updated_at` TIMESTAMP NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `referral_order_histories_user_id_index` (`user_id`),
  KEY `referral_order_histories_referral_id_index` (`referral_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

### First Orders Table
```sql
CREATE TABLE `first_orders` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `type` VARCHAR(20) NOT NULL, -- psc, red, blue, yellow
  `amount` DOUBLE NOT NULL,
  `date` VARCHAR(10) NOT NULL, -- Jalali date Y/m/d
  `bonus` DOUBLE NOT NULL,
  `created_at` TIMESTAMP NULL DEFAULT NULL,
  `updated_at` TIMESTAMP NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `first_orders_user_id_index` (`user_id`),
  KEY `first_orders_type_index` (`type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

### Users Table (referrer_id column)
```sql
-- Ensure users table has referrer_id:
ALTER TABLE `users` ADD COLUMN `referrer_id` BIGINT UNSIGNED NULL AFTER `id`;
ALTER TABLE `users` ADD KEY `users_referrer_id_index` (`referrer_id`);
```

## Configuration

Add to config file (config.yaml or environment variables):

```yaml
commercial:
  referral:
    enabled: true
    commission_rate: 0.5  # 50%
  first_order:
    enabled: true
    bonus_rate: 0.5  # 50%
  parsian:
    merchant_id: "your_merchant_id"
    api_endpoint: "https://pec.shaparak.ir/..."
```

## Testing

### Unit Tests

```go
// Test referral commission calculation
func TestReferralService_ProcessReferralCommission(t *testing.T) {
    // Test PSC order
    // Test color order
    // Test limit enforcement
    // Test IRR exclusion
}

// Test first order bonus
func TestOrderPolicy_CanGetBonus(t *testing.T) {
    // Test first order eligible
    // Test subsequent order not eligible
}

// Test formatting
func TestWalletService_GetWallet(t *testing.T) {
    // Verify compact number formatting
    // Verify satisfaction format
}
```

### Integration Tests

```bash
# Test full payment flow
curl -X POST http://localhost:8080/api/order \
  -H "Content-Type: application/json" \
  -d '{
    "asset": "psc",
    "amount": 1000
  }'

# Verify:
# 1. Order created
# 2. First order bonus applied (if eligible)
# 3. Referral commission paid (if has referrer)
# 4. Wallet formatted correctly
# 5. Dates in Jalali format
```

## Monitoring

Add metrics for:

```go
// Referral metrics
referral_commissions_total
referral_commissions_amount_total
referral_limit_exceeded_total

// First order metrics
first_order_bonus_granted_total
first_order_bonus_amount_total

// Formatting metrics
wallet_requests_total
transaction_requests_total
```

## Troubleshooting

### Common Issues

1. **Referral commission not paid**
   - Check user has referrer_id set
   - Check asset is not 'irr'
   - Check referral limit not exceeded
   - Check variable rates exist in database

2. **First order bonus not applied**
   - Check first_orders table is empty for user+asset
   - Check asset is not 'irr'
   - Check OrderPolicy returns true

3. **Wrong date format**
   - Ensure JalaliConverter is wired correctly
   - Check Gregorian to Jalali conversion
   - Verify format string is Y/m/d and H:m:s

4. **Wallet amounts not formatted**
   - Ensure helpers package is imported
   - Check FormatCompactNumber is called
   - Verify decimal to float conversion

## Next Steps

1. ✅ Wire dependencies in main.go
2. ✅ Update handlers to use new services
3. ⏳ Write unit tests
4. ⏳ Write integration tests
5. ⏳ Add gRPC call to notifications-service
6. ⏳ Add gRPC call to levels-service for scoring
7. ⏳ Deploy to staging
8. ⏳ Validate with production data

---

*Generated: October 30, 2025*
*Commercial Service Integration Guide*

