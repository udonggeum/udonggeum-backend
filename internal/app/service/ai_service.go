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
	GenerateContent(req *model.GenerateContentRequest) (string, error)
}

type aiService struct {
	config *config.Config
}

// NewAIService AI 서비스 생성자
func NewAIService(cfg *config.Config) AIService {
	return &aiService{
		config: cfg,
	}
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

// GenerateContent AI로 게시글 내용 생성
func (s *aiService) GenerateContent(req *model.GenerateContentRequest) (string, error) {
	if s.config.OpenAI.APIKey == "" {
		return "", fmt.Errorf("OpenAI API key is not configured")
	}

	systemPrompt := s.buildSystemPrompt(req)
	userPrompt := s.buildUserPrompt(req)

	content, err := s.callOpenAI(systemPrompt, userPrompt)
	if err != nil {
		return "", fmt.Errorf("failed to call OpenAI API: %v", err)
	}

	return content, nil
}

// buildSystemPrompt 게시글 타입별 역할·규칙 정의 (system role)
func (s *aiService) buildSystemPrompt(req *model.GenerateContentRequest) string {
	var sb strings.Builder

	// 타입별 페르소나 + 목적
	switch req.Type {

	case model.TypeSellGold:
		sb.WriteString(
			"당신은 개인 사용자가 특정 지역의 금은방 사장님들에게 금 매입을 요청하는 글을 작성하는 전문가입니다.\n" +
				"이 글의 독자는 일반 소비자가 아니라, 선택된 지역 내 금은방 사장님들입니다.\n" +
				"목표는 금은방으로부터 매입 가능 여부나 견적 제안을 받는 것입니다.\n\n" +
				"- 톤: 담백하고 객관적이며 신뢰 위주의 문체\n" +
				"- 개인적인 감정, 추억, 사연은 절대 포함하지 마세요.\n" +
				"- 중고거래(C2C)처럼 보이는 표현은 사용하지 마세요.\n" +
				"- '판매합니다' 보다는 '매입 가능 여부 문의', '매입 제안 요청' 표현을 사용하세요.\n" +
				"- 글의 대상은 금은방 사장님이며, 일반 중고거래 글처럼 보이면 안 됩니다.\n",
		)

	case model.TypeBuyGold:
		sb.WriteString(
			"당신은 금은방 사장님입니다. 일반 사용자들을 대상으로 한 금 매입 홍보 게시글 본문을 작성하세요.\n" +
				"목표는 매장 방문 또는 연락을 유도하는 것입니다.\n\n" +
				"- 톤: 전문적이고 신뢰감 있는 어조\n" +
				"- 높은 매입가, 정직한 거래, 신뢰 요소 강조\n" +
				"- 과장되거나 허위 느낌의 표현은 피하세요.\n",
		)

	case model.TypeProductNews:
		sb.WriteString(
			"당신은 금은방 전문가입니다. 금 제품 관련 소식 게시글의 본문을 작성하세요.\n\n" +
				"- 톤: 정보 중심, 객관적이고 이해하기 쉬운 문체\n" +
				"- 제품 특징과 장점을 간결하게 전달\n",
		)

	case model.TypeStoreNews:
		sb.WriteString(
			"당신은 금은방 사장님입니다. 매장 소식 또는 이벤트 안내 게시글 본문을 작성하세요.\n\n" +
				"- 톤: 친근하지만 상업적이지 않게\n" +
				"- 이벤트, 휴무, 혜택 등 핵심 정보 위주\n",
		)

	case model.TypeQuestion:
		sb.WriteString(
			"당신은 금 관련 질문 게시글 작성 전문가입니다. 질문 게시글 본문을 작성하세요.\n\n" +
				"- 질문의 배경과 상황을 간단히 설명\n" +
				"- 구체적인 상황과 궁금한 점을 명확히 서술\n" +
				"- 답변자가 이해하기 쉬운 구조로 작성\n",
		)

	default:
		sb.WriteString(
			"당신은 금 관련 게시글 작성 전문가입니다. 아래 정보를 바탕으로 게시글 본문을 작성하세요.\n",
		)
	}

	// 공통 규칙
	sb.WriteString("\n[중요 작성 규칙]\n")
	sb.WriteString("- 제공되지 않은 정보는 절대 추측하거나 만들어내지 마세요.\n")
	sb.WriteString("- 값이 없는 항목(금 종류, 중량, 가격, 지역 등)은 해당 내용을 언급하지 말고 문장을 생략하세요.\n")
	sb.WriteString("- 개인적인 스토리(추억, 소장품, 애정, 선물 등)는 절대 포함하지 마세요.\n")
	sb.WriteString("- 같은 입력이라도 문장 표현과 문단 구성을 매번 조금씩 바꿔 반복 티가 나지 않게 작성하세요.\n")
	sb.WriteString("- 본문은 2~4개의 짧은 문단으로 구성하고, 줄바꿈을 활용해 가독성을 높이세요.\n")
	sb.WriteString("- '아래는', '다음은', '정리하면' 같은 메타 설명은 절대 사용하지 마세요.\n")

	// 출력 형식 지시
	sb.WriteString("\n제목, 주석, 설명, 추가 안내 없이 게시글 본문 텍스트만 출력하세요.")

	return sb.String()
}

// buildUserPrompt 실제 입력 데이터 정리 (user role)
func (s *aiService) buildUserPrompt(req *model.GenerateContentRequest) string {
	var sb strings.Builder

	sb.WriteString("아래 정보를 바탕으로 게시글 본문을 작성해주세요.\n\n")

	if req.Title != nil && *req.Title != "" {
		sb.WriteString(fmt.Sprintf("제목: %s\n", *req.Title))
	}

	// 키워드에서 제목과 중복되는 항목을 제거하고 추가 요청사항만 표시
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
		Temperature: 0.7,
		MaxTokens:   500,
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
