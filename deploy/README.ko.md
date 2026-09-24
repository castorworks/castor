# 배포 가이드

[English](README.md) | [简体中文](README.zh-CN.md) | [日本語](README.ja.md) | **한국어**

Castor는 Docker Compose와 Kubernetes 두 가지 배포 방식을 제공합니다. 두 방식 모두 동일한 API / Web 이미지를 사용하며, 같은 도메인에서 `/api/v1/*`는 API로, 그 밖의 요청은 Web으로 분배합니다. Dockerfile은 `apps/api`와 `apps/web`에 있습니다.

`scripts/`의 스크립트에는 Python 3.9+(표준 라이브러리만)와 스크립트가 호출하는 도구(Docker, kubectl, ssh)만 있으면 되며, Windows, macOS, Linux에서 똑같이 동작합니다. Windows에서는 `python3` 대신 `py -3`으로 실행합니다. `cp` / `chmod` 예시는 Linux와 macOS 호스트용이며, Windows에서는 평소처럼 파일을 복사하고 권한은 NTFS에 맡깁니다.

## 릴리스 모델

1. 명시적인 버전 태그를 붙여 두 이미지를 빌드하며, 이미 게시된 태그를 덮어쓰지 않습니다.
2. PostgreSQL, Redis, S3와 API 설정을 준비합니다.
3. 현재 API 이미지로 `init-db`를 실행합니다.
4. 초기화가 성공하면 API와 Web을 업데이트하고 준비될 때까지 기다립니다.

`init-db`는 PostgreSQL 트랜잭션 안에서 advisory lock을 잡고 테이블 구조 마이그레이션을 실행하며, 시스템 설정, 딕셔너리, 권한 리소스, 역할, 메뉴, 관리자에 대해 추가만 하고 변경하지 않는 대조를 수행합니다(최초 실행 시에는 배포 식별자도 생성합니다). 실패하면 롤백하고 0이 아닌 종료 코드를 반환합니다. 반복 실행해도 관리자 비밀번호는 재설정되지 않습니다. 일반 API 시작 시에는 기존 데이터베이스와 권한 설정만 읽으며, 마이그레이션을 실행하지 않습니다.

이미지 업데이트와 데이터베이스 변경은 별개의 작업입니다. 호환되지 않는 데이터베이스 변경이 포함된 경우 유지 보수 시간을 잡고 데이터를 백업해야 하며, 롤링 업데이트에만 의존해서는 안 됩니다. 이전 이미지가 새 데이터베이스와 호환된다는 보장은 없습니다.

## 테넌시 모델 선택

Castor의 배포 하나는 조직 하나를 담당합니다. 여러 고객(테넌트)을 수용하려면 테넌트마다 인스턴스를 하나씩 운영합니다. 데이터베이스와 역할, 버킷, Redis 키 접두사, 시크릿, 도메인이 테넌트마다 독립적이며, 인프라는 공유하거나 전용으로 둘 수 있습니다(Compose는 아래 "여러 인스턴스의 인프라 공유", Kubernetes는 테넌트별 인스턴스 overlay). 두 테넌트가 데이터베이스를 공유하지 않으므로 어떤 쿼리도 테넌트 간에 데이터를 유출할 수 없고, 백업·복원·이전·삭제도 테넌트 단위로 할 수 있습니다.

| 요구 사항 | 권장 방식 |
|------|------|
| 한 조직의 부서나 자회사가 자기 데이터만 보기 | 인스턴스 하나 + 부서와 역할 데이터 범위 |
| 몇 곳에서 수십 곳의 고객, 강한 격리 필요(공공, 의료, 금융) | 테넌트마다 인스턴스 하나 |
| 가입 즉시 개설되는 수백 개의 셀프서비스 테넌트, 테넌트 간 과금 | 하위 제품에서 행 수준 멀티테넌시를 직접 구현(제공하지 않음) |

