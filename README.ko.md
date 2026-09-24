# Castor

[English](README.md) | [简体中文](README.zh-CN.md) | [日本語](README.ja.md) | **한국어**

**바이브 코딩을 위한 풀스택 스캐폴드입니다.**

Castor는 Go + Next.js로 구축된, 여러분과 AI 코딩 에이전트가 바로 확장할 수 있는 애플리케이션 기반을 제공합니다. 비즈니스 요구 사항을 자연어로 설명하고, 이미 갖춰진 인증, 권한, 파일 관리, 국제화, 배포 도구 위에 기능을 구축하여 아이디어를 실제로 동작하는 풀스택 애플리케이션으로 만들 수 있습니다.

## 바이브 코딩에 Castor가 적합한 이유

- **에이전트에게 명확한 지침 제공**: 루트와 애플리케이션별 `AGENTS.md` 파일이 아키텍처, 코드 패턴, API 계약, 모듈 추가 체크리스트를 정의하며, 여러분과 AI 코딩 에이전트가 함께 따릅니다.
- **완성된 기능에서 출발해 확장**: 공개 웹사이트와 관리자 대시보드에 사용자, 역할, 메뉴, 에셋, 알림, 딕셔너리, 감사 로그가 포함되어 있어 새 비즈니스 모듈의 구현 참고 자료가 됩니다.
- **계층을 넘나드는 제약을 검사에 맡기기**: 단일 검증 진입점이 포맷팅, 정적 분석, 테스트, 프런트엔드 빌드를 모두 다룹니다. 가드레일 테스트는 아키텍처 의존성, RBAC 리소스 등록, 딕셔너리 계약, 4개 언어 번역을 검사합니다.
- **개발에서 배포까지 빠르게**: VS Code에서 F5를 누르면 로컬 환경을 준비하고 두 디버거를 모두 시작하며, Compose / Kubernetes 배포 및 백업 도구도 포함되어 있습니다.

## AI로 구축하기

1. 아래 빠른 시작에 따라 프로젝트를 실행하고 내장 기능을 먼저 살펴봅니다.
2. AI 코딩 에이전트에게 [AGENTS.md](AGENTS.md)와 관련 애플리케이션 가이드를 읽게 한 뒤, 비즈니스 요구 사항, 필드, 권한, 인수 기준을 설명합니다.
3. 기존 모듈을 참고해 데이터베이스, API, 권한, 메뉴, 프런트엔드, 4개 언어 번역을 구현하고 `scripts/check.py`를 실행해 변경 사항을 검증합니다.

예를 들어 다음과 같은 요청으로 시작할 수 있습니다.

```text
AGENTS.md와 백엔드 및 프런트엔드 개발 가이드를 읽어 주세요. 기존의
페이지네이션 CRUD 모듈을 참고하여 이름, 설명, 상태를 가진 프로젝트 관리
기능을 추가하고, 목록 조회, 필터링, 생성, 수정, 삭제를 지원해 주세요.
데이터베이스 마이그레이션, API 엔드포인트, RBAC 리소스, 메뉴, 상태 딕셔너리,
프런트엔드 페이지, 중국어/영어/일본어/한국어 번역, 테스트를 포함하고
scripts/check.py를 실행해 변경 사항을 검증해 주세요.
```

## 프로젝트 구조

```text
apps/
├── api/                     # Go API
└── web/                     # Next.js 웹 애플리케이션

deploy/
├── config/                  # 공용 API 설정 템플릿
├── compose/                 # 단일 호스트, 공용 다중 인스턴스, 로컬 인프라
└── k8s/                     # Kubernetes Kustomize base + overlays

scripts/
├── check.py                 # 개발자와 AI 에이전트를 위한 통합 검증
├── dev.py                   # 로컬 환경 준비 (F5에서 사용)
├── build-images.py          # 이미지 빌드, 푸시, 내보내기
├── deploy-compose.py        # 로컬 및 원격 Compose 배포
├── compose-instance.py      # 공용 인프라: 인스턴스 설정, 프로비저닝, 백업/복원
├── deploy-k8s.py            # Kubernetes 초기화 및 배포
└── backup.py                # 자체 호스팅 Compose 배포의 백업 및 복원
```

## 기술 스택

### 백엔드

| 구성 요소 | 기술 |
|-----------|------------|
| 언어 | Go 1.26.0+ |
| 웹 프레임워크 | Gin |
| ORM | GORM |
| 데이터베이스 | PostgreSQL |
| 캐시 | Redis |
| 오브젝트 스토리지 | S3 (RustFS 호환) |
| 접근 제어 | RBAC3 (역할 계층, SSD, DSD, 권한 부여 세션) |
| 인증 | JWT |
| 의존성 주입 | Wire |

### 프런트엔드

| 구성 요소 | 기술 |
|-----------|------------|
| 프레임워크 | Next.js 16 (App Router) |
| UI | React 19 + shadcn/ui + Tailwind CSS 4 |
| 상태 관리 | Zustand |
| 데이터 페칭 | TanStack Query |
| 폼 | TanStack Form |
| 테이블 | TanStack Table |

## 빠른 시작

### VS Code로 디버그하기 (F5)

VS Code에서 저장소 루트를 열고 권장 Go 확장을 설치한 다음, `Castor` 디버그 구성을 선택한 상태에서 **F5**를 누릅니다. Docker Desktop, Go, Node.js 22+, Bun, Python 3.11+, Chrome을 먼저 설치해야 하며, 데이터베이스 서비스는 직접 설치할 필요가 없습니다.

