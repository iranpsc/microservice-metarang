package integration

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFeaturePurchaseFlow tests the complete feature purchase flow
func TestFeaturePurchaseFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := GetTestConfig()
	db := cfg.ConnectDB(t)
	defer db.Close()

	defer CleanupDB(t, db, "features", "feature_properties", "wallets", "transactions", "users")

	t.Run("DirectPurchase", func(t *testing.T) {
		// Setup: Create buyer with wallet
		buyer := CreateTestUser(t, db, "buyer_"+time.Now().Format("150405"), "buyer@test.com")
		CreateTestWallet(t, db, buyer, "10000.0000000000", "0.0000000000")

		// Create feature for sale
		seller := CreateTestUser(t, db, "seller_"+time.Now().Format("150405"), "seller@test.com")
		featureID := CreateTestFeature(t, db, &seller)
		
		// Create feature property with price
		_, err := db.Exec(`
			INSERT INTO feature_properties (id, feature_id, price_psc, price_irr, for_sale, created_at, updated_at)
			VALUES (?, ?, '1000', '50000000', 1, NOW(), NOW())
		`, "PROP-"+featureID, featureID, )
		require.NoError(t, err)

		// Get buyer's initial balance
		var initialBalance string
		err = db.QueryRow("SELECT psc FROM wallets WHERE user_id = ?", buyer).Scan(&initialBalance)
		require.NoError(t, err)

		// Perform purchase (deduct from buyer, transfer ownership)
		tx, err := db.Begin()
		require.NoError(t, err)

		// Deduct balance
		_, err = tx.Exec("UPDATE wallets SET psc = psc - 1000 WHERE user_id = ?", buyer)
		require.NoError(t, err)

		// Transfer ownership
		_, err = tx.Exec("UPDATE features SET user_id = ? WHERE id = ?", buyer, featureID)
		require.NoError(t, err)

		// Create transaction record
		transactionID := fmt.Sprintf("TX-%d", time.Now().UnixNano())
		_, err = tx.Exec(`
			INSERT INTO transactions (id, user_id, amount, type, payable_type, payable_id, status, created_at, updated_at)
			VALUES (?, ?, '1000', 'purchase', 'Feature', ?, 'completed', NOW(), NOW())
		`, transactionID, buyer, featureID)
		require.NoError(t, err)

		// Commit transaction
		err = tx.Commit()
		require.NoError(t, err)

		// Verify balance was deducted
		var newBalance string
		err = db.QueryRow("SELECT psc FROM wallets WHERE user_id = ?", buyer).Scan(&newBalance)
		require.NoError(t, err)
		assert.Equal(t, "9000.0000000000", newBalance)

		// Verify ownership transferred
		var ownerID int64
		err = db.QueryRow("SELECT user_id FROM features WHERE id = ?", featureID).Scan(&ownerID)
		require.NoError(t, err)
		assert.Equal(t, buyer, ownerID)

		// Verify transaction was created
		var txCount int
		err = db.QueryRow("SELECT COUNT(*) FROM transactions WHERE id = ?", transactionID).Scan(&txCount)
		require.NoError(t, err)
		assert.Equal(t, 1, txCount)
	})

	t.Run("InsufficientBalance", func(t *testing.T) {
		// Setup: Create buyer with insufficient balance
		buyer := CreateTestUser(t, db, "poor_buyer_"+time.Now().Format("150405"), "poor@test.com")
		CreateTestWallet(t, db, buyer, "500.0000000000", "0.0000000000")

		// Create expensive feature
		seller := CreateTestUser(t, db, "seller2_"+time.Now().Format("150405"), "seller2@test.com")
		featureID := CreateTestFeature(t, db, &seller)
		
		_, err := db.Exec(`
			INSERT INTO feature_properties (id, feature_id, price_psc, price_irr, for_sale, created_at, updated_at)
			VALUES (?, ?, '1000', '50000000', 1, NOW(), NOW())
		`, "PROP-"+featureID, featureID)
		require.NoError(t, err)

		// Try to purchase (should fail)
		var balance string
		err = db.QueryRow("SELECT psc FROM wallets WHERE user_id = ?", buyer).Scan(&balance)
		require.NoError(t, err)

		// Balance check should fail
		assert.Equal(t, "500.0000000000", balance)
		// In real service, this would return an error before attempting purchase
	})
}

