package util

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Solapi SMS 요청 구조체
type SolapiMessage struct {
	To   string `json:"to"`
	From string `json:"from"`
	Text string `json:"text"`
}

type SolapiRequest struct {
	Message SolapiMessage `json:"message"`
}

// Solapi HMAC-SHA256 서명 생성
func makeSolapiSignature(date, salt, secretKey string) string {
	message := date + salt
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// SendVerificationSMS sends a verification SMS via Solapi
func SendVerificationSMS(phoneNumber, code string) error {
	apiKey := os.Getenv("SOLAPI_API_KEY")
	apiSecret := os.Getenv("SOLAPI_API_SECRET")
	fromNumber := os.Getenv("SOLAPI_FROM_NUMBER")

	// 개발 모드: Solapi 설정이 없으면 콘솔에 출력만
	if apiKey == "" || apiSecret == "" || fromNumber == "" {
		log.Printf("================================")
		log.Printf("[개발 모드] SMS 인증 활성화")
		log.Printf("휴대폰 인증 코드를 아무거나 입력하세요")
		log.Printf("(실제 SMS는 발송되지 않습니다)")
		log.Printf("================================")
		return nil
	}

	// 전화번호 정규화 (하이픈/공백 제거)
	phoneNumber = strings.ReplaceAll(phoneNumber, "-", "")
	phoneNumber = strings.ReplaceAll(phoneNumber, " ", "")

	// SMS 내용
	text := fmt.Sprintf("[우리동네금은방] 인증번호는 [%s]입니다. 5분 이내에 입력해주세요.", code)

	// 요청 body 구성
	requestBody := SolapiRequest{
		Message: SolapiMessage{
			To:   phoneNumber,
			From: fromNumber,
			Text: text,
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("JSON 인코딩 실패: %v", err)
	}

	// 인증 헤더 구성
	date := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	salt := uuid.New().String()
	signature := makeSolapiSignature(date, salt, apiSecret)

	authHeader := fmt.Sprintf(
		"HMAC-SHA256 apiKey=%s, date=%s, salt=%s, signature=%s",
		apiKey, date, salt, signature,
	)

	// HTTP 요청
	apiURL := "https://api.solapi.com/messages/v4/send"
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("HTTP 요청 생성 실패: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("SMS 발송 요청 실패: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("응답 읽기 실패: %v", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		log.Printf("Solapi API 오류 응답: %s", string(body))
		return fmt.Errorf("SMS 발송 실패 (상태 코드: %d): %s", resp.StatusCode, string(body))
	}

	log.Printf("SMS 발송 완료: %s", phoneNumber)
	return nil
}
