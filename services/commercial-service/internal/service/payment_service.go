package service

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"metargb/commercial-service/internal/models"
	"metargb/commercial-service/internal/parsian"
	"metargb/commercial-service/internal/repository"
)

type PaymentService interface {
	InitiatePayment(ctx context.Context, userID uint64, asset string, amount float64) (string, uint64, string, error)
	HandleCallback(ctx context.Context, orderID uint64, status int32, token int64) (bool, string, string, error)
	VerifyPayment(ctx context.Context, token int64, merchantID string) (bool, int32, int64, string, string, error)
}

type paymentService struct {
	orderRepo       repository.OrderRepository
	transactionRepo repository.TransactionRepository
	paymentRepo     repository.PaymentRepository
	walletRepo      repository.WalletRepository
	firstOrderRepo  repository.FirstOrderRepository
	variableRepo    repository.VariableRepository
	parsianClient   *parsian.Client
	referralService ReferralService
	orderPolicy     OrderPolicy
	jalaliConverter JalaliConverter
	config          *PaymentConfig
}

// PaymentConfig holds payment-specific configuration
type PaymentConfig struct {
	ParsianMerchantID            string
	ParsianLoanAccountMerchantID string
	ParsianCallbackURL           string
}

func NewPaymentService(
	orderRepo repository.OrderRepository,
	transactionRepo repository.TransactionRepository,
	paymentRepo repository.PaymentRepository,
	walletRepo repository.WalletRepository,
	firstOrderRepo repository.FirstOrderRepository,
	variableRepo repository.VariableRepository,
	parsianClient *parsian.Client,
	referralService ReferralService,
	orderPolicy OrderPolicy,
	jalaliConverter JalaliConverter,
	config *PaymentConfig,
) PaymentService {
	return &paymentService{
		orderRepo:       orderRepo,
		transactionRepo: transactionRepo,
		paymentRepo:     paymentRepo,
		walletRepo:      walletRepo,
		firstOrderRepo:  firstOrderRepo,
		variableRepo:    variableRepo,
		parsianClient:   parsianClient,
		referralService: referralService,
		orderPolicy:     orderPolicy,
		jalaliConverter: jalaliConverter,
		config:          config,
	}
}

func (s *paymentService) InitiatePayment(ctx context.Context, userID uint64, asset string, amount float64) (string, uint64, string, error) {
	// Create order
	order := &models.Order{
		UserID: userID,
		Asset:  asset,
		Amount: amount,
		Status: 0, // Pending
	}

	err := s.orderRepo.Create(ctx, order)
	if err != nil {
		return "", 0, "", fmt.Errorf("failed to create order: %w", err)
	}

	// Create transaction
	transactionID := fmt.Sprintf("TR-%d", time.Now().UnixNano())
	transaction := &models.Transaction{
		ID:     transactionID,
		UserID: userID,
		Asset:  asset,
		Amount: amount,
		Action: "deposit",
		Status: 0, // Pending
	}

	err = s.transactionRepo.Create(ctx, transaction)
	if err != nil {
		return "", 0, "", fmt.Errorf("failed to create transaction: %w", err)
	}

	// Get rate for the asset to convert amount to Rials
	rate, err := s.variableRepo.GetRate(ctx, asset)
	if err != nil {
		return "", 0, "", fmt.Errorf("failed to get asset rate: %w", err)
	}

	amountInRials := int64(amount * rate)

	// Determine merchant ID (regular or loan account)
	// Laravel: $merchantId = $order->asset !== 'irr' ? config('parsian.merchant_id') : config('parsian.loan_account_merchant_id');
	merchantID := s.getMerchantID(asset)

	// Initiate payment request with Parsian
	// Matches Laravel: parsian()->orderId($order->id)->amount($order->amount * $rate)->merchantId($merchantId)->request()->callbackUrl(route('parsian.callback'))->send()
	params := parsian.RequestParams{
		MerchantID:     merchantID,
		OrderID:        fmt.Sprintf("%d", order.ID),
		Amount:         amountInRials,
		CallbackURL:    s.config.ParsianCallbackURL,
		AdditionalData: "",
		Originator:     "",
	}

	response, err := s.parsianClient.RequestPayment(params)
	if err != nil {
		return "", 0, "", fmt.Errorf("failed to request payment: %w", err)
	}

	// Check if request was successful
	if !response.Success() {
		return "", 0, "", fmt.Errorf("payment request failed: %s", response.Error().Message())
	}

	// Update transaction with token
	transaction.Token = &response.Token
	err = s.transactionRepo.Update(ctx, transaction)
	if err != nil {
		return "", 0, "", fmt.Errorf("failed to update transaction with token: %w", err)
	}

	// Return payment URL
	return response.URL(), order.ID, transactionID, nil
}

// getMerchantID returns the appropriate merchant ID based on asset
// Laravel logic from OrderController.php lines 48-50
func (s *paymentService) getMerchantID(asset string) string {
	if asset != "irr" {
		return s.config.ParsianMerchantID // Regular merchant ID
	}
	return s.config.ParsianLoanAccountMerchantID // Loan account merchant ID
}