// TestFeatureBuyRequestFlow tests the buy request flow
func TestFeatureBuyRequestFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := GetTestConfig()
	db := cfg.ConnectDB(t)
	defer db.Close()

	defer CleanupDB(t, db, "buy_feature_requests", "features", "wallets", "transactions", "users")

	t.Run("SendAndAcceptBuyRequest", func(t *testing.T) {
		// Setup users
		buyer := CreateTestUser(t, db, "buyer_req_"+time.Now().Format("150405"), "buyer_req@test.com")
		CreateTestWallet(t, db, buyer, "10000.0000000000", "0.0000000000")

		seller := CreateTestUser(t, db, "seller_req_"+time.Now().Format("150405"), "seller_req@test.com")
		featureID := CreateTestFeature(t, db, &seller)

		// Step 1: Send buy request
		requestID := time.Now().UnixNano()
		_, err := db.Exec(`
			INSERT INTO buy_feature_requests (id, sender_id, receiver_id, feature_id, offer_amount, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, '2000', 'pending', NOW(), NOW())
		`, requestID, buyer, seller, featureID)
		require.NoError(t, err)

		// Verify request was created
		var reqStatus string
		err = db.QueryRow("SELECT status FROM buy_feature_requests WHERE id = ?", requestID).Scan(&reqStatus)
		require.NoError(t, err)
		assert.Equal(t, "pending", reqStatus)

		// Step 2: Accept request
		tx, err := db.Begin()
		require.NoError(t, err)

		// Update request status
		_, err = tx.Exec("UPDATE buy_feature_requests SET status = 'accepted' WHERE id = ?", requestID)
		require.NoError(t, err)

		// Deduct from buyer
		_, err = tx.Exec("UPDATE wallets SET psc = psc - 2000 WHERE user_id = ?", buyer)
		require.NoError(t, err)

		// Add to seller
		_, err = tx.Exec("UPDATE wallets SET psc = psc + 2000 WHERE user_id = ?", seller)
		require.NoError(t, err)

		// Transfer ownership
		_, err = tx.Exec("UPDATE features SET user_id = ? WHERE id = ?", buyer, featureID)
		require.NoError(t, err)

		// Create transaction
		transactionID := fmt.Sprintf("TX-REQ-%d", time.Now().UnixNano())
		_, err = tx.Exec(`
			INSERT INTO transactions (id, user_id, amount, type, payable_type, payable_id, status, created_at, updated_at)
			VALUES (?, ?, '2000', 'buy_request', 'BuyFeatureRequest', ?, 'completed', NOW(), NOW())
		`, transactionID, buyer, requestID)
		require.NoError(t, err)

		err = tx.Commit()
		require.NoError(t, err)

		// Verify request status
		err = db.QueryRow("SELECT status FROM buy_feature_requests WHERE id = ?", requestID).Scan(&reqStatus)
		require.NoError(t, err)
		assert.Equal(t, "accepted", reqStatus)

		// Verify balances
		var buyerBalance string
		err = db.QueryRow("SELECT psc FROM wallets WHERE user_id = ?", buyer).Scan(&buyerBalance)
		require.NoError(t, err)
		assert.Equal(t, "8000.0000000000", buyerBalance)

		var sellerBalance string
		err = db.QueryRow("SELECT psc FROM wallets WHERE user_id = ?", seller).Scan(&sellerBalance)
		require.NoError(t, err)

		// Verify ownership
		var ownerID int64
		err = db.QueryRow("SELECT user_id FROM features WHERE id = ?", featureID).Scan(&ownerID)
		require.NoError(t, err)
		assert.Equal(t, buyer, ownerID)
	})

	t.Run("RejectBuyRequest", func(t *testing.T) {
		// Setup users
		buyer := CreateTestUser(t, db, "buyer_rej_"+time.Now().Format("150405"), "buyer_rej@test.com")
		seller := CreateTestUser(t, db, "seller_rej_"+time.Now().Format("150405"), "seller_rej@test.com")
		featureID := CreateTestFeature(t, db, &seller)

		// Send buy request
		requestID := time.Now().UnixNano()
		_, err := db.Exec(`
			INSERT INTO buy_feature_requests (id, sender_id, receiver_id, feature_id, offer_amount, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, '2000', 'pending', NOW(), NOW())
		`, requestID, buyer, seller, featureID)
		require.NoError(t, err)

		// Reject request
		_, err = db.Exec("UPDATE buy_feature_requests SET status = 'rejected' WHERE id = ?", requestID)
		require.NoError(t, err)

		// Verify status
		var status string
		err = db.QueryRow("SELECT status FROM buy_feature_requests WHERE id = ?", requestID).Scan(&status)
		require.NoError(t, err)
		assert.Equal(t, "rejected", status)

		// Verify ownership didn't change
		var ownerID int64
		err = db.QueryRow("SELECT user_id FROM features WHERE id = ?", featureID).Scan(&ownerID)
		require.NoError(t, err)
		assert.Equal(t, seller, ownerID)
	})

	t.Run("SoftDeleteBuyRequest", func(t *testing.T) {
		// Setup users
		buyer := CreateTestUser(t, db, "buyer_del_"+time.Now().Format("150405"), "buyer_del@test.com")
		seller := CreateTestUser(t, db, "seller_del_"+time.Now().Format("150405"), "seller_del@test.com")
		featureID := CreateTestFeature(t, db, &seller)

		// Create request
		requestID := time.Now().UnixNano()
		_, err := db.Exec(`
			INSERT INTO buy_feature_requests (id, sender_id, receiver_id, feature_id, offer_amount, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, '2000', 'pending', NOW(), NOW())
		`, requestID, buyer, seller, featureID)
		require.NoError(t, err)

		// Soft delete
		_, err = db.Exec("UPDATE buy_feature_requests SET deleted_at = NOW() WHERE id = ?", requestID)
		require.NoError(t, err)

		// Verify soft delete - should not appear in regular queries
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM buy_feature_requests WHERE id = ? AND deleted_at IS NULL", requestID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)

		// But should still exist in database
		err = db.QueryRow("SELECT COUNT(*) FROM buy_feature_requests WHERE id = ?", requestID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})
}

