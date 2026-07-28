# hechh 项目

## 项目结构

```
src/
├── framework/   # 主框架（go.work 工作区）
├── library/     # 公共库（被 framework 依赖）
└── tools/       # 辅助工具
```

`framework` 通过 `go.work` 引用本地 `library`，无需 `replace` 指令。

## 环境配置

### 前置条件

`framework` 依赖 `library` 模块（`github.com/hechh/library`）。该模块为私有仓库，需要在 Go 中配置跳过公共代理和校验：

```bash
go env -w GOPRIVATE=github.com/hechh/*
```

> `GOPRIVATE` 相当于同时启用 `GONOPROXY` + `GONOSUMDB`，只需设这一个即可。

### 为什么需要配置 GOPRIVATE？

`go mod tidy` 会解析完整依赖图并验证所有模块。如果没有配置 `GOPRIVATE`，Go 会尝试从公共代理（如 `goproxy.cn`）下载私有模块，导致以下错误：

```
go: github.com/hechh/library@v1.0.0: verifying module: ... 404 Not Found
```

设置 `GOPRIVATE=github.com/hechh/*` 后，Go 会直连源仓库（git），不再走公共代理和 sum 数据库校验。

### 各环境变量说明

| 变量 | 作用 |
|------|------|
| `GOPRIVATE` | 总开关，同时启用 `GONOPROXY` + `GONOSUMDB` 的效果 |
| `GONOPROXY` | 跳过模块代理，直接从源仓库（git）拉取 |
| `GONOSUMDB` | 跳过公共 sum 数据库校验（sum.golang.org） |

## 开发

```bash
# 构建
cd framework && go build ./...

# 整理依赖
cd framework && go mod tidy

# 同步工作区
cd framework && go work sync
```
