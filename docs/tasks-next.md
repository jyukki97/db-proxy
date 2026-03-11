## 다음 Phase Task (Phase 17-19)

기획 검토 결과 채택된 기능. 우선순위 순서대로 정리.

---

### Phase 17: OpenTelemetry Distributed Tracing (W26 + W27)

분산 추적으로 프록시 내부 처리 단계별 지연 시간을 가시화한다. 기존 Prometheus 메트릭의 자연스러운 확장.

| Task | 작업 | 이슈 제목 |
|------|------|-----------|
| T17-1 | OTel 의존성 및 설정 구조체 추가 | `feat(telemetry): OpenTelemetry 설정 및 TracerProvider 초기화` |
| T17-2 | TracerProvider + Exporter 초기화 | (동일 이슈) |
| T17-3 | Simple Query 경로 Span 계측 | `feat(telemetry): 쿼리 처리 경로 Span 계측` |
| T17-4 | Extended Query 경로 Span 계측 | (동일 이슈) |
| T17-5 | Data API traceparent 전파 | `feat(telemetry): Data API HTTP trace context 전파` |
| T17-6 | OTel 메트릭 브릿지 (선택) | `feat(telemetry): Prometheus → OTel Metrics 브릿지` |
| T17-7 | OTel 테스트 및 문서화 | `test(telemetry): OTel 통합 테스트 및 문서` |

<details>
<summary>Phase 17 상세</summary>

#### T17-1: OTel 의존성 및 설정 구조체 추가
- **범위**: `go.mod`에 `go.opentelemetry.io/otel`, `otel/sdk`, `otel/exporters/otlp` 추가. `config.go`에 `TelemetryConfig` 구조체 추가
- **설정 항목**:
  ```yaml
  telemetry:
    enabled: false
    exporter: "otlp"          # otlp | jaeger | stdout
    endpoint: "localhost:4317" # OTLP gRPC endpoint
    service_name: "pgmux"
    sample_ratio: 1.0          # 0.0 ~ 1.0
  ```
- **완료 기준**: 설정 파싱 및 validation 통과, `enabled: false`일 때 no-op

#### T17-2: TracerProvider + Exporter 초기화
- **범위**: `internal/telemetry/telemetry.go` 신규 생성. `Init(cfg) (shutdown func(), err)` — TracerProvider, Resource(service.name, service.version), Sampler, Exporter 설정
- **지원 Exporter**: OTLP gRPC (기본), Jaeger (호환), stdout (디버그)
- **완료 기준**: `Init()` 호출 시 TracerProvider 전역 등록, shutdown 시 flush 확인

#### T17-3: Simple Query 경로 Span 계측
- **범위**: `proxy/server.go`의 `relayQueries()` 내부에 span 삽입
- **Span 구조**:
  ```
  pgmux.query (root span)
  ├── pgmux.parse          # 쿼리 분류 (Classify/ClassifyAST)
  ├── pgmux.firewall       # 방화벽 체크 (활성 시)
  ├── pgmux.cache.lookup   # 캐시 조회 (읽기 쿼리)
  ├── pgmux.pool.acquire   # 커넥션 풀 획득 대기
  ├── pgmux.backend.exec   # 백엔드 DB 실행
  └── pgmux.cache.store    # 캐시 저장 (캐시 미스 시)
  ```
- **Span Attributes**: `db.system=postgresql`, `db.statement`(처음 100자), `db.operation`(SELECT/INSERT 등), `pgmux.route`(writer/reader), `pgmux.cached`(true/false)
- **완료 기준**: Jaeger UI에서 쿼리 처리 단계별 span 트리 확인

#### T17-4: Extended Query 경로 Span 계측
- **범위**: Parse/Bind/Execute/Sync 메시지 처리 경로에 span 삽입
- **Span 구조**:
  ```
  pgmux.extended_query (root span)
  ├── pgmux.parse_message   # Parse 메시지 처리 + 라우팅 결정
  ├── pgmux.pool.acquire
  ├── pgmux.backend.exec    # Bind~Sync 릴레이
  └── pgmux.cache.store     # (캐싱 대상일 때)
  ```
- **완료 기준**: Prepared Statement 쿼리도 span 트리 정상 출력

