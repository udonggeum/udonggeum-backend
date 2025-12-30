package util

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

// Naver Cloud SENS SMS 요청 구조체
type SENSMessageRequest struct {
	Type        string        `json:"type"`        // SMS or LMS
	From        string        `json:"from"`        // 발신번호
	Content     string        `json:"content"`     // 기본 메시지 내용
	Messages    []SENSMessage `json:"messages"`    // 수신자 정보
	Subject     string        `json:"subject,omitempty"` // LMS 제목
	ContentType string        `json:"contentType,omitempty"` // COMM or AD
}

type SENSMessage struct {
	To      string `json:"to"`      // 수신번호
	Content string `json:"content,omitempty"` // 개별 메시지 (optional)
}

// Naver Cloud SENS 시그니처 생성
func makeSignature(method, uri, timestamp, accessKey, secretKey string) string {
	space := " "
	newLine := "\n"

	message := method + space + uri + newLine + timestamp + newLine + accessKey
	hmacHash := hmac.New(sha256.New, []byte(secretKey))
	hmacHash.Write([]byte(message))
	signature := base64.StdEncoding.EncodeToString(hmacHash.Sum(nil))

	return signature
}

// SendVerificationSMS sends a verification SMS via Naver Cloud SENS
func SendVerificationSMS(phoneNumber, code string) error {
	// Naver Cloud SENS 설정
	serviceID := os.Getenv("NAVER_SENS_SERVICE_ID")
	accessKey := os.Getenv("NAVER_SENS_ACCESS_KEY")
	secretKey := os.Getenv("NAVER_SENS_SECRET_KEY")
	fromNumber := os.Getenv("NAVER_SENS_FROM_NUMBER")

	// 개발 모드: SENS 설정이 없으면 콘솔에 출력만
	if serviceID == "" || accessKey == "" || secretKey == "" || fromNumber == "" {
		log.Printf("================================")
		log.Printf("[개발 모드] SMS 인증 활성화")
		log.Printf("휴대폰 인증 코드를 아무거나 입력하세요")
		log.Printf("(실제 SMS는 발송되지 않습니다)")
		log.Printf("================================")
		return nil
	}

	// SMS 내용
	content := fmt.Sprintf("[우리동네금은방] 인증번호는 [%s]입니다. 5분 이내에 입력해주세요.", code)

	// 요청 body 구성
	requestBody := SENSMessageRequest{
		Type:    "SMS",
		From:    fromNumber,
		Content: content,
		Messages: []SENSMessage{
			{
				To: phoneNumber,
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("JSON 인코딩 실패: %v", err)
	}

	// API 요청 준비
	timestamp := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	uri := fmt.Sprintf("/sms/v2/services/%s/messages", serviceID)
	apiURL := fmt.Sprintf("https://sens.apigw.ntruss.com%s", uri)

	// 시그니처 생성
	signature := makeSignature("POST", uri, timestamp, accessKey, secretKey)

	// HTTP 요청
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("HTTP 요청 생성 실패: %v", err)
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("x-ncp-apigw-timestamp", timestamp)
	req.Header.Set("x-ncp-iam-access-key", accessKey)
	req.Header.Set("x-ncp-apigw-signature-v2", signature)

	// 요청 전송
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("SMS 발송 요청 실패: %v", err)
	}
	defer resp.Body.Close()

	// 응답 확인
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("응답 읽기 실패: %v", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		log.Printf("SENS API 오류 응답: %s", string(body))
		return fmt.Errorf("SMS 발송 실패 (상태 코드: %d): %s", resp.StatusCode, string(body))
	}

	log.Printf("SMS 발송 완료: %s", phoneNumber)
	return nil
}