func (s *paymentService) HandleCallback(ctx context.Context, orderID uint64, status int32, token int64) (bool, string, string, error) {
	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return false, "", "Failed to find order", err
	}
	if order == nil {
		return false, "", "Order not found", fmt.Errorf("order not found")
	}

	redirectURL := "/payment/result"
	message := "Payment processed"

	// Check if status from gateway is success (0)
	// Laravel: if ($request->status == 0)
	if status == 0 { // Success from gateway
		// Get rate to calculate amount in Rials
		rate, err := s.variableRepo.GetRate(ctx, order.Asset)
		if err != nil {
			return false, "", "Failed to get rate", err
		}

		amount := order.Amount * rate

		// Determine merchant ID for verification
		merchantID := s.getMerchantID(order.Asset)

		// Verify payment with Parsian
		// Matches Laravel: parsian()->token($transaction->token)->merchantId($merchantId)->verification()->send()
		verifyParams := parsian.VerificationParams{
			MerchantID: merchantID,
			Token:      token,
		}

		verifyResponse, err := s.parsianClient.VerifyPayment(verifyParams)
		if err != nil {
			return false, "", "Failed to verify payment", err
		}

		// Check if verification was successful
		if !verifyResponse.Success() {
			// Verification failed
			order.Status = verifyResponse.Status
			s.orderRepo.Update(ctx, order)

			// Update transaction
			// TODO: Get transaction by order_id and update status

			return false, "", verifyResponse.Error().Message(), nil
		}

		// Verification successful - update order status
		order.Status = verifyResponse.Status
		err = s.orderRepo.Update(ctx, order)
		if err != nil {
			return false, "", "Failed to update order", err
		}

		// Update transaction with reference ID and status
		// TODO: Get transaction by order_id and update with ref_id and status

		// Create payment record
		// Matches Laravel OrderController.php lines 129-136
		payment := &models.Payment{
			UserID:  order.UserID,
			RefID:   verifyResponse.ReferenceID,
			CardPan: verifyResponse.CardHash,
			Gateway: "parsian",
			Amount:  amount,
			Product: order.Asset,
		}

		err = s.paymentRepo.Create(ctx, payment)
		if err != nil {
			// Log error but don't fail the transaction
			fmt.Printf("Warning: failed to create payment record: %v\n", err)
		}

		message = "Payment successful"

		// Check if user can get first order bonus
		canGetBonus, err := s.orderPolicy.CanGetBonus(ctx, order.UserID, order.Asset)
		if err != nil {
			return false, "", "Failed to check bonus eligibility", err
		}

		if canGetBonus {
			// User gets 50% bonus on first order
			bonus := order.Amount * 0.5
			totalAmount := order.Amount + bonus

			// Add order amount + bonus to wallet
			totalAmountDec := decimal.NewFromFloat(totalAmount)
			err = s.walletRepo.AddBalance(ctx, order.UserID, order.Asset, totalAmountDec)
			if err != nil {
				return false, "", "Failed to add balance with bonus", err
			}

			// Get current Jalali date
			jalaliDate := s.jalaliConverter.NowJalali()

			// Create first order record
			firstOrder := &models.FirstOrder{
				UserID: order.UserID,
				Type:   order.Asset,
				Amount: order.Amount,
				Date:   jalaliDate,
				Bonus:  bonus,
			}

			err = s.firstOrderRepo.Create(ctx, firstOrder)
			if err != nil {
				// Log error but don't fail the transaction
				fmt.Printf("Warning: failed to create first order record: %v\n", err)
			}
		} else {
			// Regular order - add only order amount
			amountDec := decimal.NewFromFloat(order.Amount)
			err = s.walletRepo.AddBalance(ctx, order.UserID, order.Asset, amountDec)
			if err != nil {
				return false, "", "Failed to add balance", err
			}
		}

		// Process referral commission (only if asset is not IRR)
		if order.Asset != "irr" {
			err = s.referralService.ProcessReferralCommission(ctx, order.UserID, order)
			if err != nil {
				// Log error but don't fail the transaction
				fmt.Printf("Warning: failed to process referral commission: %v\n", err)
			}
		}

		// TODO: Send notification (requires gRPC call to notifications service)
		// user->notify(new TransactionNotification($order));

		// TODO: Call user.deposit() for score tracking (requires gRPC call to levels service)
		// $user->deposit();

		return true, redirectURL, message, nil
	} else {
		// Payment failed
		order.Status = status
		s.orderRepo.Update(ctx, order)

		// TODO: Update transaction status

		message = "Payment failed"
		return false, redirectURL, message, nil
	}
}

func (s *paymentService) VerifyPayment(ctx context.Context, token int64, merchantID string) (bool, int32, int64, string, string, error) {
	// Verify payment with Parsian
	// Matches Laravel: parsian()->token($transaction->token)->merchantId($merchantId)->verification()->send()
	params := parsian.VerificationParams{
		MerchantID: merchantID,
		Token:      token,
	}

	response, err := s.parsianClient.VerifyPayment(params)
	if err != nil {
		return false, 0, 0, "", fmt.Sprintf("Verification failed: %s", err.Error()), err
	}

	// Check if verification was successful
	success := response.Success()
	message := "Payment verified successfully"
	if !success {
		message = response.Error().Message()
	}

	return success, response.Status, response.ReferenceID, response.CardHash, message, nil
}
