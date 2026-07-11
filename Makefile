# md2wechat Makefile
# 适用于开发者和高级用户

VERSION ?= $(shell tr -d '[:space:]' < VERSION)
LDFLAGS := -s -w -X main.Version=$(VERSION)

.PHONY: all build clean test install help lint fmt vet release release-check deps quality-gates e2e-layout

# 默认目标
all: build

# 构建所有平台的二进制文件（发布到 bin/ 目录）
release:
	@echo "🔨 构建 md2wechat 所有平台版本..."
	@echo "Version: $(VERSION)"
	@echo ""
	@echo "📦 Building for Linux amd64..."
	@GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="$(LDFLAGS)" -o bin/md2wechat-linux-amd64 ./cmd/md2wechat
	@echo "✓ Linux amd64"
	@echo "📦 Building for Linux arm64..."
	@GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="$(LDFLAGS)" -o bin/md2wechat-linux-arm64 ./cmd/md2wechat
	@echo "✓ Linux arm64"
	@echo "📦 Building for macOS amd64 (Intel)..."
	@GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="$(LDFLAGS)" -o bin/md2wechat-darwin-amd64 ./cmd/md2wechat
	@echo "✓ macOS amd64"
	@echo "📦 Building for macOS arm64 (Apple Silicon)..."
	@GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="$(LDFLAGS)" -o bin/md2wechat-darwin-arm64 ./cmd/md2wechat
	@echo "✓ macOS arm64"
	@echo "📦 Building for Windows amd64..."
	@GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="$(LDFLAGS)" -o bin/md2wechat-windows-amd64.exe ./cmd/md2wechat
	@echo "✓ Windows amd64"
	@echo ""
	@chmod +x bin/*-linux* bin/*-darwin* 2>/dev/null || true
	@echo "✅ 构建完成！二进制文件在 bin/ 目录"
	@echo ""
	@ls -lh bin/

# 构建当前平台
build:
	@echo "🔨 构建当前平台..."
	@echo "Version: $(VERSION)"
	@go build -trimpath -ldflags="$(LDFLAGS)" -o md2wechat ./cmd/md2wechat
	@echo "✅ 构建完成: ./md2wechat"

# 快速构建（仅当前平台，用于开发）
fast:
	@go build -trimpath -ldflags="$(LDFLAGS)" -o md2wechat ./cmd/md2wechat

# 清理
clean:
	@echo "🧹 清理..."
	@rm -f md2wechat
	@rm -rf dist/ release/
	@rm -f *.log

# 运行测试
test:
	@echo "🧪 运行测试..."
	@CGO_ENABLED=1 go test -count=1 ./...

# 代码检查
lint:
	@echo "🔍 代码检查..."
	@bash scripts/run-golangci-lint.sh

# 格式化代码
fmt:
	@echo "🎨 格式化代码..."
	@go fmt ./...
	@gofmt -w .

# 静态分析
vet:
	@echo "🔬 静态分析..."
	@go vet ./...

# 发布前一致性检查
release-check:
	@echo "🔍 检查发布一致性..."
	@bash scripts/release-check.sh

# 本地/CI 统一质量门
quality-gates:
	@bash scripts/quality-gates.sh

# E2E: layout module catalog vs production /api/convert consistency (opt-in)
e2e-layout:
	@./scripts/layout-conformance.sh

# 安装到 GOPATH/bin
install:
	@echo "📦 安装到 $(GOPATH)/bin..."
	@go install ./cmd/md2wechat

# 下载依赖
deps:
	@echo "📥 下载依赖..."
	@go mod download
	@go mod tidy

# 帮助
help:
	@echo "md2wechat Makefile 命令:"
	@echo ""
	@echo "开发命令:"
	@echo "  make build       - 构建当前平台二进制"
	@echo "  make fast        - 快速构建（开发用）"
	@echo "  make release     - 构建所有平台二进制到 bin/"
	@echo "  make clean       - 清理构建文件"
	@echo ""
	@echo "代码质量:"
	@echo "  make fmt         - 格式化代码"
	@echo "  make vet         - 静态分析"
	@echo "  make test        - 运行测试"
	@echo "  make lint        - 运行与 CI 一致的 golangci-lint"
	@echo "  make quality-gates - 运行与 CI 一致的完整发布前检查"
	@echo "  make release-check - 检查版本/文档/workflow 一致性"
	@echo ""
	@echo "依赖管理:"
	@echo "  make deps        - 下载依赖"
	@echo "  make install     - 安装到 GOPATH/bin"
	@echo ""
	@echo "用户快速安装:"
	@echo "  go install github.com/geekjourneyx/md2wechat-skill/cmd/md2wechat@latest"
