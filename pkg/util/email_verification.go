package util

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"net/smtp"
	"os"
	"sync"
	"time"
)

// VerificationCode represents a verification code with expiration
type VerificationCode struct {
	Code      string
	ExpiresAt time.Time
}

// In-memory storage for verification codes
var (
	emailVerificationCodes = make(map[string]VerificationCode)
	phoneVerificationCodes = make(map[string]VerificationCode)
	verificationMutex      sync.RWMutex
)

// GenerateVerificationCode generates a random 6-digit code
func GenerateVerificationCode() (string, error) {
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// StoreEmailVerificationCode stores the verification code for an email
func StoreEmailVerificationCode(email, code string) {
	verificationMutex.Lock()
	defer verificationMutex.Unlock()

	emailVerificationCodes[email] = VerificationCode{
		Code:      code,
		ExpiresAt: time.Now().Add(5 * time.Minute), // 5분 유효
	}
}

// StorePhoneVerificationCode stores the verification code for a phone number
func StorePhoneVerificationCode(phone, code string) {
	verificationMutex.Lock()
	defer verificationMutex.Unlock()

	phoneVerificationCodes[phone] = VerificationCode{
		Code:      code,
		ExpiresAt: time.Now().Add(5 * time.Minute), // 5분 유효
	}
}

// VerifyEmailCode verifies the email verification code
func VerifyEmailCode(email, code string) bool {
	verificationMutex.RLock()
	defer verificationMutex.RUnlock()

	storedCode, exists := emailVerificationCodes[email]
	if !exists {
		return false
	}

	// Check if expired
	if time.Now().After(storedCode.ExpiresAt) {
		return false
	}

	// Check if code matches
	if storedCode.Code != code {
		return false
	}

	// Delete the code after successful verification
	verificationMutex.RUnlock()
	verificationMutex.Lock()
	delete(emailVerificationCodes, email)
	verificationMutex.Unlock()
	verificationMutex.RLock()

	return true
}

// VerifyPhoneCode verifies the phone verification code
func VerifyPhoneCode(phone, code string) bool {
	// 개발 모드: SENS 설정이 없으면 모든 코드 허용
	serviceID := os.Getenv("NAVER_SENS_SERVICE_ID")
	accessKey := os.Getenv("NAVER_SENS_ACCESS_KEY")
	secretKey := os.Getenv("NAVER_SENS_SECRET_KEY")
	fromNumber := os.Getenv("NAVER_SENS_FROM_NUMBER")

	if serviceID == "" || accessKey == "" || secretKey == "" || fromNumber == "" {
		// 개발 모드: 6자리 숫자면 모두 허용
		if len(code) == 6 {
			log.Printf("[개발 모드] 휴대폰 인증 성공: %s", phone)
			return true
		}
		return false
	}

	verificationMutex.RLock()
	defer verificationMutex.RUnlock()

	storedCode, exists := phoneVerificationCodes[phone]
	if !exists {
		return false
	}

	// Check if expired
	if time.Now().After(storedCode.ExpiresAt) {
		return false
	}

	// Check if code matches
	if storedCode.Code != code {
		return false
	}

	// Delete the code after successful verification
	verificationMutex.RUnlock()
	verificationMutex.Lock()
	delete(phoneVerificationCodes, phone)
	verificationMutex.Unlock()
	verificationMutex.RLock()

	return true
}

// SendVerificationEmail sends a verification email via Gmail SMTP
func SendVerificationEmail(toEmail, code string) error {
	// Gmail SMTP 설정 (환경변수에서 가져오기)
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"
	fromEmail := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")

	// 개발 모드: SMTP 설정이 없으면 콘솔에 출력만
	if fromEmail == "" || password == "" {
		log.Printf("[DEV MODE] 이메일 인증 코드: %s (받는 사람: %s)", code, toEmail)
		return nil
	}

	// 이메일 본문 구성
	subject := "[우리동네금은방] 이메일 인증 코드"
	body := fmt.Sprintf(`
<html>
<body style="font-family: Arial, sans-serif; padding: 20px; background-color: #f5f5f5;">
	<div style="max-width: 600px; margin: 0 auto; background-color: white; padding: 40px; border-radius: 10px; box-shadow: 0 2px 4px rgba(0,0,0,0.1);">
		<h1 style="color: #333; margin-bottom: 20px;">이메일 인증</h1>
		<p style="color: #666; line-height: 1.6; margin-bottom: 30px;">
			우리동네금은방에 가입해주셔서 감사합니다.<br>
			아래 인증 코드를 입력하여 이메일 인증을 완료해주세요.
		</p>
		<div style="background-color: #f8f9fa; padding: 30px; border-radius: 8px; text-align: center; margin-bottom: 30px;">
			<h2 style="color: #333; margin: 0; font-size: 36px; letter-spacing: 4px;">%s</h2>
		</div>
		<p style="color: #999; font-size: 14px; margin-bottom: 10px;">
			* 이 인증 코드는 5분 동안 유효합니다.
		</p>
		<p style="color: #999; font-size: 14px;">
			* 본인이 요청하지 않은 경우, 이 이메일을 무시하셔도 됩니다.
		</p>
	</div>
</body>
</html>
`, code)

	// 이메일 메시지 구성
	message := []byte(fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/html; charset=UTF-8\r\n"+
			"\r\n"+
			"%s",
		fromEmail, toEmail, subject, body,
	))

	// SMTP 인증
	auth := smtp.PlainAuth("", fromEmail, password, smtpHost)

	// 이메일 전송
	err := smtp.SendMail(
		smtpHost+":"+smtpPort,
		auth,
		fromEmail,
		[]string{toEmail},
		message,
	)

	if err != nil {
		log.Printf("이메일 전송 실패: %v", err)
		return fmt.Errorf("이메일 전송에 실패했습니다: %v", err)
	}

	log.Printf("인증 이메일 발송 완료: %s", toEmail)
	return nil
}

