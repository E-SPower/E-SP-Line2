# 贡献指南

## 欢迎贡献

感谢您对 E-SP-Line2 项目的关注！我们欢迎各种形式的贡献，包括代码、文档、问题报告和功能建议。

## 贡献流程

### 1. 发现问题或想要添加功能

- 检查 [Issues](https://github.com/your-org/E-SP-Line2/issues) 是否已有相关讨论
- 如果没有，创建新的 Issue 描述问题或功能

### 2. 开始开发

- Fork 项目到您的 GitHub 账号
- 克隆到本地：
  ```bash
  git clone https://github.com/your-username/E-SP-Line2.git
  cd E-SP-Line2
  ```
- 创建功能分支：
  ```bash
  git checkout -b feature/your-feature-name
  ```

### 3. 开发规范

#### 代码风格

- **Go 代码**：遵循 [Effective Go](https://go.dev/doc/effective-go) 和 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- **Python 代码**：遵循 [PEP 8](https://peps.python.org/pep-0008/)
- **TypeScript/React**：遵循项目 ESLint 配置

#### 提交信息

使用约定式提交（Conventional Commits）：

```
<type>(<scope>): <subject>

<body>

<footer>
```

**类型**：
- `feat`: 新功能
- `fix`: 修复 bug
- `docs`: 文档更新
- `style`: 代码格式（不影响代码运行）
- `refactor`: 代码重构
- `test`: 添加测试
- `chore`: 构建过程或辅助工具变动

**示例**：
```
feat(adapter): add support for Pinduoduo platform

- Implement Pinduoduo message adapter
- Add product card message type
- Add unit tests

Closes #123
```

### 4. 提交 Pull Request

- 确保代码通过所有测试
- 更新相关文档
- 创建 Pull Request 并描述更改

## 开发环境设置

### 后端开发

```bash
# 安装 Go 依赖
cd backend
go mod download

# 运行测试
go test ./...

# 启动开发服务器
go run main.go
```

### 前端开发

```bash
# 安装依赖
cd web
npm install

# 启动开发服务器
npm run dev

# 构建生产版本
npm run build
```

### Python 适配器开发

```bash
# 创建虚拟环境
cd adapters/myplatform
python -m venv venv
source venv/bin/activate

# 安装依赖
pip install -r requirements.txt

# 运行测试
pytest
```

## 代码审查

所有 Pull Request 都需要经过代码审查。审查者会关注：

- 代码质量和风格
- 测试覆盖率
- 文档完整性
- 性能影响
- 安全性

## 报告 Bug

### Bug 报告模板

```markdown
**描述**
清晰简洁地描述 bug。

**复现步骤**
1. 执行 '...'
2. 点击 '...'
3. 看到错误

**期望行为**
描述期望发生什么。

**实际行为**
描述实际发生了什么。

**环境**
- OS: [e.g. Ubuntu 22.04]
- Go version: [e.g. 1.22]
- Node version: [e.g. 18.x]

**日志**
```
粘贴相关日志
```

**截图**
如果适用，添加截图。
```

## 功能建议

### 功能建议模板

```markdown
**问题描述**
清晰简洁地描述问题或需求。

**建议的解决方案**
描述您建议的解决方案。

**替代方案**
描述您考虑过的替代方案。

**其他信息**
添加任何其他相关信息。
```

## 文档贡献

文档与代码同样重要。贡献文档时：

- 使用清晰的 Markdown 格式
- 提供代码示例
- 保持语言简洁
- 检查拼写和语法

## 测试

### 编写测试

- 为新功能编写单元测试
- 为 bug 修复编写回归测试
- 保持测试覆盖率 > 80%

### 运行测试

```bash
# 后端测试
cd backend
go test ./... -v

# 前端测试
cd web
npm test

# Python 适配器测试
cd adapters/myplatform
pytest
```

## 发布流程

### 版本号

遵循语义化版本（Semantic Versioning）：

- `MAJOR.MINOR.PATCH`
- `MAJOR`: 不兼容的 API 更改
- `MINOR`: 向后兼容的功能添加
- `PATCH`: 向后兼容的 bug 修复

### 发布检查清单

- [ ] 更新版本号
- [ ] 更新 CHANGELOG
- [ ] 运行所有测试
- [ ] 构建生产版本
- [ ] 创建 Git tag
- [ ] 发布到包管理器

## 行为准则

### 我们的承诺

为了营造一个开放和友好的环境，我们承诺：

- 使用友好和包容的语言
- 尊重不同的观点和经验
- 优雅地接受建设性批评
- 关注对社区最有利的事情
- 对其他社区成员表示同理心

### 不可接受的行为

- 使用性化的语言或图像
- 人身攻击或侮辱性评论
- 公开或私下骚扰
- 未经许可发布他人信息
- 其他不道德或不专业的行为

## 获取帮助

如果您在贡献过程中需要帮助：

- 查看 [文档](./)
- 在 Issue 中提问
- 联系项目维护者

## 致谢

感谢所有贡献者帮助改进 E-SP-Line2 项目！
