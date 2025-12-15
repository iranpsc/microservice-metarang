package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"metargb/financial-service/internal/models"
	"metargb/financial-service/internal/parsian"
	"metargb/financial-service/internal/repository"
)

var (
	ErrInvalidAmount   = errors.New("amount must be at least 1")
	ErrInvalidAsset    = errors.New("invalid asset type")
	ErrOrderNotFound   = errors.New("order not found")
	ErrPaymentFailed   = errors.New("payment request failed")
	ErrUserNotEligible = errors.New("user not eligible to buy from store")
)

type OrderService interface {
	CreateOrder(ctx context.Context, userID uint64, amount int32, asset string) (string, error)
	HandleCallback(ctx context.Context, orderID uint64, status int32, token int64, additionalParams map[string]string) (string, error)
}

type orderService struct {
	orderRepo       repository.OrderRepository
	transactionRepo repository.TransactionRepository
	paymentRepo     repository.PaymentRepository
	variableRepo    repository.VariableRepository
	firstOrderRepo  repository.FirstOrderRepository
	parsianClient   ParsianClient // Interface for easier testing
	orderPolicy     OrderPolicy
	jalaliConverter JalaliConverter
	merchantID      string
	loanMerchantID  string
	callbackURL     string
	frontendURL     string
	// TODO: Add gRPC clients for wallet and referral services
	// walletClient    commercialpb.WalletServiceClient
	// referralService ReferralService
}

type OrderConfig struct {
	ParsianMerchantID            string
	ParsianLoanAccountMerchantID string
	ParsianCallbackURL           string
	FrontendURL                  string
}

func NewOrderService(
	orderRepo repository.OrderRepository,
	transactionRepo repository.TransactionRepository,
	paymentRepo repository.PaymentRepository,
	variableRepo repository.VariableRepository,
	firstOrderRepo repository.FirstOrderRepository,
	parsianClient ParsianClient,
	orderPolicy OrderPolicy,
	jalaliConverter JalaliConverter,
	config OrderConfig,
) OrderService {
	return &orderService{
		orderRepo:       orderRepo,
		transactionRepo: transactionRepo,
		paymentRepo:     paymentRepo,
		variableRepo:    variableRepo,
		firstOrderRepo:  firstOrderRepo,
		parsianClient:   parsianClient,
		orderPolicy:     orderPolicy,
		jalaliConverter: jalaliConverter,
		merchantID:      config.ParsianMerchantID,
		loanMerchantID:  config.ParsianLoanAccountMerchantID,
		callbackURL:     config.ParsianCallbackURL,
		frontendURL:     config.FrontendURL,
	}
}

func (s *orderService) CreateOrder(ctx context.Context, userID uint64, amount int32, asset string) (string, error) {
	// Validation
	if amount < 1 {
		return "", ErrInvalidAmount
	}

	validAssets := map[string]bool{"psc": true, "irr": true, "red": true, "blue": true, "yellow": true}
	if !validAssets[asset] {
		return "", ErrInvalidAsset
	}

	// Check policy: buyFromStore
	canBuy, err := s.orderPolicy.CanBuyFromStore(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to check buy permission: %w", err)
	}
	if !canBuy {
		return "", ErrUserNotEligible
	}

	// Get conversion rate
	rate, err := s.variableRepo.GetRate(ctx, asset)
	if err != nil {
		return "", fmt.Errorf("failed to get asset rate: %w", err)
	}

	// Create order with default status -138 (pending Parsian verification)
	order := &models.Order{
		UserID: userID,
		Asset:  asset,
		Amount: float64(amount),
		Status: -138, // Default status per documentation
	}

	err = s.orderRepo.Create(ctx, order)
	if err != nil {
		return "", fmt.Errorf("failed to create order: %w", err)
	}

	// Create transaction (morph-one relationship)
	transactionID := fmt.Sprintf("TR-%d", time.Now().UnixNano())
	transaction := &models.Transaction{
		ID:          transactionID,
		UserID:      userID,
		Asset:       asset,
		Amount:      float64(amount),
		Action:      "deposit",
		Status:      -138,
		PayableType: stringPtr("App\\Models\\Order"),
		PayableID:   &order.ID,
	}

	err = s.transactionRepo.Create(ctx, transaction)
	if err != nil {
		return "", fmt.Errorf("failed to create transaction: %w", err)
	}

	// Select merchant ID based on asset
	merchantID := s.merchantID
	if asset == "irr" {
		merchantID = s.loanMerchantID
	}

	// Calculate amount in Rials
	amountInRials := int64(float64(amount) * rate)

	// Send purchase request to Parsian
	params := parsian.RequestParams{
		MerchantID:     merchantID,
		OrderID:        fmt.Sprintf("%d", order.ID),
		Amount:         amountInRials,
		CallbackURL:    s.callbackURL,
		AdditionalData: "",
		Originator:     "",
	}

	response, err := s.parsianClient.RequestPayment(params)
	if err != nil {
		return "", fmt.Errorf("failed to request payment: %w", err)
	}

	// Check if request was successful
	if !response.Success() {
		return "", fmt.Errorf("%w: %s", ErrPaymentFailed, response.Error().Message())
	}

	// Update transaction with token
	transaction.Token = &response.Token
	err = s.transactionRepo.Update(ctx, transaction)
	if err != nil {
		// Log error but don't fail - token is stored
		fmt.Printf("Warning: failed to update transaction with token: %v\n", err)
	}

	// Return payment URL
	return response.URL(), nil
}