Windows에서는 Python과 함께 Python 런처(`py`)를 설치합니다. VS Code 작업은 오래된 Windows Store `python3` 별칭을 피하기 위해 `py -3`을 사용합니다. `py -3 --version`으로 선택된 버전을 확인하고(3.11+ 필요), 아래의 수동 명령에서는 `python3`을 `py -3`으로 바꿔 실행합니다.

첫 실행 시 PostgreSQL, Redis, RustFS, 스토리지 버킷, 개발용 설정을 생성하고 의존성을 설치한 뒤 데이터베이스를 초기화하고, Go / Next.js 디버거를 시작한 다음 브라우저를 엽니다. 이후 실행에서는 설정, 키, 데이터를 재사용하며, 데이터베이스 초기화는 반복 실행해도 됩니다. Windows, macOS, Linux에서 같은 방식으로 개발합니다. Docker Desktop이 실행 중이 아니면 자동으로 시작되며, Linux에서 Docker Engine을 사용할 때는 서비스를 직접 시작합니다.

- 프런트엔드 기본 주소는 `http://127.0.0.1:3000`, API 기본 주소는 `http://127.0.0.1:1234`입니다. 포트가 사용 중이면 사용 가능한 포트가 자동으로 선택되므로 실제 주소는 준비 작업 출력에서 확인합니다.
- 기본 관리자는 `system`입니다. 초기 비밀번호는 `.local/dev/settings.json`의 `admin_password` 필드에 저장되며, 기존 계정의 비밀번호는 재설정되지 않습니다.
- `.local/dev/`는 Git에서 무시되며 개발용 키, API 설정, 디버거 환경 변수를 담고 있습니다. 스크립트는 연결 주소 등 관리 대상 필드를 업데이트하며, 그 밖의 API 설정은 `api.config.toml`에서 조정할 수 있습니다.
- **Shift+F5**를 누르면 두 디버거가 모두 중지됩니다. 인프라는 다음 세션을 위해 계속 실행됩니다. 데이터 볼륨을 유지한 채 컨테이너를 중지하려면 “Tasks: Run Task → Castor: Stop services” 또는 `python3 scripts/dev.py stop`을 사용합니다.
- 환경만 준비하려면 `python3 scripts/dev.py prepare`를 실행합니다. 스크립트는 중국어, 영어, 일본어, 한국어를 지원하며 `CASTOR_DEV_LANG`으로 언어를 선택합니다.

저장소 경로마다 별도의 Compose 프로젝트와 데이터 볼륨을 사용합니다. 인프라 포트는 localhost에만 바인딩되며 자동으로 할당됩니다. Go 디버깅은 `CASTOR_CONFIG_FILE`로 생성된 설정 파일을 찾습니다.

### VS Code 없이 실행하기

Docker, Go 1.26+, Node.js 22+, Bun, Python 3.11+가 필요합니다.

```bash
python3 scripts/dev.py prepare     # 인프라, 설정, 의존성, init-db
python3 scripts/dev.py run api   # 터미널 1: Go API
python3 scripts/dev.py run web   # 터미널 2: Next.js
```

`dev.py run`은 `prepare`가 선택한 포트와 연결 설정(`.local/dev/debug.env`)으로 각 프로세스를 시작합니다. 셸에 의존하지 않으므로 Windows, macOS, Linux에서 같은 명령을 사용합니다.

## 개발 가이드라인

개발 지침은 AGENTS.md 파일에 정리되어 있으며, 사람 기여자와 AI 코딩 에이전트(Claude Code, Codex, Cursor 등)가 함께 따릅니다.

- [AGENTS.md](AGENTS.md): 작업 원칙, 엔드투엔드 모듈 체크리스트, 보안 불변 조건, API 계약, 커밋 규칙
- [apps/api/AGENTS.md](apps/api/AGENTS.md): 백엔드 계층, RBAC/메뉴, 마이그레이션, 테스트 규칙
- [apps/web/AGENTS.md](apps/web/AGENTS.md): 프런트엔드 데이터 계층, 페이지, 권한, i18n, 테스트 규칙
- [deploy/README.ko.md](deploy/README.ko.md): 이미지 빌드, Compose / Kubernetes 배포, 백업 및 복원
- [SECURITY.ko.md](SECURITY.ko.md): 보안 정책

커밋하기 전에 통합 검사를 실행합니다.

```bash
python3 scripts/check.py          # 또는 python3 scripts/check.py api | web | scripts
python3 scripts/check.py vuln     # 의존성 취약점 스캔, 릴리스 전 또는 정기적으로 실행
```

클론한 뒤 저장소 루트에서 `bun install`을 한 번 실행해 Git hooks를 활성화합니다. pre-commit은 스테이징된 파일을 포맷팅하고, pre-push는 프런트엔드가 변경된 경우 이를 빌드하고 변경된 Go 패키지를 테스트합니다.

## 내장 기능

- 다양한 인증 방식 (비밀번호 / 이메일 / 휴대폰 번호)
- RBAC3 접근 제어 (다중 역할 상속, SSD/DSD, 세션 역할 활성화)
- 파일 및 에셋 관리 (S3 호환 스토리지)
- 감사 로그
- 온라인 세션과 강제 로그아웃, 비밀번호 정책(길이, 복잡도, 유효 기간)
- 부서 트리와 역할 데이터 범위(전체 / 소속 부서 및 하위 / 소속 부서 / 지정 부서 / 본인만)
- 동적 시스템 설정
- 데이터 딕셔너리
- 알림
- 국제화 (중국어 / 영어 / 일본어 / 한국어)

## 라이선스

[MIT](LICENSE). 프런트엔드는 [next-shadcn-dashboard-starter](https://github.com/Kiranism/next-shadcn-dashboard-starter)(MIT)를 기반으로 수정했습니다. 서드파티 고지는 [NOTICE](NOTICE)를 참조하십시오.
