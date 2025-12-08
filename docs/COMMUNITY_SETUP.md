# 금광산 커뮤니티 설정 가이드

## 개요

우동금 플랫폼의 "금광산" 커뮤니티 기능이 성공적으로 구현되었습니다.

## 구현된 기능

### ✅ 백엔드 (Go/Gin)

1. **데이터 모델**
   - `CommunityPost` - 게시글 모델
   - `CommunityComment` - 댓글 모델 (대댓글 지원)
   - `PostLike` - 게시글 좋아요
   - `CommentLike` - 댓글 좋아요

2. **Repository 레이어**
   - CRUD 작업
   - 페이지네이션
   - 필터링 (카테고리, 타입, 상태, 검색 등)
   - 좋아요 관리
   - QnA 답변 채택

3. **Service 레이어**
   - 비즈니스 로직
   - 권한 검증
   - 자동 조회수 증가

4. **Controller 레이어**
   - RESTful API 엔드포인트
   - JWT 인증 통합
   - 에러 핸들링

5. **라우터 통합**
   - `/api/v1/community/posts/*`
   - `/api/v1/community/comments/*`

### ✅ 프론트엔드 (TypeScript/Zod)

1. **Zod 스키마** (`src/schemas/community.ts`)
   - 모든 요청/응답 타입 정의
   - 런타임 검증
   - TypeScript 타입 자동 생성

2. **유틸리티 상수**
   - 카테고리/타입 레이블
   - 권한 체크 헬퍼

## 마이그레이션

### 데이터베이스 테이블 생성

백엔드 서버 실행 시 자동으로 다음 테이블이 생성됩니다:

```sql
-- 게시글
CREATE TABLE community_posts (
  id SERIAL PRIMARY KEY,
  title VARCHAR(200) NOT NULL,
  content TEXT NOT NULL,
  category VARCHAR(20) NOT NULL,
  type VARCHAR(20) NOT NULL,
  status VARCHAR(20) DEFAULT 'active',
  user_id INTEGER NOT NULL REFERENCES users(id),
  store_id INTEGER REFERENCES stores(id),
  gold_type VARCHAR(50),
  weight FLOAT,
  price BIGINT,
  location VARCHAR(100),
  is_answered BOOLEAN DEFAULT FALSE,
  accepted_answer_id INTEGER,
  view_count INTEGER DEFAULT 0,
  like_count INTEGER DEFAULT 0,
  comment_count INTEGER DEFAULT 0,
  image_urls TEXT[],
  created_at TIMESTAMP,
  updated_at TIMESTAMP,
  deleted_at TIMESTAMP
);

-- 댓글
CREATE TABLE community_comments (
  id SERIAL PRIMARY KEY,
  content TEXT NOT NULL,
  user_id INTEGER NOT NULL REFERENCES users(id),
  post_id INTEGER NOT NULL REFERENCES community_posts(id),
  parent_id INTEGER REFERENCES community_comments(id),
  is_answer BOOLEAN DEFAULT FALSE,
  is_accepted BOOLEAN DEFAULT FALSE,
  like_count INTEGER DEFAULT 0,
  created_at TIMESTAMP,
  updated_at TIMESTAMP,
  deleted_at TIMESTAMP
);

-- 게시글 좋아요
CREATE TABLE post_likes (
  id SERIAL PRIMARY KEY,
  post_id INTEGER NOT NULL REFERENCES community_posts(id),
  user_id INTEGER NOT NULL REFERENCES users(id),
  created_at TIMESTAMP,
  UNIQUE(post_id, user_id)
);

-- 댓글 좋아요
CREATE TABLE comment_likes (
  id SERIAL PRIMARY KEY,
  comment_id INTEGER NOT NULL REFERENCES community_comments(id),
  user_id INTEGER NOT NULL REFERENCES users(id),
  created_at TIMESTAMP,
  UNIQUE(comment_id, user_id)
);
```

### 인덱스 (자동 생성)