인스턴스마다 고정 비용(API·Web 프로세스, 데이터베이스 연결, 아래 `max_connections` 참고)이 들고 업그레이드도 인스턴스별로 진행합니다. 아래 일괄 명령으로 이를 관리하기 쉽게 했습니다. 행 수준 멀티테넌시(공유 데이터베이스에서 모든 행에 `tenant_id`를 두는 방식)는 모든 하위 프로젝트가 그 비용과 위험을 떠안게 되므로 의도적으로 Castor에 포함하지 않았습니다. 이것이 필요한 제품은 최소한 다음을 계획해야 합니다.

- 모든 테이블에 `tenant_id`를 추가하고 고유 제약을 복합 제약으로 바꾸며(사용자 이름, 연락처, 역할·부서·사전 코드, 설정 키), 어떤 저장소 쿼리나 원시 SQL도 우회할 수 없는 테넌트 필터를 둘 것(GORM 스코프, 심층 방어로 PostgreSQL 행 수준 보안)
- 시스템 테이블마다 데이터가 전역인지 테넌트별인지 정하고(메뉴, 리소스, 사전, 설정) 그에 맞게 시드 조정을 다시 작성할 것
- 테넌트 단위의 자산 중복 제거(전역 콘텐츠 해시는 다른 테넌트가 같은 파일을 가지고 있음을 드러냄)
- 로그인 시 테넌트 식별(도메인 또는 코드)과 JWT, 세션, Redis 키, 속도 제한에 테넌트 반영
- "전체" 알림, 대시보드, 데이터 범위를 테넌트 안으로 한정하고, 테넌트 간 운영자는 별도로 권한을 부여할 것

## 이미지 빌드

Docker + Buildx가 필요합니다. 기본적으로 `linux/amd64`로 빌드하며, ARM 호스트에서 로컬 검증할 때는 `PLATFORM=linux/arm64`를 설정합니다. API 빌드는 기본적으로 공식 Go/Ubuntu 소스를 사용하며, 이 소스에 접근할 수 없는 빌드 머신(예: 중국 내)에서는 스크립트를 실행할 때 환경 변수 `GOPROXY`, `UBUNTU_MIRROR`, `UBUNTU_SECURITY_MIRROR`를 설정하면 빌드 인자로 전달됩니다(예: `GOPROXY=https://goproxy.cn,direct`).

```bash
# 로컬에 이미지 로드
python3 scripts/build-images.py load castor local

# 이미지 레지스트리에 푸시 (빌드 머신과 배포 머신에서 각각 docker login 실행)
python3 scripts/build-images.py push registry.example.com/castor v1.0.0

# 오프라인 전달용 애플리케이션 이미지
python3 scripts/build-images.py tar castor v1.0.0 deploy/artifacts/castor-v1.0.0.tar.gz
```

Web은 기본적으로 Node Dockerfile을 사용하여 런타임과 헬스 체크 명령이 일치하도록 합니다. 프로덕션 라우팅은 진입점에서 처리하므로 환경별 도메인을 프런트엔드 이미지에 넣을 필요가 없습니다. `NEXT_PUBLIC_*`를 커스터마이즈하는 경우에는 여전히 빌드 설정에 해당합니다.

## Compose

### 설정

```bash
cp deploy/compose/.env.example deploy/compose/.env
cp deploy/config/api.config.example.toml deploy/compose/config/api.config.toml
chmod 600 deploy/compose/.env
```

`.env`의 비어 있는 키를 모두 채웁니다. JWT는 32바이트 이상을 권장하고, RSA는 정확히 32개의 ASCII 바이트여야 하며, 관리자 비밀번호는 최소 12자여야 합니다. JWT 등의 키는 `python3 -c "import secrets; print(secrets.token_hex(32))"`로 생성하고, RSA 키는 같은 명령에서 32를 16으로 바꿔 생성합니다. 항목마다 서로 다른 값을 사용합니다. 실제 `.env`는 커밋하지 않습니다.

