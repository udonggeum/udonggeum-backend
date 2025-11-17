package kakaopay

import (
	"encoding/json"
	"fmt"
	"time"
)

// ReadyRequest represents the request parameters for the Ready API
type ReadyRequest struct {
	CID            string `json:"cid"`
	PartnerOrderID string `json:"partner_order_id"`
	PartnerUserID  string `json:"partner_user_id"`
	ItemName       string `json:"item_name"`
	Quantity       int    `json:"quantity"`
	TotalAmount    int64  `json:"total_amount"`
	TaxFreeAmount  int64  `json:"tax_free_amount"`
	ApprovalURL    string `json:"approval_url"`
	FailURL        string `json:"fail_url"`
	CancelURL      string `json:"cancel_url"`
}

// ReadyResponse represents the response from the Ready API
type ReadyResponse struct {
	TID                   string    `json:"tid"`
	NextRedirectAppURL    string    `json:"next_redirect_app_url"`
	NextRedirectMobileURL string    `json:"next_redirect_mobile_url"`
	NextRedirectPCURL     string    `json:"next_redirect_pc_url"`
	AndroidAppScheme      string    `json:"android_app_scheme"`
	IOSAppScheme          string    `json:"ios_app_scheme"`
	CreatedAt             time.Time `json:"created_at"`
}

// ApproveRequest represents the request parameters for the Approve API
type ApproveRequest struct {
	CID            string `json:"cid"`
	TID            string `json:"tid"`
	PartnerOrderID string `json:"partner_order_id"`
	PartnerUserID  string `json:"partner_user_id"`
	PgToken        string `json:"pg_token"`
}

// Amount represents payment amount information
type Amount struct {
	Total        int64 `json:"total"`
	TaxFree      int64 `json:"tax_free"`
	VAT          int64 `json:"vat"`
	Point        int64 `json:"point"`
	Discount     int64 `json:"discount"`
	GreenDeposit int64 `json:"green_deposit"`
}

// CardInfo represents card payment details
type CardInfo struct {
	KakaoPayPurchaseCorp     string `json:"kakaopay_purchase_corp"`
	KakaoPayPurchaseCorpCode string `json:"kakaopay_purchase_corp_code"`
	KakaoPayIssuerCorp       string `json:"kakaopay_issuer_corp"`
	KakaoPayIssuerCorpCode   string `json:"kakaopay_issuer_corp_code"`
	BIN                      string `json:"bin"`
	CardType                 string `json:"card_type"`
	InstallMonth             string `json:"install_month"`
	ApprovedID               string `json:"approved_id"`
	CardMID                  string `json:"card_mid"`
	InterestFreeInstall      string `json:"interest_free_install"`
	InstallmentType          string `json:"installment_type"`
	CardItemCode             string `json:"card_item_code"`
}

// ApproveResponse represents the response from the Approve API
type ApproveResponse struct {
	AID               string     `json:"aid"`
	TID               string     `json:"tid"`
	CID               string     `json:"cid"`
	SID               string     `json:"sid"`
	PartnerOrderID    string     `json:"partner_order_id"`
	PartnerUserID     string     `json:"partner_user_id"`
	PaymentMethodType string     `json:"payment_method_type"`
	Amount            Amount     `json:"amount"`
	CardInfo          *CardInfo  `json:"card_info,omitempty"`
	ItemName          string     `json:"item_name"`
	ItemCode          string     `json:"item_code"`
	Quantity          int        `json:"quantity"`
	CreatedAt         time.Time  `json:"created_at"`
	ApprovedAt        time.Time  `json:"approved_at"`
	Payload           string     `json:"payload"`
}

// UnmarshalJSON custom unmarshal for datetime fields
func (a *ApproveResponse) UnmarshalJSON(data []byte) error {
	type Alias ApproveResponse
	aux := &struct {
		CreatedAt  string `json:"created_at"`
		ApprovedAt string `json:"approved_at"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Parse the created_at time format
	if aux.CreatedAt != "" {
		createdTime, err := time.Parse("2006-01-02T15:04:05", aux.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to parse created_at: %w", err)
		}
		a.CreatedAt = createdTime
	}

	// Parse the approved_at time format
	if aux.ApprovedAt != "" {
		approvedTime, err := time.Parse("2006-01-02T15:04:05", aux.ApprovedAt)
		if err != nil {
			return fmt.Errorf("failed to parse approved_at: %w", err)
		}
		a.ApprovedAt = approvedTime
	}

	return nil
}

// CancelRequest represents the request parameters for the Cancel API
type CancelRequest struct {
	CID               string `json:"cid"`
	TID               string `json:"tid"`
	CancelAmount      int64  `json:"cancel_amount"`
	CancelTaxFreeAmount int64  `json:"cancel_tax_free_amount"`
	CancelVatAmount   int64  `json:"cancel_vat_amount,omitempty"`
}

// CancelResponse represents the response from the Cancel API
type CancelResponse struct {
	AID                      string    `json:"aid"`
	TID                      string    `json:"tid"`
	CID                      string    `json:"cid"`
	Status                   string    `json:"status"`
	PartnerOrderID           string    `json:"partner_order_id"`
	PartnerUserID            string    `json:"partner_user_id"`
	PaymentMethodType        string    `json:"payment_method_type"`
	Amount                   Amount    `json:"amount"`
	ApprovedCancelAmount     Amount    `json:"approved_cancel_amount"`
	CanceledAmount           Amount    `json:"canceled_amount"`
	CancelAvailableAmount    Amount    `json:"cancel_available_amount"`
	ItemName                 string    `json:"item_name"`
	ItemCode                 string    `json:"item_code"`
	Quantity                 int       `json:"quantity"`
	CreatedAt                time.Time `json:"created_at"`
	ApprovedAt               time.Time `json:"approved_at"`
	CanceledAt               time.Time `json:"canceled_at"`
	Payload                  string    `json:"payload"`
}

// UnmarshalJSON custom unmarshal for CancelResponse datetime fields
func (c *CancelResponse) UnmarshalJSON(data []byte) error {
	type Alias CancelResponse
	aux := &struct {
		CreatedAt  string `json:"created_at"`
		ApprovedAt string `json:"approved_at"`
		CanceledAt string `json:"canceled_at"`
		*Alias
	}{
		Alias: (*Alias)(c),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	timeLayout := "2006-01-02T15:04:05"

	// Parse timestamps
	if aux.CreatedAt != "" {
		t, err := time.Parse(timeLayout, aux.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to parse created_at: %w", err)
		}
		c.CreatedAt = t
	}

	if aux.ApprovedAt != "" {
		t, err := time.Parse(timeLayout, aux.ApprovedAt)
		if err != nil {
			return fmt.Errorf("failed to parse approved_at: %w", err)
		}
		c.ApprovedAt = t
	}

	if aux.CanceledAt != "" {
		t, err := time.Parse(timeLayout, aux.CanceledAt)
		if err != nil {
			return fmt.Errorf("failed to parse canceled_at: %w", err)
		}
		c.CanceledAt = t
	}

	return nil
}

// ErrorResponse represents an error response from Kakao Pay API
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
	Extras  map[string]interface{} `json:"extras,omitempty"`
}

func (e *ErrorResponse) Error() string {
	return fmt.Sprintf("kakao pay error: code=%d, msg=%s", e.Code, e.Message)
}
