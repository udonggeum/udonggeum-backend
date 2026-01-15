# 1단계: Build stage
FROM golang:1.24 AS builder

# 작업 디렉토리 설정
WORKDIR /app

# 모듈 캐싱
COPY go.mod go.sum ./
RUN go mod download

# 소스 전체 복사
COPY . .

# Go 빌드 (server와 seed 모두 빌드)
RUN CGO_ENABLED=0 GOOS=linux go build -o app ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -o seed ./cmd/seed

# 2단계: Run stage
FROM alpine:latest

# 타임존 및 인증서 설치
RUN apk --no-cache add ca-certificates tzdata

# 작업 디렉토리 설정
WORKDIR /root/

# 데이터 디렉토리 생성
RUN mkdir -p /app/data

# 빌드된 실행 파일 복사
COPY --from=builder /app/app .
COPY --from=builder /app/seed .

# 스크립트 복사
COPY scripts/import_all.sh /root/import_all.sh
RUN chmod +x /root/import_all.sh

# 포트 (네 서버에서 사용하는 포트로 맞춰)
EXPOSE 8080

# 실행 명령
CMD ["./app"]