SMS 및 이메일 인증 코드는 선택 사항입니다. 사용하지 않을 경우 `.env`의 `CASTOR_SMS_*`와 TOML의 `[Mail].Host`를 비워 두면 서비스는 정상적으로 시작되며, 사용자가 해당 방식을 선택하면 설정되지 않았다는 안내가 표시됩니다. SMS를 활성화하려면 `CASTOR_SMS_ACCESS_KEY`, `CASTOR_SMS_SECRET_KEY`와 TOML의 `SignName`, `TemplateCode`를 함께 채워야 하고, 이메일을 활성화하려면 `[Mail]`의 Host와 계정을 채우고 비밀번호는 `CASTOR_MAIL_PASSWORD`로 주입합니다. K8s에서 필요하면 같은 이름의 변수를 `secrets.env`에 추가합니다.

`CASTOR_INSTANCE_ID`는 인스턴스 식별자(Redis key 접두사이자 JWT `aud`)이며, 단일 인스턴스라면 기본값 `castor`를 그대로 두면 됩니다. 다른 배포와 Redis를 공유하는 경우 아래 '다중 인스턴스의 인프라 공유'를 참조하십시오. `API_IMAGE` / `WEB_IMAGE`는 빌드 결과와 일치해야 합니다. `S3_BUCKET`은 API 설정과 자체 호스팅 버킷 초기화에 모두 사용되며, 초기화는 비공개 버킷만 생성합니다.

API 일반 설정은 `config/api.config.toml`에 두고, 키는 `.env`로 주입합니다. API는 UID 10001로 실행되므로 마운트한 TOML을 이 사용자가 읽을 수 있어야 합니다. 파일에는 민감하지 않은 설정만 저장하고 0644 권한을 사용하는 것을 권장합니다. 컨테이너 로그는 stdout과 임시 디렉터리에 기록되며, 장기 로그는 플랫폼이 수집합니다.

### 단일 호스트 전체 배포

```bash
python3 scripts/deploy-compose.py config full
python3 scripts/deploy-compose.py pull full  # 로컬 빌드 이미지를 사용할 때는 건너뜀
python3 scripts/deploy-compose.py up full
```

PostgreSQL, Redis, RustFS를 시작하고 정상 상태가 될 때까지 기다린 뒤, 버킷을 생성하고, 일회성 데이터베이스 초기화를 실행하며, 마지막으로 API, Web, Nginx를 시작합니다. `up full`을 반복 실행하면 초기화가 다시 실행되므로, 이전 버전에서 종료된 컨테이너의 성공 상태를 재사용하지 않습니다.

기본 진입점은 `http://localhost:8080`입니다. Nginx만 포트를 게시하며, 데이터베이스와 애플리케이션 서비스는 호스트에 직접 게시되지 않습니다. HTTPS는 서버에 이미 있는 TLS 프록시에서 종료한 뒤 이 포트로 전달합니다. 기본적으로 `127.0.0.1`에만 바인딩되며, TLS 프록시가 다른 호스트에 있을 때만 `HTTP_BIND=0.0.0.0`을 설정합니다. 개발 모드가 아니면 로그인 쿠키에 `Secure`가 붙으므로 브라우저는 HTTPS나 `http://localhost`에서만 세션을 유지합니다. `http://<서버 IP>:8080` 같은 평문 주소로 로그인하면 성공한 것처럼 보이지만 로그인 페이지로 돌아갑니다. 로그인, 업로드, 다운로드는 모두 진입점 도메인을 사용합니다. 프록시는 원래 Host를 유지해야 하며, 신뢰 경계 안에서 전달 헤더를 설정하고, 업로드 한도는 최소 100 MB로 하며, 스트리밍 요청에 적절한 타임아웃과 버퍼링 정책을 설정해야 합니다.

### 외부 인프라 사용

`.env`의 데이터베이스, Redis, S3 주소와 자격 증명을 편집하고, TOML에서 데이터베이스 SSL, Redis TLS/Sentinel/Cluster, S3 Secure/Region 등의 옵션을 설정합니다(AWS S3의 `Region`은 버킷이 있는 리전과 일치해야 하며 자동으로 감지되지 않습니다). 외부 S3 버킷은 미리 생성해 두어야 합니다.

