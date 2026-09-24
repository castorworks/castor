# 보안 정책

[English](SECURITY.md) | [简体中文](SECURITY.zh-CN.md) | [日本語](SECURITY.ja.md) | **한국어**

## 지원 버전

보안 수정은 최신 릴리스와 `main` 브랜치에만 제공됩니다.

## 취약점 보고

보안 문제는 공개 issue로 **등록하지 마십시오**.

[GitHub Security Advisories](https://github.com/castorworks/castor/security/advisories/new)를 통해 비공개로 보고해 주십시오. 영향을 받는 버전 또는 커밋, 재현 절차, 영향 범위를 포함해 주십시오.

보고는 영업일 기준 3일 이내에 접수를 확인하고, 확인된 고위험 문제는 30일 이내에 수정 또는 완화 조치를 제공하는 것을 목표로 합니다. 보고자가 원하지 않는 경우를 제외하고 권고문에 보고자를 명시합니다.

## 배포 체크리스트

- `Development = false`로 설정하고 모든 예시 비밀 값(`JwtKey`, RSA 키 비밀 값, 데이터베이스 / Redis / S3 자격 증명)을 교체합니다.
- `init-db`용 `CASTOR_DEFAULT_ADMIN_PASSWORD`를 설정하고, 최초 로그인 후 관리자 비밀번호를 변경합니다.
- API와 Web 앞단에서 TLS를 종료하고, 로드 밸런서에 맞게 `TrustedProxies`를 설정합니다.
- PostgreSQL, Redis, S3는 사설 네트워크에 둡니다.
