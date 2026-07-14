#!/bin/sh
# Conventional Commits 헤더 검증. $1 = 커밋 메시지 파일 경로(lefthook {1}).
# 의존성 0 — sh + grep만. type(scope)!: 요약 형식과 헤더 길이만 강제한다.
set -eu

# 주석(#)·빈 줄을 걷어낸 첫 줄 = 헤더.
header=$(grep -vE '^[[:space:]]*#' "$1" | grep -vE '^[[:space:]]*$' | head -n 1)

# merge/revert/fixup/squash 자동 커밋은 통과.
case "$header" in
"Merge "* | "Revert "* | fixup!* | squash!*) exit 0 ;;
esac

# type(scope)!: subject — 스펙 type, 선택 scope, 선택 !, ": ", 비어있지 않은 subject.
pattern='^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\([a-z0-9._-]+\))?(!)?: .+'

if ! printf '%s' "$header" | grep -qE "$pattern"; then
	echo "✗ Conventional Commits 형식이 아닙니다." >&2
	echo "  헤더: $header" >&2
	echo "  형식: type(scope): 요약   예) feat(tui): 서버 상태판 추가" >&2
	echo "  type: feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert (!로 breaking)" >&2
	exit 1
fi

# 헤더 길이 캡(문자 기준 — 한글 대응). 72자 초과 거부.
len=$(printf '%s' "$header" | LC_ALL=en_US.UTF-8 wc -m | tr -d ' ')
if [ "$len" -gt 72 ]; then
	echo "✗ 헤더가 너무 깁니다(${len} > 72자): $header" >&2
	exit 1
fi