```bash
python3 scripts/deploy-compose.py config external
python3 scripts/deploy-compose.py up external
```

이 모드는 외부 인프라를 생성, 중지, 백업하지 않습니다. Compose의 `CASTOR_DB_PORT`로 기본값 5432를 재정의할 수 있습니다. 복잡한 Redis 토폴로지를 사용할 때는 설정 검증에 쓰이는 Address도 채워져 있는지 함께 확인해야 합니다.

### 다중 인스턴스의 인프라 공유

한 호스트에서 하나의 PostgreSQL, Redis, RustFS를 공유하며 여러 Castor(각각 독립된 API + Web + 게이트웨이)를 실행합니다. 격리 방식은 다음과 같습니다.

| 리소스 | 인스턴스별 전용 | 생성 주체 |
|------|--------------|----------|
| PostgreSQL | 데이터베이스 + 같은 이름의 역할(`REVOKE CONNECT FROM PUBLIC`, 다른 인스턴스의 역할은 접속 불가) | `compose-instance.py provision` |
| RustFS | 버킷 + 해당 버킷의 오브젝트만 읽고 쓸 수 있는 사용자(다른 버킷 접근 불가, 버킷 정책 변경 불가) | `compose-instance.py provision` |
| Redis | key 접두사 `CASTOR_INSTANCE_ID`(비밀번호 공유, 논리적 격리) | API 시작 시 점유 |
| 키 | `CASTOR_JWT_KEY`, `CASTOR_RSA_SECRET`, `CASTOR_DATA_KEY`, 관리자 초기 비밀번호 | `compose-instance.py new`가 무작위 생성 |

`CASTOR_INSTANCE_ID`는 JWT의 `aud`이기도 하므로, 한 인스턴스가 발급한 token은 다른 인스턴스에서 유효하지 않습니다. API는 시작 시 Redis 네임스페이스를 자체 데이터베이스의 배포 식별자(`init-db`가 생성) 명의로 등록합니다. 다른 배포가 실수로 같은 `CASTOR_INSTANCE_ID`를 사용하면 RSA 키, 딕셔너리 캐시, 로그인 속도 제한을 조용히 공유하는 대신 원인을 알리며 시작에 실패합니다.

```bash
# 1. 공용 인프라 (한 번만)
cp deploy/compose/.env.shared.example deploy/compose/.env.shared
chmod 600 deploy/compose/.env.shared      # POSTGRES_PASSWORD, REDIS_PASSWORD, S3_SECRET_KEY 입력
python3 scripts/deploy-compose.py up shared

# 2. 인스턴스마다: env 생성(무작위 키, DB 이름, 버킷), 필요에 따라 이미지, 포트, CORS 도메인 수정
python3 scripts/compose-instance.py new tenant-a 8081   # → deploy/compose/instances/tenant-a.env
python3 scripts/deploy-compose.py up instance deploy/compose/instances/tenant-a.env
```

`up instance`는 데이터베이스와 버킷을 차례로 프로비저닝하고(반복 실행 가능하며, 역할 비밀번호와 RustFS 사용자 키를 env의 값으로 동기화합니다), `init-db`를 실행한 뒤 애플리케이션을 시작합니다. 관리자 초기 비밀번호는 생성된 env 파일에 있습니다. 업데이트와 롤백은 단일 인스턴스 배포와 같으며, `full`을 `instance <env-file>`로 바꾸기만 하면 됩니다. `down instance <env-file>`은 해당 인스턴스만 중지합니다.

모든 인스턴스를 한 번에 조작하려면(예: 릴리스 배포) 일괄 명령을 사용합니다. 인스턴스 식별자 순서로 처리하며 첫 번째 실패에서 멈추므로, 일부만 업그레이드된 상태를 놓치지 않습니다. `--keep-going`은 계속 진행하지만 여전히 오류로 종료합니다.

```bash
python3 scripts/compose-instance.py status
python3 scripts/compose-instance.py all pull
python3 scripts/compose-instance.py all up
python3 scripts/compose-instance.py all backup
```