#### T17-5: Data API traceparent 전파
- **범위**: `dataapi/handler.go`에서 HTTP 요청의 `traceparent` 헤더를 파싱하여 부모 span context로 사용. `otel/propagation` 패키지 활용
- **완료 기준**: 외부 서비스 → Data API → DB 실행까지 단일 trace로 연결

#### T17-6: OTel 메트릭 브릿지 (선택)
- **범위**: 기존 Prometheus 메트릭을 OTel Metrics로도 내보낼 수 있도록 브릿지 구성. `prometheus.Exporter`로 기존 registry를 OTel에 노출
- **주의**: 기존 `/metrics` 엔드포인트는 그대로 유지 (하위 호환)
- **완료 기준**: OTel Collector에서 pgmux 메트릭 수신 확인, 기존 Prometheus scrape도 정상

#### T17-7: OTel 테스트 및 문서화
- **범위**: stdout exporter 기반 단위 테스트 (span 생성 확인), docker-compose에 Jaeger 추가하여 E2E 검증. README에 telemetry 설정 섹션 추가
- **완료 기준**: `make test` 통과, Jaeger에서 pgmux 서비스 trace 확인 가능

</details>

---

### Phase 18: Config File Watch — fsnotify 자동 리로드 (W28)

설정 파일 변경 시 자동으로 SIGHUP을 트리거하여 무중단 리로드. K8s ConfigMap, Docker bind mount 환경에서 별도 조작 없이 설정 반영.

| Task | 작업 | 이슈 제목 |
|------|------|-----------|
| T18-1 | fsnotify 의존성 및 설정 추가 | `feat(config): 설정 파일 변경 감지 자동 리로드` |
| T18-2 | FileWatcher 구현 | (동일 이슈) |
| T18-3 | Server 통합 및 디바운싱 | (동일 이슈) |
| T18-4 | File Watch 테스트 | (동일 이슈) |

<details>
<summary>Phase 18 상세</summary>

#### T18-1: fsnotify 의존성 및 설정 추가
- **범위**: `go.mod`에 `github.com/fsnotify/fsnotify` 추가. `config.go`에 `watch` 필드 추가
- **설정 항목**:
  ```yaml
  config:
    watch: true    # 파일 변경 시 자동 리로드 (기본값: false)
  ```
- **완료 기준**: 설정 파싱 통과, `watch: false`일 때 watcher 미생성

#### T18-2: FileWatcher 구현
- **범위**: `internal/config/watcher.go` 신규 생성
  ```go
  type FileWatcher struct {
      path     string
      onChange func()       // 콜백: 기존 reloadConfig() 호출
      debounce time.Duration // 연속 이벤트 디바운싱 (기본 1초)
  }
  func NewFileWatcher(path string, onChange func()) (*FileWatcher, error)
  func (w *FileWatcher) Start(ctx context.Context) error
  func (w *FileWatcher) Stop()
  ```
- **주의사항**:
  - K8s ConfigMap은 symlink swap 방식이므로 `CREATE` 이벤트도 감시해야 함
  - 부모 디렉토리를 watch하여 symlink 교체를 감지
  - 디바운싱으로 연속 이벤트(에디터의 임시파일 생성 등) 1회로 병합
- **완료 기준**: 파일 수정/symlink 교체 시 onChange 콜백 1회 호출

#### T18-3: Server 통합 및 디바운싱
- **범위**: `cmd/pgmux/main.go`에서 `config.watch: true`일 때 FileWatcher 시작. onChange 콜백은 기존 `reloadConfig()` 함수를 그대로 호출 (Phase 11에서 구현된 SIGHUP 리로드 경로 재사용)
- **로그**: `slog.Info("config file changed, reloading", "path", configPath)`
- **완료 기준**: config.yaml 수정 → 자동 리로드 → `/admin/config`에서 새 값 확인

#### T18-4: File Watch 테스트
- **범위**: 단위 테스트 — 파일 수정 → 콜백 호출 확인, 디바운싱 동작 확인, symlink swap 시나리오
- **완료 기준**: 전체 테스트 통과, 기존 SIGHUP 리로드 테스트도 여전히 통과

</details>

---

### Phase 19: Prepared Statement Multiplexing PoC (W29 + W30 + W31)

