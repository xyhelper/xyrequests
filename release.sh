#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
用法:
  ./release.sh <version> [options]

示例:
  ./release.sh v1.2.3
  ./release.sh 1.2.3 --branch master
  ./release.sh v1.2.3 --skip-tests
  ./release.sh v1.2.3 --no-push

参数:
  <version>             语义化版本号，如 1.2.3 或 v1.2.3

选项:
  --branch <name>       指定发布分支，默认 master
  --remote <name>       指定远端仓库，默认 origin
  --skip-tests          跳过 go test ./...
  --allow-dirty         允许工作区有未提交改动（默认不允许）
  --no-push             只创建本地 tag，不推送
  -h, --help            显示帮助

发布动作:
  1) 校验版本号与环境
  2) 校验当前分支与工作区状态
  3) 拉取远端标签并检查标签是否已存在
  4) 执行 go test ./...（可跳过）
  5) 创建注解标签
  6) 推送分支和标签（可关闭）
EOF
}

require_cmd() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "错误: 缺少命令 '$cmd'" >&2
    exit 1
  fi
}

normalize_tag() {
  local v="$1"
  if [[ "$v" =~ ^v ]]; then
    echo "$v"
  else
    echo "v$v"
  fi
}

validate_semver() {
  local v="$1"
  local raw="$v"
  raw="${raw#v}"
  [[ "$raw" =~ ^[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$ ]]
}

main() {
  require_cmd git
  require_cmd go

  local branch="master"
  local remote="origin"
  local skip_tests=0
  local allow_dirty=0
  local no_push=0
  local version=""

  while [[ $# -gt 0 ]]; do
    case "$1" in
      -h|--help)
        usage
        exit 0
        ;;
      --branch)
        shift
        [[ $# -gt 0 ]] || { echo "错误: --branch 需要参数" >&2; exit 1; }
        branch="$1"
        ;;
      --remote)
        shift
        [[ $# -gt 0 ]] || { echo "错误: --remote 需要参数" >&2; exit 1; }
        remote="$1"
        ;;
      --skip-tests)
        skip_tests=1
        ;;
      --allow-dirty)
        allow_dirty=1
        ;;
      --no-push)
        no_push=1
        ;;
      --*)
        echo "错误: 未知选项 '$1'" >&2
        usage
        exit 1
        ;;
      *)
        if [[ -n "$version" ]]; then
          echo "错误: 仅允许一个版本号参数" >&2
          usage
          exit 1
        fi
        version="$1"
        ;;
    esac
    shift
  done

  if [[ -z "$version" ]]; then
    echo "错误: 缺少版本号参数" >&2
    usage
    exit 1
  fi

  if ! validate_semver "$version"; then
    echo "错误: 非法版本号 '$version'，示例: v1.2.3 或 1.2.3" >&2
    exit 1
  fi

  local tag
  tag="$(normalize_tag "$version")"

  if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    echo "错误: 当前目录不是 git 仓库" >&2
    exit 1
  fi

  local current_branch
  current_branch="$(git rev-parse --abbrev-ref HEAD)"
  if [[ "$current_branch" != "$branch" ]]; then
    echo "错误: 当前分支是 '$current_branch'，要求在 '$branch' 分支发布" >&2
    exit 1
  fi

  if [[ "$allow_dirty" -eq 0 ]]; then
    if [[ -n "$(git status --porcelain)" ]]; then
      echo "错误: 工作区有未提交改动，请先提交或使用 --allow-dirty" >&2
      exit 1
    fi
  fi

  if ! git remote get-url "$remote" >/dev/null 2>&1; then
    echo "错误: 远端 '$remote' 不存在" >&2
    exit 1
  fi

  echo "[1/5] 同步远端标签..."
  git fetch "$remote" --tags

  if git rev-parse -q --verify "refs/tags/$tag" >/dev/null 2>&1; then
    echo "错误: 本地标签 '$tag' 已存在" >&2
    exit 1
  fi

  if git ls-remote --exit-code --tags "$remote" "refs/tags/$tag" >/dev/null 2>&1; then
    echo "错误: 远端标签 '$tag' 已存在" >&2
    exit 1
  fi

  if [[ "$skip_tests" -eq 0 ]]; then
    echo "[2/5] 运行测试 go test ./..."
    go test ./...
  else
    echo "[2/5] 跳过测试 (--skip-tests)"
  fi

  echo "[3/5] 创建注解标签 $tag"
  git tag -a "$tag" -m "release $tag"

  if [[ "$no_push" -eq 1 ]]; then
    echo "[4/5] 已按要求跳过推送 (--no-push)"
    echo "完成: 已创建本地标签 $tag"
    echo "手动推送命令:"
    echo "  git push $remote $branch"
    echo "  git push $remote $tag"
    exit 0
  fi

  echo "[4/5] 推送分支到 $remote/$branch"
  git push "$remote" "$branch"

  echo "[5/5] 推送标签 $tag"
  git push "$remote" "$tag"

  echo "发布完成: $tag"
}

main "$@"