- 각 인스턴스의 게이트웨이는 각자 `HTTP_PORT`를 게시하고, 앞단의 TLS 프록시가 도메인별로 분배합니다. **인스턴스마다 반드시 별도의 도메인 또는 서브도메인을 사용해야 합니다**. 로그인 상태는 Domain이 없는 host-only cookie이므로, 같은 도메인에서 경로로 인스턴스를 구분하면 로그인 상태가 서로 덮어써집니다.
- API만 공용 네트워크 `SHARED_NETWORK`에 참여하여 서비스 이름 `postgres`, `redis`, `rustfs`로 인프라에 접근합니다. Web과 게이트웨이는 인스턴스 자체 네트워크에 남으며, `castor-api`는 해당 인스턴스로만 해석됩니다.
- 모든 인스턴스가 `config/api.config.toml`을 공유합니다(인스턴스 간 차이는 모두 env에 있습니다). 별도 설정이 필요하면 env에서 `API_CONFIG_FILE`을 설정합니다.
- PostgreSQL 연결 수: API 복제본 하나당 최대 `MaxOpenConns`(기본값 25)개의 연결을 사용하며, 인스턴스 수 × 복제본 수 × 25가 `max_connections`(기본값 100)를 넘으면 안 됩니다. 초과하면 `MaxOpenConns`를 줄이거나 상한을 높입니다.
- (key 격리뿐 아니라) Redis 권한 격리가 필요하면 외부 Redis를 사용하여 인스턴스마다 ACL 사용자(`~<instance-id>:* +@all -@dangerous`)를 만들고, TOML의 `[Redis]`에 `Username`을 입력합니다. 비밀번호는 여전히 `CASTOR_REDIS_PASSWORD`로 주입합니다.
- 원격 배포(`deploy-compose.py remote`)는 단일 인스턴스만 지원합니다. 다중 인스턴스는 대상 호스트에서 위 명령을 직접 실행합니다.

### 로컬 개발

로컬 개발에는 이 절의 스크립트를 사용하지 않습니다. `python3 scripts/dev.py prepare`가 독립된 인프라를 시작하고 설정을 생성하며, 자세한 내용은 루트 README를 참조하십시오.

### 원격 배포

먼저 로컬의 `deploy/compose/.env`와 `config/api.config.toml`을 준비합니다. 스크립트는 Compose 설정과 키를 SSH로 전송하고 원격 호스트에서 `docker compose`를 실행하며, 원격 초기화가 성공하면 애플리케이션을 시작합니다. 이미지는 자동으로 빌드하지 않습니다.

```bash
# 레지스트리 방식
python3 scripts/deploy-compose.py remote registry deploy@example.com /opt/castor full

# tar 방식 (애플리케이션 이미지를 미리 빌드/내보내기)
python3 scripts/deploy-compose.py remote tar deploy@example.com /opt/castor full deploy/artifacts/castor-v1.0.0.tar.gz
```

대상 머신에는 Docker Compose v2(`up/start --wait` 지원)와 tar만 있으면 되며, Python이나 Bash는 필요하지 않습니다. tar 패키지에는 애플리케이션 이미지만 포함되므로, 오프라인 환경에서는 Compose 매니페스트에 있는 PostgreSQL, Redis, RustFS, rc, Nginx 이미지도 미리 로드해야 합니다. 원격 호스트에 대한 최초 연결과 인증은 SSH 자체 메커니즘을 사용합니다.

### 검증 및 업데이트

```bash
curl -f http://localhost:8080/healthz
curl -f http://localhost:8080/api/v1/settings/public
docker compose --project-directory deploy/compose -f deploy/compose/compose.yaml ps
```

API 컨테이너의 준비 상태 검사는 `/ready`에 접근하여 PostgreSQL과 Redis를 확인합니다. 실제 인수 검증에서는 오브젝트 스토리지 경로까지 확인할 수 있도록 로그인, 파일 업로드, 다운로드도 수행해야 합니다. 또한 자신에게 알림을 보내 벨이 즉시 갱신되는지 확인하세요. 갱신되지 않으면 앞단 프록시가 이벤트 스트림(`/api/v1/account/notifications/stream`)을 버퍼링하고 있는 것입니다.

