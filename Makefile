# KAS 常用命令入口（T1.2.1.1.3）。目标与 CI（.github/workflows/ci.yml）同源；
# 改动命令时两处同步修改。
SHELL := /bin/bash
GO    ?= go
MODULE := github.com/kas-runtime/kas

# 覆盖率阈值：初始 60%（Story 1.2.2.1），随 Stage 递增至 D-8 目标
# （核心包 80%，P0 安全路径 90%，Stage 7 全量 80%）。
COVER_THRESHOLD ?= 60
GOLANGCI_VERSION ?= v1.61.0
GOFUMPT_VERSION ?= v0.7.0

.PHONY: help build test test-race lint cover fmt fmt-check tidy verify clean acceptance goldenset-freeze audit-deps

help: ## 列出可用目标
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

build: ## 编译全部包
	$(GO) build ./...

test: ## 跑全部测试（默认即含 -race，见 D-9）
	$(MAKE) test-race

test-race: ## 带竞态检测跑全部测试
	$(GO) test -race ./...

vet: ## go vet
	$(GO) vet ./...

lint: ## golangci-lint（本地未安装时提示安装命令）
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint 未安装；回退 go vet（CI 中以固定版本运行 lint）"; $(GO) vet ./...; \
	fi

cover: ## 生成覆盖率报告并检查阈值
	$(GO) test -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GO) tool cover -func=coverage.out | tail -1
	@total=$$($(GO) tool cover -func=coverage.out | tail -1 | awk '{print substr($$3, 1, length($$3)-1)}'); \
	awk -v t=$$total -v th=$(COVER_THRESHOLD) 'BEGIN{ if (t+0 < th+0) { printf "覆盖率 %.1f%% 低于阈值 %d%%\n", t, th; exit 1 } else { printf "覆盖率 %.1f%% ≥ 阈值 %d%%\n", t, th } }'

fmt: ## gofumpt 格式化（回退 gofmt）
	@if command -v gofumpt >/dev/null 2>&1; then gofumpt -w -extra .; else $(GO) fmt ./...; fi

fmt-check: ## 校验格式化零 diff（CI 使用）
	@if command -v gofumpt >/dev/null 2>&1; then \
		test -z "$$(gofumpt -l -extra .)" || { echo "以下文件未通过 gofumpt："; gofumpt -l -extra .; exit 1; }; \
	else \
		test -z "$$($(GO) fmt ./...)" || { echo "存在未格式化文件"; exit 1; }; \
	fi

tidy: ## go mod tidy
	$(GO) mod tidy

verify: build vet fmt-check test-race ## 本地全量门禁（与 CI 等价子集）

acceptance: ## 跑 GoldenSet 验收（runner + 门槛判定 + 矩阵套件 + 审计）
	$(GO) run ./acceptance/cmd/runner -cases acceptance/goldenset/cases -out acceptance/report
	$(GO) test -race -count=1 -timeout 600s ./acceptance/...

goldenset-freeze: ## 重新冻结 GoldenSet 内容 hash（P0 变更需重跑并评审）
	$(GO) run ./acceptance/cmd/freeze

audit-deps: ## 依赖漏洞与 license 审计（govulncheck 未装时提示）
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./... ; \
	else \
		echo "govulncheck 未安装：go install golang.org/x/vuln/cmd/govulncheck@latest；license 清单见 docs/dependency-license-baseline.md"; \
	fi

clean: ## 清理构建产物
	rm -f coverage.out
	$(GO) clean -testcache