Transaction Pooling 환경에서 Prepared Statement를 지원하기 위해, Parse/Bind를 프록시가 인터셉트하여 Simple Query로 합성. PgBouncer의 최대 맹점을 해결하는 킬러 피처. **보안 리스크가 높으므로 PoC → 호환성 테스트 → 프로덕션 순서로 진행.**

| Task | 작업 | 이슈 제목 |
|------|------|-----------|
| T19-1 | Bind 메시지 파서 구현 | `feat(protocol): Bind 메시지 파라미터 파싱` |
| T19-2 | 파라미터 타입별 SQL 리터럴 직렬화 | `feat(protocol): PG 타입별 안전한 SQL 리터럴 직렬화` |
| T19-3 | Query Synthesizer 구현 | `feat(proxy): Prepared Statement → Simple Query 합성기` |
| T19-4 | Describe 메시지 프록시 처리 | `feat(proxy): Describe Statement/Portal 프록시 응답` |
| T19-5 | Multiplexing 모드 설정 및 통합 | `feat(proxy): Prepared Statement Multiplexing 모드 통합` |
| T19-6 | SQL Injection 방어 테스트 매트릭스 | `test(security): 파라미터 직렬화 SQL Injection 테스트` |
| T19-7 | 드라이버 호환성 테스트 | `test(compat): 주요 PG 드라이버 Multiplexing 호환성 테스트` |
| T19-8 | Multiplexing E2E 테스트 및 문서화 | `test(proxy): Prepared Statement Multiplexing E2E` |

<details>
<summary>Phase 19 상세</summary>

#### T19-1: Bind 메시지 파서 구현
- **범위**: `internal/protocol/message.go`에 `ParseBindMessage()` 추가
  ```
  'B' + int32(len) + string(portal) + string(stmt) +
  int16(num_format_codes) + int16[](format_codes) +
  int16(num_params) + (int32(len) + byte[](value))[] +
  int16(num_result_format_codes) + int16[](result_format_codes)
  ```
- **파싱 결과**: portal name, statement name, format codes (text/binary), parameter values ([][]byte), result format codes
- **완료 기준**: 다양한 Bind 메시지 바이트에서 파라미터 정확 추출 단위 테스트

#### T19-2: 파라미터 타입별 SQL 리터럴 직렬화
- **범위**: `internal/protocol/literal.go` 신규 생성. OID별 파라미터 값을 안전한 SQL 리터럴 문자열로 변환
- **지원 타입**: bool(16), int2(21), int4(23), int8(20), float4(700), float8(701), numeric(1700), text(25), varchar(1043), bytea(17), timestamp(1114), timestamptz(1184), date(1082), uuid(2950), json(114), jsonb(3802), array types, NULL
- **보안 핵심**:
  - 문자열: single quote 이스케이핑 (`'` → `''`), backslash 이스케이핑
  - bytea: `'\x...'` hex 포맷
  - NULL: `-1` length → `NULL` 리터럴
  - 알 수 없는 타입: text로 취급, 이스케이핑 적용
- **완료 기준**: 모든 지원 타입의 이스케이핑 단위 테스트, 특수문자/유니코드/멀티바이트 포함

#### T19-3: Query Synthesizer 구현
- **범위**: `internal/proxy/synthesizer.go` 신규 생성
  ```go
  type Synthesizer struct {
      statements map[string]*PreparedStmt  // name → {query, paramOIDs}
  }
  // Parse 시 statement 등록
  func (s *Synthesizer) RegisterStatement(name, query string, paramOIDs []uint32)
  // Bind+Execute 시 Simple Query 합성
  func (s *Synthesizer) Synthesize(stmtName string, params [][]byte, formatCodes []int16) (string, error)
  ```
- **합성 로직**: Parse의 쿼리에서 `$1`, `$2` ... 플레이스홀더를 Bind 파라미터의 리터럴 값으로 치환
- **주의**: 플레이스홀더가 문자열 리터럴 내부에 있는 경우는 치환하지 않음
- **완료 기준**: `SELECT * FROM users WHERE id = $1` + param `[42]` → `SELECT * FROM users WHERE id = 42`

#### T19-4: Describe 메시지 프록시 처리
- **범위**: Describe(Statement) 요청 시, 등록된 statement의 쿼리를 실제 백엔드에 임시 Parse → Describe → Close 하여 ParameterDescription/RowDescription을 획득 후 클라이언트에 반환
- **캐싱**: statement name + query 해시로 Describe 결과를 캐싱하여 반복 호출 방지
- **완료 기준**: ORM의 Describe 요청에 올바른 컬럼 메타데이터 반환