이후 릴리스에서는 두 이미지의 버전을 수정한 뒤 `pull`, `up`을 실행합니다. `up`은 애플리케이션 컨테이너를 다시 생성하여 TOML과 프록시 설정 변경이 반영되도록 합니다. Compose 업데이트 중에는 잠시 중단이 발생할 수 있습니다.

애플리케이션을 롤백할 때는 먼저 데이터베이스 호환성을 확인한 뒤, 두 이미지 참조와 설정을 이전 버전으로 되돌립니다.

```bash
python3 scripts/deploy-compose.py pull full
python3 scripts/deploy-compose.py rollback full
```

`rollback`은 애플리케이션만 업데이트하며, 이전 버전의 데이터베이스 초기화는 실행하지 않습니다.

## Kubernetes

기존 클러스터, kubectl, 해당 클러스터를 지원하는 인그레스 컨트롤러, 외부 PostgreSQL/Redis/S3, 이미지 레지스트리 접근 권한이 필요합니다. 이 디렉터리는 데이터베이스 클러스터와 스토리지 시스템을 관리하지 않습니다.

### 환경 준비

```bash
cp deploy/config/api.config.example.toml deploy/k8s/overlays/staging/api.config.toml
cp deploy/k8s/overlays/staging/secrets.env.example deploy/k8s/overlays/staging/secrets.env
cp deploy/k8s/overlays/staging/bootstrap.env.example deploy/k8s/overlays/staging/bootstrap.env
chmod 600 deploy/k8s/overlays/staging/{secrets,bootstrap}.env
```

프로덕션 환경에서는 위의 `staging`을 `production`으로 바꿉니다.

각 overlay는 하나의 인스턴스(독립된 namespace)이며, `staging`과 `production`은 예시입니다. 테넌트마다 인스턴스를 하나씩 둘 때는 인스턴스 overlay를 생성합니다. `new`는 `production`을 `deploy/k8s/instances/<id>/`로 복사하고 namespace `castor-<id>`, 도메인, `CASTOR_INSTANCE_ID=<id>`를 설정하며, JWT 키, RSA 키, 관리자 초기 비밀번호를 생성합니다. 그런 다음 `secrets.env`와 TOML에 외부 자격 증명(독립된 데이터베이스와 역할, 버킷과 액세스 키)을 채웁니다. 빈 값이 있으면 `apply`가 실행을 거부합니다. `apply-all` / `rollback-all`은 모든 인스턴스를 식별자 순서로 처리하며 첫 번째 실패에서 멈춥니다(`--keep-going`은 계속 진행하지만 여전히 오류로 종료).

```bash
python3 scripts/deploy-k8s.py new tenant-a tenant-a.example.com
python3 scripts/deploy-k8s.py apply tenant-a your-kube-context
python3 scripts/deploy-k8s.py apply-all your-kube-context
```

- TOML: 외부 서비스 주소, 데이터베이스 이름/사용자, TLS, S3 버킷/리전, CORS 도메인을 설정합니다. 민감한 필드는 Secret으로 주입합니다.
- `secrets.env`: 런타임 키를 입력합니다. `bootstrap.env`에는 관리자 초기 비밀번호만 들어 있으며 초기화 Job만 사용합니다.
- `kustomization.yaml`: 두 이미지의 레지스트리와 버전, 도메인을 바꿉니다. 클러스터에 맞게 Ingress class와 비공개 레지스트리용 `imagePullSecrets`를 설정합니다(애플리케이션과 초기화 템플릿 모두에 설정해야 합니다).
- TLS: 대상 namespace에 `castor-tls`라는 이름의 TLS Secret을 미리 만들어 두거나, 기존 인증서 컨트롤러로 생성합니다.
- 실제 인그레스 컨트롤러에 맞게 최소 100 MB 업로드 한도, 스트리밍 요청, 타임아웃을 설정합니다. 기본 IngressClass가 없는 클러스터에서는 class를 명시적으로 지정해야 합니다.

