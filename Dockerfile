# 1단계: Build stage
FROM golang:1.23 AS builder

# 작업 디렉토리 설정
WORKDIR /app

# 모듈 캐싱
COPY go.mod go.sum ./
RUN go mod download

# 소스 전체 복사
COPY . .

# Go 빌드 (cmd/server 경로 기준)
RUN CGO_ENABLED=0 GOOS=linux go build -o app ./cmd/server

# 2단계: Run stage
FROM alpine:latest

# 타임존 및 인증서 설치
RUN apk --no-cache add ca-certificates tzdata

# 작업 디렉토리 설정
WORKDIR /root/

# 빌드된 실행 파일 복사
COPY --from=builder /app/app .

# .env 파일 복사 (환경 변수용)
COPY --from=builder /app/.env .env

# 포트 (네 서버에서 사용하는 포트로 맞춰)
EXPOSE 8080

# 실행 명령
CMD ["./app"]