#### T19-5: Multiplexing 모드 설정 및 통합
- **범위**: `config.go`에 설정 추가, `server.go`에 분기 로직
  ```yaml
  pool:
    prepared_statement_mode: "proxy"  # "proxy" (기본, 기존 동작) | "multiplex" (합성 모드)
  ```
- **동작**: `multiplex` 모드일 때 Parse→Register, Bind+Execute→Synthesize→Simple Query로 전송
- **완료 기준**: `multiplex` 모드에서 ORM의 Prepared Statement 쿼리가 정상 실행

#### T19-6: SQL Injection 방어 테스트 매트릭스
- **범위**: 파라미터 값에 의도적 SQL injection 페이로드를 넣어 합성된 쿼리가 안전한지 검증
- **테스트 케이스**:
  - `'; DROP TABLE users; --` → 이스케이핑 확인
  - `$1` 플레이스홀더 포함 문자열
  - NULL byte, unicode escape, backslash
  - bytea 바이너리 데이터
  - 매우 긴 문자열 (버퍼 오버플로우)
  - 중첩 이스케이핑 (`''''`)
- **완료 기준**: 모든 injection 페이로드가 리터럴 문자열로 처리됨

#### T19-7: 드라이버 호환성 테스트
- **범위**: 주요 PG 드라이버에서 multiplex 모드로 기본 CRUD + Prepared Statement 동작 확인
- **대상 드라이버**:
  - Go: `pgx`, `lib/pq`
  - Python: `psycopg2`, `asyncpg`
  - Java: `JDBC PostgreSQL`
  - Node.js: `pg` (node-postgres)
- **테스트 시나리오**: SELECT, INSERT, UPDATE, DELETE with parameters, Describe, DEALLOCATE
- **완료 기준**: 4개 이상 드라이버에서 기본 CRUD 정상 동작

#### T19-8: Multiplexing E2E 테스트 및 문서화
- **범위**: Docker 환경 E2E 테스트 — multiplex 모드에서 동시 10 클라이언트가 Prepared Statement 사용, 커넥션 다중화 확인. README에 설정 및 제한사항 문서화
- **제한사항 문서화**: 지원되지 않는 시나리오 (binary format result, COPY, 커서 등)
- **완료 기준**: E2E 테스트 통과, README 업데이트

</details>

---

### 우선순위 및 의존성

```
Phase 17 (OTel)       ← 독립, 즉시 착수 가능
Phase 18 (fsnotify)   ← 독립, Phase 11(SIGHUP Reload) 재사용
Phase 19 (PS Mux)     ← Phase 8(Transaction Pooling) 기반, PoC 선행 필수
```

### 리스크 매트릭스

| Phase | 구현 난이도 | 보안 리스크 | 외부 의존성 | 예상 Task 수 |
|-------|------------|------------|------------|-------------|
| 17 (OTel) | 중 | 낮음 | `go.opentelemetry.io/*` (3-4개) | 7 |
| 18 (fsnotify) | 낮 | 낮음 | `fsnotify` (1개) | 4 |
| 19 (PS Mux) | 높 | **높음** (SQL Injection) | 없음 | 8 |

---
---

## 고도화 추천 기능 (Phase 20+)

경쟁 제품(PgBouncer, PgCat, Odyssey) 대비 분석 및 오픈소스 생태계 관점에서 추천하는 기능 목록.

---

### 경쟁 제품 대비 현재 위치

| 기능 | PgBouncer | PgCat | Odyssey | **pgmux** |
|------|-----------|-------|---------|-----------|
| Transaction Pooling | O | O | O | **O** |
| R/W Splitting | X | O | O | **O** |
| Query Caching | X | X | X | **O** (차별점) |
| Query Firewall | X | X | X | **O** (차별점) |
| AST Parser | X | X | X | **O** (차별점) |
| Prepared Stmt Mux | X | X | X | **Phase 19** |
| Multi-DB | O | O | O | **미지원** (갭) |
| Auto Failover | X | O | X | **미지원** (갭) |
| Sharding | X | O | X | 미지원 |
| Audit Log | X | X | X | **O** (차별점) |
| Data API (HTTP) | X | X | X | **O** (차별점) |