ConfigMap / Secret은 내용 해시가 붙은 이름을 사용하므로, 설정이 바뀌면 Pod 템플릿이 업데이트되어 롤링 업데이트가 트리거됩니다. 실제 설정 파일과 Secret env 파일은 모두 Git에서 무시됩니다. `render` 출력에는 Secret 데이터가 포함되므로 커밋하거나 공개 로그에 올리지 마십시오.

### 배포

```bash
python3 scripts/deploy-k8s.py render staging deploy/artifacts/castor-staging.yaml
# 내용을 확인한 뒤 이 임시 파일을 삭제
python3 scripts/deploy-k8s.py apply staging your-kube-context
python3 scripts/deploy-k8s.py apply production your-kube-context
```

스크립트는 현재 클러스터를 잘못 사용하는 일을 막기 위해 context를 명시적으로 지정하도록 요구합니다. namespace 적용, 환경 릴리스 잠금 획득, 설정 및 초기화 템플릿 적용, 현재 버전 Job 생성, 완료 대기, Deployment/Ingress 적용, 롤링 업데이트 대기 순으로 진행합니다.

`castor-init-db`는 스케줄링이 비활성화된 CronJob으로, 매 릴리스 Job의 템플릿 역할만 하며 정해진 시간에 실행되는 일은 없습니다. 초기화가 시간 초과되거나 실패하면 릴리스를 중단하고 기존 애플리케이션 Deployment를 유지합니다. Job 로그는 문제 해결에 사용하며, Job은 하루 뒤 자동으로 정리됩니다.

같은 namespace에서의 스크립트 릴리스는 `castor-release-lock` ConfigMap으로 직렬화됩니다. 프로세스가 강제 종료되어 잠금이 남아 있으면, 먼저 실행 중인 릴리스 프로세스와 초기화 작업이 없는지 확인한 뒤 이 ConfigMap을 삭제합니다. 스크립트가 정상 종료되면 잠금을 정리합니다.

staging은 복제본 하나이며, production은 HPA가 복제본 수를 관리합니다(최소 2개). 매니페스트에 `replicas`를 쓰지 않으므로, `apply`를 반복해도 HPA가 늘린 복제본이 줄어들지 않습니다. API는 `/health`, `/ready`와 시작 프로브를 사용합니다. 종료 유예 기간은 30초로, 기본 애플리케이션 종료 시간 10초보다 깁니다. 리소스 요청/한도는 부하에 맞게 조정할 수 있습니다.

```bash
kubectl --context your-kube-context -n castor-staging get pods,jobs,ingress
kubectl --context your-kube-context -n castor-staging logs deployment/castor-api
curl -f https://castor-staging.example.com/api/v1/settings/public
```

### 업데이트 및 롤백

overlay의 이미지 버전과 설정을 업데이트한 뒤 `apply`를 실행합니다. 초기화에 파괴적 변경이 포함되면 먼저 유지 보수 시간을 잡습니다.

롤백 전에 이전 애플리케이션이 데이터베이스를 읽을 수 있는지 확인하고, overlay의 이미지와 설정을 이전 버전으로 되돌린 다음 실행합니다.

```bash
python3 scripts/deploy-k8s.py rollback production your-kube-context
```

이 명령은 릴리스 잠금을 획득하고 지정된 설정을 적용하여 애플리케이션을 업데이트하며, 초기화 Job은 생성하지 않습니다.
`kubectl rollout undo`는 Deployment만 처리하며 데이터베이스, Job, Ingress, 외부 서비스는 롤백하지 않습니다. 이전 ConfigMap/Secret은 문제 해결과 롤백을 위해 당분간 자동으로 삭제하지 않으며, Deployment의 이전 리비전에서 더 이상 참조하지 않는 것을 확인한 뒤 정리합니다.

## 백업 및 복원

