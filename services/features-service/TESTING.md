# Testing Guide for Features Service

This document describes how to test the Features Service, including unit tests, integration tests, and end-to-end tests.

## Test Structure

```
tests/
├── features-service/
│   ├── internal/
│   │   ├── repository/      # Repository unit tests
│   │   ├── service/         # Service integration tests
│   │   └── handler/         # Handler end-to-end tests
```

## Running Tests

### All Tests

```bash
make test-all
```

### Unit Tests Only

```bash
make test-unit
```

### Integration Tests Only

```bash
make test-integration
```

## Repository Tests

Repository tests use mocked database connections or test databases.

### Example: Buy Request Repository Test

```go
func TestBuyRequestRepository_Create(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)
    defer db.Close()
    
    repo := repository.NewBuyRequestRepository(db)
    
    // Test create
    id, err := repo.Create(ctx, buyerID, sellerID, featureID, note, pricePSC, priceIRR)
    assert.NoError(t, err)
    assert.Greater(t, id, uint64(0))
}
```

## Service Tests

Service tests mock external dependencies (CommercialClient, NotificationClient, EventBroadcaster).

### Example: Marketplace Service Test

```go
func TestMarketplaceService_SendBuyRequest(t *testing.T) {
    // Mock dependencies
    mockCommercialClient := &MockCommercialClient{}
    mockNotificationClient := &MockNotificationClient{}
    mockEventBroadcaster := &MockEventBroadcaster{}
    
    // Create service with mocks
    service := service.NewMarketplaceService(
        // ... repositories ...
        mockCommercialClient,
        mockNotificationClient,
        mockEventBroadcaster,
        // ...
    )
    
    // Test SendBuyRequest
    req := &pb.SendBuyRequestRequest{
        BuyerId: 1,
        FeatureId: 100,
        PricePsc: "100.0",
        PriceIrr: "1000000.0",
    }
    
    result, err := service.SendBuyRequest(ctx, req)
    assert.NoError(t, err)
    assert.NotNil(t, result)
    
    // Verify mocks were called
    assert.True(t, mockCommercialClient.DeductBalanceCalled)
    assert.True(t, mockNotificationClient.SendBuyRequestNotificationCalled)
}
```

### Mock Setup

Create mock implementations for:
- `CommercialClient`: Mock wallet operations
- `NotificationClient`: Mock notification sending
- `EventBroadcaster`: Mock event publishing

## Handler Tests

Handler tests verify gRPC request/response conversion and error mapping.

### Example: Marketplace Handler Test

```go
func TestMarketplaceHandler_SendBuyRequest(t *testing.T) {
    // Setup
    mockService := &MockMarketplaceService{}
    handler := handler.NewMarketplaceHandler(mockService, ...)
    
    // Test request
    req := &pb.SendBuyRequestRequest{
        BuyerId: 1,
        FeatureId: 100,
        PricePsc: "100.0",
        PriceIrr: "1000000.0",
    }
    
    resp, err := handler.SendBuyRequest(ctx, req)
    
    // Assertions
    assert.NoError(t, err)
    assert.NotNil(t, resp)
    assert.Equal(t, uint64(1), resp.BuyRequest.BuyerId)
}
```

## Test Scenarios

### Buy Request Flow

1. **SendBuyRequest**:
   - Validates floor price
   - Checks buyer balance
   - Locks assets
   - Sends notifications

2. **AcceptBuyRequest**:
   - Validates seller ownership
   - Checks underpriced restriction
   - Transfers ownership
   - Updates feature status
   - Sends notifications

3. **RejectBuyRequest**:
   - Validates seller ownership
   - Refunds buyer
   - Deletes request

### Sell Request Flow

1. **CreateSellRequest**:
   - Validates ownership
   - Checks age-based pricing limits
   - Updates feature status
   - Sends notification

2. **DeleteSellRequest**:
   - Validates ownership
   - Reverts feature status
   - Broadcasts event

### Buy Feature Flow

1. **Limited Feature**:
   - Checks color balance
   - Deducts color
   - Transfers ownership

2. **RGB Purchase**:
   - Checks color balance
   - Deducts color
   - Credits RGB

3. **User-to-User Purchase**:
   - Checks PSC/IRR balance
   - Calculates fees
   - Transfers ownership
   - Updates profits

## Mock Dependencies

### Mock Commercial Client

```go
type MockCommercialClient struct {
    DeductBalanceCalled bool
    AddBalanceCalled bool
    CheckBalanceResult bool
}

func (m *MockCommercialClient) DeductBalance(ctx context.Context, userID uint64, asset string, amount float64) error {
    m.DeductBalanceCalled = true
    return nil
}
```

### Mock Notification Client

```go
type MockNotificationClient struct {
    SendBuyRequestNotificationCalled bool
}

func (m *MockNotificationClient) SendBuyRequestNotification(ctx context.Context, userID uint64, notificationType string, buyRequestID, featureID uint64, pricePSC, priceIRR float64) error {
    m.SendBuyRequestNotificationCalled = true
    return nil
}
```

## Test Database Setup

For integration tests, use a test database:

```go
func setupTestDB(t *testing.T) *sql.DB {
    db, err := sql.Open("mysql", "test_dsn")
    if err != nil {
        t.Fatal(err)
    }
    
    // Run migrations or seed data
    // ...
    
    return db
}
```

## Coverage Goals

- Repository layer: >80% coverage
- Service layer: >80% coverage
- Handler layer: >70% coverage

## Continuous Integration

Tests run automatically on:
- Pull requests
- Commits to main branch
- Nightly builds

See `.github/workflows/test.yml` for CI configuration.