- `community_posts`: user_id, store_id, category, type, status
- `community_comments`: user_id, post_id, parent_id
- `post_likes`: (post_id, user_id) UNIQUE
- `comment_likes`: (comment_id, user_id) UNIQUE

## 서버 시작

```bash
cd ../nq-logmgragent-test

# 환경 변수 설정
cp .env.example .env
# .env 파일 수정 (DATABASE_URL 등)

# 의존성 설치
go mod download

# 서버 실행
go run cmd/server/main.go
```

서버가 실행되면:
1. 데이터베이스 마이그레이션 자동 실행
2. 테이블 생성
3. API 엔드포인트 활성화: `http://localhost:8080/api/v1/community/*`

## API 테스트

### VS Code REST Client 사용

1. VS Code에서 `api_examples_community.http` 파일 열기
2. REST Client 확장 설치
3. 각 요청 위의 "Send Request" 클릭

### cURL 사용

```bash
# 게시글 목록 조회
curl http://localhost:8080/api/v1/community/posts

# 금거래 카테고리만 조회
curl "http://localhost:8080/api/v1/community/posts?category=gold_trade"

# 게시글 작성 (인증 필요)
curl -X POST http://localhost:8080/api/v1/community/posts \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "24K 금반지 판매합니다",
    "content": "순도 99.9% 금반지 5돈 판매합니다.",
    "category": "gold_trade",
    "type": "sell_gold",
    "gold_type": "24K",
    "weight": 18.75,
    "price": 3500000,
    "location": "서울 강남구"
  }'
```

## 프론트엔드 통합

### 1. API Service 생성

`src/services/community.ts`:

```typescript
import { apiClient } from '@/api/client';
import {
  CreatePostRequestSchema,
  PostListResponseSchema,
  PostDetailResponseSchema,
  type CreatePostRequest,
  type PostListQuery,
} from '@/schemas/community';

class CommunityService {
  // 게시글 목록 조회
  async getPosts(query?: PostListQuery) {
    const { data } = await apiClient.get('/community/posts', { params: query });
    return PostListResponseSchema.parse(data);
  }

  // 게시글 상세 조회
  async getPost(id: number) {
    const { data } = await apiClient.get(`/community/posts/${id}`);
    return PostDetailResponseSchema.parse(data);
  }

  // 게시글 작성
  async createPost(request: CreatePostRequest) {
    const validated = CreatePostRequestSchema.parse(request);
    const { data } = await apiClient.post('/community/posts', validated);
    return data;
  }

  // ... 기타 메서드
}

export const communityService = new CommunityService();
```

### 2. TanStack Query Hook 생성

`src/hooks/queries/useCommunityQueries.ts`:

```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { communityService } from '@/services/community';
import type { PostListQuery, CreatePostRequest } from '@/schemas/community';

// Query Keys
export const communityKeys = {
  all: ['community'] as const,
  posts: () => [...communityKeys.all, 'posts'] as const,
  post: (id: number) => [...communityKeys.posts(), id] as const,
  postList: (query?: PostListQuery) => [...communityKeys.posts(), 'list', query] as const,
};

// 게시글 목록 조회
export function usePosts(query?: PostListQuery) {
  return useQuery({
    queryKey: communityKeys.postList(query),
    queryFn: () => communityService.getPosts(query),
  });
}

// 게시글 상세 조회
export function usePost(id: number) {
  return useQuery({
    queryKey: communityKeys.post(id),
    queryFn: () => communityService.getPost(id),
  });
}

// 게시글 작성
export function useCreatePost() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: CreatePostRequest) => communityService.createPost(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: communityKeys.posts() });
    },
  });
}
```

### 3. 컴포넌트에서 사용

