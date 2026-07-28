# hechh 项目

## 项目结构

```
src/
├── framework/   # 主框架（go.work 工作区）
├── library/     # 公共库（开源，被 framework 依赖）
└── tools/       # 辅助工具
```

`framework` 通过 `go.work` 引用本地 `library`，无需 `replace` 指令。

## 环境配置

`library`（`github.com/hechh/library`）是公开仓库，但新版本发布后 Go 代理和 sum 数据库需要同步时间。在此期间 `go mod tidy` 可能报错：

```
verifying module: github.com/hechh/library@v1.0.0:
reading https://goproxy.cn/sumdb/sum.golang.org/lookup/... : 404 Not Found
```

只需跳过 sumdb 校验即可（代理的 `direct` 回退能正常拉取代码）：

```bash
go env -w GONOSUMDB=github.com/hechh/library
```

> 不推荐设 `GOPRIVATE`——它是为私有仓库设计的，公开仓库用 `GONOSUMDB` 就够了。

### 为什么公开仓库也会报错？

Go 模块解析流程：

```
go mod tidy
  → GOPROXY（如 goproxy.cn）查找模块
    → 缓存未命中（新 tag 还没同步）
      → sum.golang.org 查校验和
        → 同样没有新版本的记录
          → 404 报错
```

这是代理同步延迟导致的，与仓库公开/私有无关。等几分钟到几小时后代理同步完成，不设 `GONOSUMDB` 也能正常通过。

## 开发

```bash
# 构建
cd framework && go build ./...

# 整理依赖
cd framework && go mod tidy

# 同步工作区
cd framework && go work sync
```
