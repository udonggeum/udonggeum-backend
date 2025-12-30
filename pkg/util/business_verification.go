package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// BusinessVerificationRequest 사업자 등록번호 진위확인 요청 구조체
type BusinessVerificationRequest struct {
	BusinessNumber    string `json:"b_no"`       // 사업자등록번호 (10자리)
	StartDate         string `json:"start_dt"`   // 개업일자 (YYYYMMDD)
	RepresentativeName string `json:"p_nm"`       // 대표자명
}

// BusinessVerificationResponse 사업자 등록번호 진위확인 응답 구조체
type BusinessVerificationResponse struct {
	RequestCount int                          `json:"request_cnt"`
	StatusCode   string                       `json:"status_code"`
	Data         []BusinessVerificationData   `json:"data"`
}

type BusinessVerificationData struct {
	BusinessNumber string                    `json:"b_no"`
	Valid          string                    `json:"valid"`          // "01": 확인, "02": 미확인
	ValidMessage   string                    `json:"valid_msg"`
	RequestParam   BusinessVerificationRequest `json:"request_param"`
	Status         *BusinessStatus           `json:"status"`
}

type BusinessStatus struct {
	BusinessStatus     string `json:"b_stt"`      // 사업자 상태 (계속사업자, 휴업자, 폐업자)
	BusinessStatusCode string `json:"b_stt_cd"`   // 01: 계속사업자, 02: 휴업자, 03: 폐업자
	TaxType            string `json:"tax_type"`   // 과세 유형 (일반과세자, 간이과세자)
	TaxTypeCode        string `json:"tax_type_cd"`
	EndDate            string `json:"end_dt"`     // 폐업일 (YYYYMMDD)
	UtccYn             string `json:"utcc_yn"`    // 단위과세전환사업자 여부
	TaxTypeChangeDate  string `json:"tax_type_change_dt"` // 과세 유형 전환일자
	InvoiceApplyDate   string `json:"invoice_apply_dt"`   // 전자(세금)계산서 적용일자
}

// BusinessVerificationResult 사업자 인증 결과
type BusinessVerificationResult struct {
	IsValid            bool   `json:"is_valid"`
	BusinessStatus     string `json:"business_status"`
	BusinessStatusCode string `json:"business_status_code"`
	TaxType            string `json:"tax_type"`
	Message            string `json:"message"`
}

// VerifyBusinessNumber 사업자 등록번호 진위 확인
func VerifyBusinessNumber(businessNumber, startDate, representativeName string) (*BusinessVerificationResult, error) {
	// 환경 변수에서 API 키 가져오기
	apiKey := os.Getenv("BUSINESS_VERIFICATION_API_KEY")
	if apiKey == "" {
		// API 키가 없으면 개발 모드로 간주하고 자동 승인 (실제 운영에서는 필수)
		return &BusinessVerificationResult{
			IsValid:            true,
			BusinessStatus:     "계속사업자",
			BusinessStatusCode: "01",
			TaxType:            "일반과세자",
			Message:            "개발 모드: 자동 승인",
		}, nil
	}

	// 요청 데이터 생성
	requestData := []BusinessVerificationRequest{
		{
			BusinessNumber:    businessNumber,
			StartDate:         startDate,
			RepresentativeName: representativeName,
		},
	}

	// JSON 인코딩
	requestBody, err := json.Marshal(map[string]interface{}{
		"businesses": requestData,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// API 요청
	apiURL := fmt.Sprintf("https://api.odcloud.kr/api/nts-businessman/v1/validate?serviceKey=%s", apiKey)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// HTTP 클라이언트 생성 및 요청 전송
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 응답 읽기
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// HTTP 상태 코드 확인
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned non-200 status code: %d, body: %s", resp.StatusCode, string(body))
	}

	// JSON 디코딩
	var apiResponse BusinessVerificationResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// 응답 데이터 검증
	if len(apiResponse.Data) == 0 {
		return nil, fmt.Errorf("no data in API response")
	}

	data := apiResponse.Data[0]

	// 결과 생성
	result := &BusinessVerificationResult{
		IsValid: data.Valid == "01",
		Message: data.ValidMessage,
	}

	// 사업자 상태 정보 추가
	if data.Status != nil {
		result.BusinessStatus = data.Status.BusinessStatus
		result.BusinessStatusCode = data.Status.BusinessStatusCode
		result.TaxType = data.Status.TaxType

		// 계속사업자가 아니면 등록 불가
		if data.Status.BusinessStatusCode != "01" {
			result.IsValid = false
			if data.Status.BusinessStatusCode == "02" {
				result.Message = "휴업 중인 사업자입니다"
			} else if data.Status.BusinessStatusCode == "03" {
				result.Message = "폐업한 사업자입니다"
			}
		}
	}

	return result, nil
}