```tsx
import { usePosts, useCreatePost } from '@/hooks/queries/useCommunityQueries';

function CommunityPage() {
  const { data, isLoading } = usePosts({ category: 'gold_trade', page: 1 });
  const createPost = useCreatePost();

  if (isLoading) return <div>Loading...</div>;

  return (
    <div>
      <h1>금광산 - 금거래</h1>
      {data?.data.map(post => (
        <div key={post.id}>
          <h2>{post.title}</h2>
          <p>{post.content}</p>
        </div>
      ))}
    </div>
  );
}
```

## 주요 파일 위치

### 백엔드
```
nq-logmgragent-test/
├── internal/app/
│   ├── model/
│   │   ├── community_post.go        # 게시글 모델
│   │   └── community_comment.go     # 댓글 모델
│   ├── repository/
│   │   └── community_repository.go  # DB 접근 레이어
│   ├── service/
│   │   └── community_service.go     # 비즈니스 로직
│   └── controller/
│       └── community_controller.go  # API 핸들러
├── internal/router/
│   └── router.go                    # 라우터 (커뮤니티 경로 추가됨)
├── internal/db/
│   └── migrate.go                   # 마이그레이션 (커뮤니티 테이블 추가됨)
├── docs/
│   ├── COMMUNITY_API.md             # API 문서
│   └── COMMUNITY_SETUP.md           # 이 파일
└── api_examples_community.http      # API 테스트 예제
```

### 프론트엔드
```
nq-logmgrtransfer-test/
└── src/
    └── schemas/
        └── community.ts             # Zod 스키마 + TypeScript 타입
```

## 권한 체계

### 일반 사용자 (role: "user")
- ✅ 금 매수 글 작성 (`type: "sell_gold"`)
- ✅ 금소식 글 작성 (`type: "news"`, `"review"`, `"tip"`)
- ✅ QnA 질문 작성 (`type: "question"`)
- ✅ 댓글/대댓글 작성
- ✅ 좋아요
- ❌ 금 매입 글 작성 (`type: "buy_gold"`)
- ❌ FAQ 작성 (`type: "faq"`)

### 금은방 사장님 (role: "admin")
- ✅ 모든 일반 사용자 권한
- ✅ 금 매입 글 작성 (`type: "buy_gold"`, `store_id` 필수)
- ✅ FAQ 작성 (`type: "faq"`)
- ✅ 모든 게시글/댓글 관리 (수정/삭제)

## 다음 단계

### 프론트엔드 구현 필요
1. `src/services/community.ts` - API 호출 서비스
2. `src/hooks/queries/useCommunityQueries.ts` - TanStack Query 훅
3. 페이지 컴포넌트:
   - `CommunityListPage.tsx` - 게시글 목록
   - `CommunityDetailPage.tsx` - 게시글 상세
   - `CommunityWritePage.tsx` - 게시글 작성/수정
4. 공통 컴포넌트:
   - `PostCard.tsx` - 게시글 카드
   - `CommentList.tsx` - 댓글 목록
   - `CommentForm.tsx` - 댓글 작성 폼

### 추가 기능 (선택)
- [ ] 이미지 업로드 API
- [ ] 게시글 신고 기능
- [ ] 알림 시스템
- [ ] 통계 대시보드
- [ ] 태그 시스템
- [ ] 북마크 기능

## 문제 해결

### 마이그레이션 실패
```bash
# 수동 마이그레이션
psql -d udonggeum -f migration.sql
```

### 권한 에러
- 토큰에 `role` 클레임이 포함되어 있는지 확인
- 미들웨어가 `UserRoleKey`를 context에 설정하는지 확인

### Import 에러
백엔드 패키지 경로 확인:
```go
// 올바른 경로
import "github.com/ikkim/nq-logmgragent-test/internal/app/model"

// main.go나 go.mod의 module 선언 확인
module github.com/ikkim/udonggeum-backend
```

프로젝트에 따라 import 경로 수정 필요.

## 참고 문서

- [API 문서](./COMMUNITY_API.md)
- [API 테스트 예제](../api_examples_community.http)
- [프론트엔드 스키마](../src/schemas/community.ts)

---

**구현 완료일**: 2025-12-08