func (s *orderService) HandleCallback(ctx context.Context, orderID uint64, status int32, token int64, additionalParams map[string]string) (string, error) {
	// Fetch order with user
	order, _, err := s.orderRepo.FindByIDWithUser(ctx, orderID)
	if err != nil {
		return "", fmt.Errorf("failed to find order: %w", err)
	}
	if order == nil {
		return "", ErrOrderNotFound
	}

	// Find transaction for this order
	transaction, err := s.transactionRepo.FindByPayable(ctx, "App\\Models\\Order", orderID)
	if err != nil {
		return "", fmt.Errorf("failed to find transaction: %w", err)
	}
	if transaction == nil {
		return "", fmt.Errorf("transaction not found for order")
	}

	// Build redirect URL with all query parameters
	redirectURL := s.frontendURL + "/metaverse/payment/verify"
	u, err := url.Parse(redirectURL)
	if err != nil {
		return "", fmt.Errorf("invalid frontend URL: %w", err)
	}

	q := u.Query()
	q.Set("OrderId", fmt.Sprintf("%d", orderID))
	q.Set("status", fmt.Sprintf("%d", status))
	q.Set("Token", fmt.Sprintf("%d", token))

	// Add all additional parameters from Parsian
	for k, v := range additionalParams {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	// If status == 0, verify payment
	if status == 0 {
		// Get rate to calculate amount in Rials
		rate, err := s.variableRepo.GetRate(ctx, order.Asset)
		if err != nil {
			return u.String(), fmt.Errorf("failed to get rate: %w", err)
		}

		// Select merchant ID
		merchantID := s.merchantID
		if order.Asset == "irr" {
			merchantID = s.loanMerchantID
		}

		// Verify payment with Parsian
		verifyParams := parsian.VerificationParams{
			MerchantID: merchantID,
			Token:      token,
		}

		verifyResponse, err := s.parsianClient.VerifyPayment(verifyParams)
		if err != nil {
			// Verification failed - still redirect but don't update order
			return u.String(), nil
		}

		if verifyResponse.Success() {
			// Verification successful
			order.Status = verifyResponse.Status
			err = s.orderRepo.Update(ctx, order)
			if err != nil {
				return u.String(), fmt.Errorf("failed to update order: %w", err)
			}

			// Update transaction
			transaction.Status = verifyResponse.Status
			refID := verifyResponse.ReferenceID
			transaction.RefID = &refID
			err = s.transactionRepo.Update(ctx, transaction)
			if err != nil {
				return u.String(), fmt.Errorf("failed to update transaction: %w", err)
			}

			// Check if user can get bonus
			canGetBonus, err := s.orderPolicy.CanGetBonus(ctx, order.UserID, order.Asset)
			if err != nil {
				return u.String(), fmt.Errorf("failed to check bonus eligibility: %w", err)
			}

			amount := order.Amount * rate

			if canGetBonus {
				// First order bonus: 50%
				bonus := order.Amount * 0.5
				_ = order.Amount + bonus // totalAmount for future gRPC call

				// TODO: Add balance via gRPC call to commercial-service
				// walletClient.AddBalance(ctx, order.UserID, order.Asset, totalAmount)

				// Create first order record
				jalaliDate := s.jalaliConverter.NowJalali()
				firstOrder := &models.FirstOrder{
					UserID: order.UserID,
					Type:   order.Asset,
					Amount: order.Amount,
					Date:   jalaliDate,
					Bonus:  bonus,
				}
				err = s.firstOrderRepo.Create(ctx, firstOrder)
				if err != nil {
					fmt.Printf("Warning: failed to create first order record: %v\n", err)
				}
			} else {
				// Regular order - add only order amount
				// TODO: Add balance via gRPC call to commercial-service
				// walletClient.AddBalance(ctx, order.UserID, order.Asset, order.Amount)
			}

			// Create payment record
			cardPan := additionalParams["CardMaskPan"]
			if cardPan == "" {
				cardPan = additionalParams["card_pan"]
			}
			payment := &models.Payment{
				UserID:  order.UserID,
				RefID:   verifyResponse.ReferenceID,
				CardPan: cardPan,
				Gateway: "parsian",
				Amount:  amount,
				Product: order.Asset,
			}
			err = s.paymentRepo.Create(ctx, payment)
			if err != nil {
				fmt.Printf("Warning: failed to create payment record: %v\n", err)
			}

			// Process referral (only for non-irr assets)
			if order.Asset != "irr" {
				// TODO: Process referral via gRPC call to commercial-service
				// referralService.ProcessReferralCommission(ctx, order.UserID, order)
			}

			// TODO: Send notification and call user.deposit() via gRPC
		} else {
			// Verification failed - update order with status
			order.Status = verifyResponse.Status
			s.orderRepo.Update(ctx, order)
		}
	} else {
		// Payment failed (status != 0)
		order.Status = status
		s.orderRepo.Update(ctx, order)
		transaction.Status = status
		s.transactionRepo.Update(ctx, transaction)
	}

	return u.String(), nil
}

func stringPtr(s string) *string {
	return &s
}
