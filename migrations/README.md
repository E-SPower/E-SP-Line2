# 数据库迁移脚本

## 概述

本目录包含 E-SP-Line2 项目的数据库迁移脚本。

## 使用方法

### 自动迁移

系统启动时会自动执行迁移（如果配置了自动迁移）。

### 手动迁移

```bash
# 执行所有迁移
go run main.go migrate

# 执行特定版本迁移
go run main.go migrate --version 001

# 回滚到上一个版本
go run main.go migrate rollback
```

## 迁移文件命名规范

```
XXX_description.sql
```

- `XXX`: 三位数字版本号（如 001, 002）
- `description`: 迁移描述（使用下划线分隔）

## 当前迁移列表

| 版本 | 文件 | 描述 |
|------|------|------|
| 001 | 001_initial_schema.sql | 初始数据库架构 |

## 编写迁移脚本

### 基本结构

```sql
-- E-SP-Line2 Database Migration
-- Version: XXX
-- Description: Your description

-- Your SQL here
```

### 注意事项

1. 使用 `IF NOT EXISTS` 避免重复创建
2. 使用 `ON CONFLICT` 处理冲突
3. 添加必要的索引
4. 提供回滚脚本（可选）

## 数据库支持

- SQLite（开发环境）
- PostgreSQL（生产环境）

## 版本历史

- v001 (2024-01-01): 初始架构