pgmux는 **캐싱, 방화벽, AST 파서, Audit, Data API** 등 경쟁 제품에 없는 고유 기능이 다수.
반면 **Multi-Database**와 **Auto Failover**가 가장 큰 갭 — 오픈소스 채택률의 핵심.

---

### 1. 운영 안정성 (Production Readiness)

| 기능 | 설명 | 우선순위 |
|------|------|----------|
| **Graceful Shutdown + Connection Draining** | SIGTERM 수신 시 신규 연결 거부 + 기존 쿼리 완료 대기 후 종료. K8s `terminationGracePeriodSeconds`와 연동 필수 | **높음** |
| **Adaptive Pool Sizing** | 트래픽 패턴에 따라 `min_connections` ~ `max_connections` 사이에서 풀 크기를 자동 조절. 유휴 시간대 리소스 절약 | 중간 |
| **Connection Warming** | 프록시 시작 시 또는 Reader 추가 시 `min_connections`까지 백그라운드 사전 연결. 콜드 스타트 지연 제거 | 중간 |
| **Online Maintenance Mode** | `POST /admin/maintenance` — 신규 쿼리 거부 + 진행 중 쿼리 완료 대기. DB 마이그레이션/패치 시 유용 | 중간 |

---

### 2. 멀티테넌시 & 접근 제어 (Multi-Tenancy)

| 기능 | 설명 | 우선순위 |
|------|------|----------|
| **Multi-Database Routing** | 단일 프록시 인스턴스에서 여러 데이터베이스를 동시 프록시. StartupMessage의 `database` 필드로 분기. 현재는 단일 `backend.database`만 지원 | **높음** |
| **Per-User Connection Limits** | 사용자별/데이터베이스별 최대 커넥션 수 제한. `auth.users[].max_connections` | 중간 |
| **Per-User Rate Limiting** | 현재 전역 Rate Limit만 존재. 사용자별/IP별 차등 제한 (`rate_limit.per_user`) | 중간 |
| **IP Allow/Deny List** | `pg_hba.conf` 스타일 접근 제어. CIDR 기반 허용/차단 리스트 | 중간 |

---

### 3. 오픈소스 킬러 피처 (Differentiators)

| 기능 | 설명 | 우선순위 |
|------|------|----------|
| **Query Mirroring (Shadow Traffic)** | 프로덕션 쿼리를 복제하여 테스트 DB에도 전송. 결과 비교 없이 fire-and-forget. DB 마이그레이션/인덱스 변경 전 영향 분석에 필수적 | **높음** |
| **Query Rewriting Rules** | 정규식 또는 AST 기반 쿼리 변환 규칙. 예: `SELECT *` → 특정 컬럼으로 치환, deprecated 테이블명 자동 변환. 무중단 스키마 마이그레이션 지원 | 중간 |
| **Read-Only Mode** | `POST /admin/readonly` — 모든 쓰기 쿼리를 프록시 단에서 즉시 거부. Writer 장애 또는 긴급 유지보수 시 서비스 가용성 유지 | 중간 |
| **Query Tagging & Routing Rules** | `/* app:payment, priority:high */` 같은 태그로 라우팅 규칙 정의. 특정 앱/마이크로서비스의 쿼리를 전용 Reader로 고정 | 중간 |

---

### 4. 관측성 강화 (Observability)

| 기능 | 설명 | 우선순위 |
|------|------|----------|
| **Grafana Dashboard 템플릿** | `deploy/grafana/` 디렉토리에 JSON 대시보드 제공. 커넥션 풀, 캐시 히트율, 쿼리 레이턴시, 방화벽 차단 등 한눈에 확인 | **높음** |
| **Query Digest / Top-N Queries** | 쿼리를 정규화(`$N` 치환)하여 패턴별 실행 횟수, 평균/P99 레이턴시 집계. `GET /admin/queries/top` | 중간 |
| **Connection 추적 대시보드** | `GET /admin/connections` — 현재 활성 세션 목록 (클라이언트 IP, 실행 중 쿼리, 지속 시간). `pg_stat_activity`의 프록시 버전 | 중간 |
| **Structured JSON Logging** | 현재 `slog` 사용 중이지만 출력 포맷 설정 추가 (`log.format: json|text`). 로그 수집기(Loki, ELK)와 연동 용이 | 낮음 |