`scripts/backup.py`는 Compose 자체 호스팅 데이터만 담당하며, 짧은 중단을 동반하는 물리 스냅샷으로 PostgreSQL, Redis(AOF 포함), RustFS의 모든 데이터 볼륨을 저장하고, 배포 설정도 별도로 아카이브합니다. 외부 서비스는 수정하지 않습니다. Compose 프로젝트 안의 실제 컨테이너를 읽으므로 고정된 컨테이너 이름에 의존하지 않습니다.

```bash
python3 scripts/backup.py create full
python3 scripts/backup.py list
python3 scripts/backup.py restore full snapshot-20260909_120000 --confirm snapshot-20260909_120000
```

백업/복원 전에 Alpine 도구 이미지를 가져온 뒤 프로젝트 서비스를 중지하고, 성공하면 원래 실행 중이던 서비스를 다시 시작합니다. 백업이 실패해도 서비스는 다시 시작되지만, 복원이 실패하면 일부만 복원된 데이터가 제공되지 않도록 중지 상태를 유지합니다.

물리 복원을 하려면 세 인프라 컨테이너의 이미지 ID가 백업 시점과 같아야 하며, 데이터베이스의 버전 간 업그레이드에는 전용 마이그레이션 도구를 사용합니다. 복원은 기존 데이터 볼륨을 덮어쓰므로 `--confirm <snapshot>`을 명시적으로 지정해야만 실행됩니다. `config.tar`는 대조용일 뿐 기존 배포 설정을 덮어쓰지 않으며, 자격 증명이 바뀐 경우에는 일치 여부를 직접 확인해야 합니다.

공용 인프라 전체의 콜드 스냅샷은 `full` 대신 `shared`를 사용합니다. 이 경우 PostgreSQL, Redis, RustFS가 중지되어 그동안 모든 인스턴스를 사용할 수 없으므로, 먼저 각 인스턴스를 중지해야 합니다. 단일 인스턴스를 백업하거나 복원할 때는 논리 스냅샷을 사용하며, 해당 인스턴스의 API만 중지합니다.

```bash
python3 scripts/compose-instance.py backup deploy/compose/instances/tenant-a.env
python3 scripts/compose-instance.py list deploy/compose/instances/tenant-a.env
python3 scripts/compose-instance.py restore deploy/compose/instances/tenant-a.env snapshot-20260921_120000 --confirm tenant-a/snapshot-20260921_120000
```

스냅샷에는 `pg_dump` 사용자 지정 형식의 데이터베이스, 버킷 미러, 당시의 env 파일이 포함되며 `deploy/backups/instances/<instance-id>/`에 저장됩니다. 복원 시 `pg_restore --clean`으로 해당 인스턴스의 데이터베이스를 덮어쓰고, `rc mirror --remove`로 버킷을 스냅샷과 일치시키며, 실패하면 API는 중지 상태로 유지됩니다. 같은 애플리케이션 버전으로만 복원하며, 버전이 다르면 먼저 복원한 뒤 `init-db`를 실행합니다. Redis에는 만료되는 런타임 데이터만 있으므로 백업하지 않습니다.

스냅샷은 `deploy/backups/`에 있으며 데이터와 키를 포함하고, 현재 사용자만 읽을 수 있는 디렉터리에 저장합니다. 안전한 백업 위치로 직접 옮겨 보관해야 합니다. K8s와 외부 데이터베이스는 해당 플랫폼의 백업/PITR, 오브젝트 스토리지 버전 관리를 사용하고, 정기적으로 복원을 검증해야 합니다.

## 저장소 검증

`scripts/check.py`(루트 AGENTS.md 참조). 이 중 `bootstrap` 통합 테스트는 `CASTOR_TEST_POSTGRES_DSN`을 사용하며, 반드시 별도의 빈 테스트 데이터베이스를 가리켜야 합니다. 이 테스트는 두 초기화 작업의 동시 실행, 반복 실행 시 관리자 비밀번호가 재설정되지 않는 것, 역할 할당이 중복되지 않는 것을 검증합니다.