// SendPasswordResetEmail sends a password reset link via Gmail SMTP
func SendPasswordResetEmail(toEmail, resetToken string) error {
	// Gmail SMTP 설정 (환경변수에서 가져오기)
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"
	fromEmail := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")
	frontendURL := os.Getenv("FRONTEND_URL")

	// 개발 모드: SMTP 설정이 없으면 콘솔에 출력만
	if fromEmail == "" || password == "" {
		log.Printf("[DEV MODE] 비밀번호 재설정 토큰: %s (받는 사람: %s)", resetToken, toEmail)
		log.Printf("[DEV MODE] 재설정 링크: %s/reset-password?token=%s", frontendURL, resetToken)
		return nil
	}

	// 프론트엔드 URL 기본값 설정
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}

	// 비밀번호 재설정 링크 생성
	resetLink := fmt.Sprintf("%s/reset-password?token=%s", frontendURL, resetToken)

	// 이메일 본문 구성
	subject := "[우리동네금은방] 비밀번호 재설정"
	body := fmt.Sprintf(`
<html>
<body style="font-family: Arial, sans-serif; padding: 20px; background-color: #f5f5f5;">
	<div style="max-width: 600px; margin: 0 auto; background-color: white; padding: 40px; border-radius: 10px; box-shadow: 0 2px 4px rgba(0,0,0,0.1);">
		<h1 style="color: #333; margin-bottom: 20px;">비밀번호 재설정</h1>
		<p style="color: #666; line-height: 1.6; margin-bottom: 30px;">
			우리동네금은방 계정의 비밀번호 재설정을 요청하셨습니다.<br>
			아래 버튼을 클릭하여 새로운 비밀번호를 설정하세요.
		</p>
		<div style="text-align: center; margin-bottom: 30px;">
			<a href="%s" style="display: inline-block; background-color: #FFD700; color: #333; padding: 15px 40px; text-decoration: none; border-radius: 8px; font-weight: bold; font-size: 16px;">
				비밀번호 재설정하기
			</a>
		</div>
		<p style="color: #999; font-size: 14px; margin-bottom: 10px;">
			* 이 링크는 1시간 동안 유효합니다.
		</p>
		<p style="color: #999; font-size: 14px; margin-bottom: 10px;">
			* 버튼이 작동하지 않으면 아래 링크를 복사하여 브라우저에 붙여넣으세요:
		</p>
		<p style="color: #666; font-size: 12px; word-break: break-all; background-color: #f8f9fa; padding: 10px; border-radius: 4px;">
			%s
		</p>
		<p style="color: #999; font-size: 14px; margin-top: 30px;">
			* 본인이 요청하지 않은 경우, 이 이메일을 무시하셔도 됩니다.
		</p>
	</div>
</body>
</html>
`, resetLink, resetLink)

	// 이메일 메시지 구성
	message := []byte(fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/html; charset=UTF-8\r\n"+
			"\r\n"+
			"%s",
		fromEmail, toEmail, subject, body,
	))

	// SMTP 인증
	auth := smtp.PlainAuth("", fromEmail, password, smtpHost)

	// 이메일 전송
	err := smtp.SendMail(
		smtpHost+":"+smtpPort,
		auth,
		fromEmail,
		[]string{toEmail},
		message,
	)

	if err != nil {
		log.Printf("비밀번호 재설정 이메일 전송 실패: %v", err)
		return fmt.Errorf("이메일 전송에 실패했습니다: %v", err)
	}

	log.Printf("비밀번호 재설정 이메일 발송 완료: %s", toEmail)
	return nil
}

// CleanupExpiredCodes periodically removes expired verification codes
func CleanupExpiredCodes() {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			verificationMutex.Lock()

			// Clean email codes
			for email, code := range emailVerificationCodes {
				if time.Now().After(code.ExpiresAt) {
					delete(emailVerificationCodes, email)
				}
			}

			// Clean phone codes
			for phone, code := range phoneVerificationCodes {
				if time.Now().After(code.ExpiresAt) {
					delete(phoneVerificationCodes, phone)
				}
			}

			verificationMutex.Unlock()
		}
	}()
}