---

### 5. 고가용성 & 자동 장애 대응 (HA)

| 기능 | 설명 | 우선순위 |
|------|------|----------|
| **Auto Failover (Writer)** | Writer 장애 감지 시 설정된 standby를 자동 promote (또는 새 Writer 주소로 전환). Patroni/Stolon 연동 또는 독립 구현 | **높음** |
| **DNS-based Service Discovery** | Reader 목록을 DNS SRV 레코드에서 자동 갱신. K8s headless service, AWS RDS Proxy 스타일. 수동 설정 변경 불필요 | 중간 |
| **Health Check Endpoint (LB용)** | `/healthz` (liveness), `/readyz` (readiness) 분리. 현재 Admin API의 `/admin/health`는 있지만 별도 포트/경로로 LB 전용 엔드포인트 필요 | 중간 |

---

### 6. 개발자 경험 & 오픈소스 생태계 (DX & Community)

| 기능 | 설명 | 우선순위 |
|------|------|----------|
| **CLI 관리 도구 (`pgmuxctl`)** | `pgmuxctl stats`, `pgmuxctl cache flush`, `pgmuxctl reload` 등. Admin API의 CLI 래퍼. 운영자 UX 대폭 개선 | **높음** |
| **공식 Docker Image (GHCR)** | GitHub Actions CI/CD + `ghcr.io/org/pgmux:latest` 자동 빌드/푸시. Dockerfile은 있지만 배포 파이프라인 미구축 | **높음** |
| **벤치마크 Suite & 비교 문서** | PgBouncer, PgCat 대비 성능 벤치마크. `make bench-compare`로 재현 가능한 결과. 오픈소스 선택 시 가장 먼저 보는 자료 | **높음** |
| **문서 사이트 (GitHub Pages)** | MkDocs 또는 Hugo 기반. 설정 레퍼런스, 아키텍처 가이드, 마이그레이션 가이드(PgBouncer → pgmux) | 중간 |
| **CONTRIBUTING.md + Issue Templates** | 컨트리뷰터 가이드, PR 템플릿, 이슈 템플릿. 오픈소스 커뮤니티 참여 진입장벽 낮추기 | 중간 |
| **GitHub Actions CI** | PR마다 lint + unit test + E2E test + 벤치마크 리그레션 자동 실행 | 중간 |

---

### 7. 고급 기능 (Advanced / Long-term)

| 기능 | 설명 | 우선순위 |
|------|------|----------|
| **Read Replica Auto-Discovery** | AWS RDS API / K8s API를 통해 Reader 목록 자동 갱신. 수동 설정 제거 | 낮음 |
| **Sharding (Horizontal Partitioning)** | 테이블별 샤딩 키 기반 라우팅. `shard_key: user_id`. 장기 로드맵 | 낮음 |
| **Query Result Streaming** | 대용량 SELECT 결과를 스트리밍 방식으로 전달. 현재는 전체 버퍼링 후 캐시/전달 | 낮음 |
| **Plugin System** | 쿼리 처리 파이프라인에 사용자 정의 미들웨어 삽입. Go plugin 또는 Wasm | 낮음 |

---

### 추천 실행 로드맵

```
Phase 17: OpenTelemetry              ← 기획 완료, 즉시 착수
Phase 18: fsnotify Auto-Reload       ← 기획 완료, 독립 진행
Phase 19: PS Multiplexing PoC        ← 기획 완료, PoC 선행

── 여기까지 기존 기획 ──

Phase 20: OSS Release Ready          ← GitHub Actions CI + Docker Image + 벤치마크
Phase 21: Multi-Database Routing     ← 단일 인스턴스 다중 DB (가장 요청 많을 기능)
Phase 22: Grafana + Query Digest     ← 관측성 강화
Phase 23: Query Mirroring            ← 차별화 킬러 피처
Phase 24: Auto Failover + DNS SD     ← 고가용성
Phase 25: pgmuxctl CLI               ← 운영자 UX
Phase 26: Query Rewriting Rules      ← 무중단 마이그레이션 지원
Phase 27: Multi-Tenancy              ← Per-User Limits + IP ACL
Phase 28: Advanced (Sharding 등)     ← 장기 로드맵
```
