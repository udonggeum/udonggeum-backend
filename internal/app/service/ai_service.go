package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ikkim/udonggeum-backend/config"
	"github.com/ikkim/udonggeum-backend/internal/app/model"
)

// AIService AI 서비스 인터페이스
type AIService interface {
	GenerateContent(req *model.GenerateContentRequest) ([]string, error)
}

type aiService struct {
	config *config.Config
}

// NewAIService AI 서비스 생성자
func NewAIService(cfg *config.Config) AIService {
	return &aiService{config: cfg}
}

// OpenAI API 요청 구조체
type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
	MaxTokens   int             `json:"max_tokens"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// GenerateContent AI로 게시글 내용 3가지 버전 생성
func (s *aiService) GenerateContent(req *model.GenerateContentRequest) ([]string, error) {
	if s.config.OpenAI.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is not configured")
	}

	systemPrompt := s.buildSystemPrompt(req)
	userPrompt := s.buildUserPrompt(req)

	raw, err := s.callOpenAI(systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to call OpenAI API: %v", err)
	}

	return parseVersions(raw), nil
}

// parseVersions [버전1] ~ [버전3] 마커 기준으로 분리
func parseVersions(raw string) []string {
	markers := []string{"[버전1]", "[버전2]", "[버전3]"}
	var versions []string

	for i, marker := range markers {
		idx := strings.Index(raw, marker)
		if idx == -1 {
			break
		}
		start := idx + len(marker)
		end := len(raw)
		if i+1 < len(markers) {
			nextIdx := strings.Index(raw[start:], markers[i+1])
			if nextIdx != -1 {
				end = start + nextIdx
			}
		}
		v := strings.TrimSpace(raw[start:end])
		if v != "" {
			versions = append(versions, v)
		}
	}

	if len(versions) == 0 {
		return []string{strings.TrimSpace(raw)}
	}
	return versions
}

// buildSystemPrompt 게시글 타입별 역할·규칙 정의
func (s *aiService) buildSystemPrompt(req *model.GenerateContentRequest) string {
	var sb strings.Builder

	sb.WriteString("너는 상황에 따라 글을 다르게 작성하는 마케팅 전문가다.\n\n")

	sb.WriteString("[공통 규칙]\n")
	sb.WriteString("- 과한 광고 느낌 금지\n")
	sb.WriteString("- 실제 사람이 쓴 것처럼 자연스럽게\n")
	sb.WriteString("- 짧고 가독성 좋게\n")
	sb.WriteString("- 불필요한 과장 표현 금지\n")
	sb.WriteString("- 제공되지 않은 정보는 절대 추측하거나 만들어내지 마세요.\n\n")

	switch req.Type {
	case model.TypeStoreNews:
		sb.WriteString("[역할] 금은방 사장\n")
		sb.WriteString("[목표] 신뢰 확보, 방문 유도\n")
		sb.WriteString("[특징] 동네 장사 느낌, 편하게 방문 유도, 친근하지만 과하지 않게\n")

	case model.TypeProductNews:
		sb.WriteString("[역할] 금은방 사장\n")
		sb.WriteString("[목표] 상품 관심 유도\n")
		sb.WriteString("[특징] 자연스럽게 제품 소개, 부담 없는 느낌\n")

	case model.TypeBuyGold:
		sb.WriteString("[역할] 금은방 사장\n")
		sb.WriteString("[목표] 금을 팔도록 유도\n")
		sb.WriteString("[특징] 공감형, 신뢰감, 문의 유도\n")

	case model.TypeSellGold:
		sb.WriteString("[역할] 일반 사용자 (금 판매 희망)\n")
		sb.WriteString("[목표] 금은방에 빠르게 판매\n")
		sb.WriteString("[특징] 투박하고 현실적인 말투, 개인 느낌, 광고 느낌 없이\n")

	case model.TypeQuestion:
		sb.WriteString("[역할] 금에 대해 궁금한 일반 사용자\n")
		sb.WriteString("[목표] 명확한 질문 전달, 답변 유도\n")
		sb.WriteString("[특징] 솔직하고 구체적, 상황 설명 포함\n")

	default:
		sb.WriteString("[역할] 금 관련 글 작성자\n")
		sb.WriteString("[목표] 정보 전달\n")
	}

	sb.WriteString("\n[출력 형식]\n")
	sb.WriteString("- 아래 형식으로 정확히 3개 버전을 출력하세요.\n")
	sb.WriteString("- 각 버전은 서로 다른 느낌과 표현을 사용하세요.\n")
	sb.WriteString("- 제목, 주석, 설명 없이 본문 텍스트만 출력하세요.\n\n")
	sb.WriteString("[버전1]\n(본문)\n\n[버전2]\n(본문)\n\n[버전3]\n(본문)")

	return sb.String()
}

// buildUserPrompt 실제 입력 데이터 정리
func (s *aiService) buildUserPrompt(req *model.GenerateContentRequest) string {
	var sb strings.Builder

	sb.WriteString("아래 정보를 바탕으로 게시글 본문 3가지 버전을 작성해주세요.\n\n")

	if req.Title != nil && *req.Title != "" {
		sb.WriteString(fmt.Sprintf("제목: %s\n", *req.Title))
	}

	titleVal := ""
	if req.Title != nil {
		titleVal = *req.Title
	}
	var filteredKeywords []string
	for _, kw := range req.Keywords {
		if strings.TrimSpace(kw) != "" && kw != titleVal {
			filteredKeywords = append(filteredKeywords, kw)
		}
	}
	if len(filteredKeywords) > 0 {
		sb.WriteString(fmt.Sprintf("추가 요청사항: %s\n", strings.Join(filteredKeywords, ", ")))
	}

	if req.GoldType != nil && *req.GoldType != "" && *req.GoldType != "알 수 없음" {
		sb.WriteString(fmt.Sprintf("금 종류: %s\n", *req.GoldType))
	}

	if req.Weight != nil && *req.Weight > 0 {
		don := *req.Weight / 3.75
		sb.WriteString(fmt.Sprintf("중량: %.2fg (약 %.2f돈)\n", *req.Weight, don))
	}

	if req.Price != nil {
		if *req.Price == 0 {
			sb.WriteString("가격: 협의\n")
		} else {
			sb.WriteString(fmt.Sprintf("가격: %d원\n", *req.Price))
		}
	}

	if req.Location != nil && *req.Location != "" {
		sb.WriteString(fmt.Sprintf("거래 희망 지역: %s\n", *req.Location))
	}

	return sb.String()
}

// callOpenAI OpenAI API 호출
func (s *aiService) callOpenAI(systemPrompt, userPrompt string) (string, error) {
	reqData := openAIRequest{
		Model: s.config.OpenAI.Model,
		Messages: []openAIMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.6,
		MaxTokens:   1200,
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.config.OpenAI.APIKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	var openAIResp openAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	if openAIResp.Error != nil {
		return "", fmt.Errorf("OpenAI API error: %s", openAIResp.Error.Message)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return strings.TrimSpace(openAIResp.Choices[0].Message.Content), nil
}