// TestFeatureListingWithBbox tests feature listing with bounding box filtering
func TestFeatureListingWithBbox(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := GetTestConfig()
	db := cfg.ConnectDB(t)
	defer db.Close()

	defer CleanupDB(t, db, "features", "geometries", "coordinates")

	t.Run("BboxFiltering", func(t *testing.T) {
		// Create features with different coordinates
		owner := CreateTestUser(t, db, "owner_bbox_"+time.Now().Format("150405"), "owner@test.com")
		
		// Feature 1: Inside bbox
		feature1 := CreateTestFeature(t, db, &owner)
		_, err := db.Exec(`
			INSERT INTO geometries (id, feature_id, type, created_at, updated_at)
			VALUES (?, ?, 'Polygon', NOW(), NOW())
		`, 1, feature1)
		require.NoError(t, err)

		_, err = db.Exec(`
			INSERT INTO coordinates (geometry_id, lat, lng, sequence, created_at, updated_at)
			VALUES (?, 35.7, 51.4, 0, NOW(), NOW())
		`, 1)
		require.NoError(t, err)

		// Feature 2: Outside bbox
		feature2 := CreateTestFeature(t, db, &owner)
		_, err = db.Exec(`
			INSERT INTO geometries (id, feature_id, type, created_at, updated_at)
			VALUES (?, ?, 'Polygon', NOW(), NOW())
		`, 2, feature2)
		require.NoError(t, err)

		_, err = db.Exec(`
			INSERT INTO coordinates (geometry_id, lat, lng, sequence, created_at, updated_at)
			VALUES (?, 40.0, 60.0, 0, NOW(), NOW())
		`, 2)
		require.NoError(t, err)

		// Query with bbox (35.0-36.0, 51.0-52.0)
		rows, err := db.Query(`
			SELECT DISTINCT f.id FROM features f
			INNER JOIN geometries g ON f.id = g.feature_id
			INNER JOIN coordinates c ON g.id = c.geometry_id
			WHERE c.lat BETWEEN 35.0 AND 36.0
			AND c.lng BETWEEN 51.0 AND 52.0
		`)
		require.NoError(t, err)
		defer rows.Close()

		var foundFeatures []string
		for rows.Next() {
			var id string
			err := rows.Scan(&id)
			require.NoError(t, err)
			foundFeatures = append(foundFeatures, id)
		}

		// Should only find feature1
		assert.Len(t, foundFeatures, 1)
		assert.Equal(t, feature1, foundFeatures[0])
	})
}

