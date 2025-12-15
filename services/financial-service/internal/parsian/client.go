package parsian

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// Parsian SOAP endpoints - exactly as in Laravel
	saleServiceURL    = "https://pec.shaparak.ir/NewIPGServices/Sale/SaleService.asmx"
	confirmServiceURL = "https://pec.shaparak.ir/NewIPGServices/Confirm/ConfirmService.asmx"
	paymentGatewayURL = "https://pec.shaparak.ir/NewIPG/"
)

// Client handles Parsian payment gateway operations
// Matches Laravel's App\Parsian\Parsian class
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new Parsian client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// RequestParams for payment request
// Matches Laravel's App\Parsian\Request parameters
type RequestParams struct {
	MerchantID     string
	OrderID        string
	Amount         int64
	CallbackURL    string
	AdditionalData string
	Originator     string
}

// RequestResponse is the response from Parsian payment request
// Matches Laravel's App\Parsian\RequestResponse
type RequestResponse struct {
	Status  int32
	Message string
	Token   int64
}

// VerificationParams for payment verification
// Matches Laravel's App\Parsian\Verification parameters
type VerificationParams struct {
	MerchantID string
	Token      int64
}

// VerificationResponse is the response from Parsian verification
// Matches Laravel's App\Parsian\VerificationResponse
type VerificationResponse struct {
	Status      int32
	ReferenceID int64  // RRN in Parsian response
	CardHash    string // Card number masked
}

// RequestPayment initiates a payment request
// Matches Laravel's App\Parsian\Request::send()
func (c *Client) RequestPayment(params RequestParams) (*RequestResponse, error) {
	// Build SOAP envelope - exactly as in Laravel
	additionalData := params.AdditionalData
	originator := params.Originator

	soapEnvelope := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <SalePaymentRequest xmlns="https://pec.Shaparak.ir/NewIPGServices/Sale/SaleService">
      <requestData>
        <LoginAccount>%s</LoginAccount>
        <Amount>%d</Amount>
        <OrderId>%s</OrderId>
        <CallBackUrl>%s</CallBackUrl>
        <AdditionalData>%s</AdditionalData>
        <Originator>%s</Originator>
      </requestData>
    </SalePaymentRequest>
  </soap:Body>
</soap:Envelope>`, params.MerchantID, params.Amount, params.OrderID, params.CallbackURL, additionalData, originator)

	req, err := http.NewRequest("POST", saleServiceURL, bytes.NewBufferString(soapEnvelope))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", "https://pec.Shaparak.ir/NewIPGServices/Sale/SaleService/SalePaymentRequest")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse SOAP response
	var envelope struct {
		Body struct {
			Response struct {
				Result struct {
					Status  int32  `xml:"Status"`
					Message string `xml:"Message"`
					Token   int64  `xml:"Token"`
				} `xml:"SalePaymentRequestResult"`
			} `xml:"SalePaymentRequestResponse"`
		} `xml:"Body"`
	}

	if err := xml.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &RequestResponse{
		Status:  envelope.Body.Response.Result.Status,
		Message: envelope.Body.Response.Result.Message,
		Token:   envelope.Body.Response.Result.Token,
	}, nil
}

// VerifyPayment verifies a payment
// Matches Laravel's App\Parsian\Verification::send()
func (c *Client) VerifyPayment(params VerificationParams) (*VerificationResponse, error) {
	// Build SOAP envelope - exactly as in Laravel
	soapEnvelope := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <ConfirmPayment xmlns="https://pec.Shaparak.ir/NewIPGServices/Confirm/ConfirmService">
      <requestData>
        <LoginAccount>%s</LoginAccount>
        <Token>%d</Token>
      </requestData>
    </ConfirmPayment>
  </soap:Body>
</soap:Envelope>`, params.MerchantID, params.Token)

	req, err := http.NewRequest("POST", confirmServiceURL, bytes.NewBufferString(soapEnvelope))
	if err != nil {
		return nil, fmt.Errorf("failed to create verification request: %w", err)
	}

	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", "https://pec.Shaparak.ir/NewIPGServices/Confirm/ConfirmService/ConfirmPayment")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send verification request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read verification response: %w", err)
	}

	// Parse SOAP response
	var envelope struct {
		Body struct {
			Response struct {
				Result struct {
					Status int32 `xml:"Status"`
					RRN    int64 `xml:"RRN"` // Reference ID in Parsian
				} `xml:"ConfirmPaymentResult"`
			} `xml:"ConfirmPaymentResponse"`
		} `xml:"Body"`
	}

	if err := xml.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse verification response: %w", err)
	}

	return &VerificationResponse{
		Status:      envelope.Body.Response.Result.Status,
		ReferenceID: envelope.Body.Response.Result.RRN,
		CardHash:    "", // CardHash not provided in response, set after if needed
	}, nil
}

// Success checks if the request response indicates success
// Matches Laravel's App\Parsian\RequestResponse::success()
// Success criteria: status === 0 AND token > 0
func (r *RequestResponse) Success() bool {
	return r.Status == 0 && r.Token > 0
}

// URL returns the payment gateway URL for the given token
// Matches Laravel's App\Parsian\RequestResponse::url()
func (r *RequestResponse) URL() string {
	if !r.Success() {
		return ""
	}
	return fmt.Sprintf("%s?Token=%d", paymentGatewayURL, r.Token)
}

// Error returns error information for the request
// Matches Laravel's App\Parsian\RequestResponse::error()
func (r *RequestResponse) Error() *ParsianError {
	return NewParsianError(r.Status)
}

// Success checks if the verification response indicates success
// Matches Laravel's App\Parsian\VerificationResponse::success()
// Success criteria: status === 0 AND referenceId > 0
func (v *VerificationResponse) Success() bool {
	return v.Status == 0 && v.ReferenceID > 0
}

// Error returns error information for the verification
// Matches Laravel's App\Parsian\VerificationResponse::error()
func (v *VerificationResponse) Error() *ParsianError {
	return NewParsianError(v.Status)
}
