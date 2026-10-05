.PHONY: help test test-verbose test-spec test-all tidy clean

# 默认目标：显示帮助
help:
	@echo "可用命令："
	@echo "  make test        - 运行所有测试 (go test ./...)"
	@echo "  make test-verbose - 运行所有测试并输出详细日志 (go test -v ./...)"
	@echo "  make test-spec   - 运行 TestPrintAPISpec 并输出详细日志"
	@echo "  make test-all    - 依次运行 test 和 test-spec"
	@echo "  make tidy        - 整理依赖 (go mod tidy)"
	@echo "  make clean       - 清理测试缓存 (go clean -testcache)"

# 运行所有测试
test:
	go test ./...

# 运行所有测试并输出详细日志
test-verbose:
	go test -v ./...

# 打印 ApiSpec 结构
test-spec:
	go test -run TestPrintAPISpec -v ./...

# 一次性跑完所有测试
test-all: test test-spec

# 整理依赖
tidy:
	go mod tidy

# 清理测试缓存
clean:
	go clean -testcache
