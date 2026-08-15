.PHONY: build run test clean deps

# 构建项目
build:
	go build -o bin/e-sp-line2 main.go

# 运行项目
run:
	go run main.go

# 运行测试
test:
	go test -v ./...

# 清理构建文件
clean:
	rm -rf bin/
	rm -rf data/*.db

# 安装依赖
deps:
	go mod download
	go mod tidy

# 格式化代码
fmt:
	go fmt ./...

# 代码检查
lint:
	golangci-lint run

# 生成 API 文档
docs:
	swag init

# 数据库迁移
migrate:
	go run main.go migrate

# 开发模式运行
dev:
	go run main.go --dev
