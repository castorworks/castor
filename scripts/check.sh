#!/usr/bin/env bash
# 仓库统一验证入口：本地提交前、AI 代理完成改动后、CI 均执行同一套检查。
# 用法：scripts/check.sh [all|api|web|scripts]...   （默认 all）
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

step() { printf '\n\033[1;34m==> %s\033[0m\n' "$*"; }

check_api() {
  cd "$ROOT/apps/api"
  step 'api: gofmt'
  unformatted="$(gofmt -l .)"
  if [[ -n "$unformatted" ]]; then
    printf '未格式化文件（运行 cd apps/api && gofmt -w .）：\n%s\n' "$unformatted" >&2
    exit 1
  fi
  step 'api: go vet'
  go vet ./...
  step 'api: go test -race (TZ=UTC)'
  # 与 CI 及生产镜像时区一致，避免依赖本机时区的测试只在本地通过
  TZ=UTC go test -race -count=1 ./...
}

check_web() {
  cd "$ROOT/apps/web"
  step 'web: format'
  bun run format:check
  step 'web: lint'
  bun run lint
  step 'web: typecheck'
  bunx tsc --noEmit
  step 'web: test'
  bun run test
  step 'web: build'
  bun run build
}

check_scripts() {
  cd "$ROOT"
  step 'scripts: unittest'
  python3 -m unittest discover -s scripts/tests -p '*_test.py'
}

targets=("$@")
[[ ${#targets[@]} -eq 0 ]] && targets=(all)

for target in "${targets[@]}"; do
  case "$target" in
    # 子 shell 不能放在 && 链中，否则 set -e 在函数内部失效
    all)
      (check_api)
      (check_web)
      (check_scripts)
      ;;
    api) (check_api) ;;
    web) (check_web) ;;
    scripts) (check_scripts) ;;
    *) echo "未知目标：$target（可选 all|api|web|scripts）" >&2; exit 2 ;;
  esac
done

step '全部检查通过'
