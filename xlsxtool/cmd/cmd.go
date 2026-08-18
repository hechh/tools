// Package cmd 定义 xlsxtool 的全部子命令（proto / data / code）。
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hechh/tools/xlsxtool/app"
	"github.com/hechh/tools/xlsxtool/domain"
	"github.com/hechh/tools/xlsxtool/infra"

	"github.com/spf13/cobra"
)

var (
	xlsxDir   string // xlsx 文件或目录
	dstDir    string // 输出目录
	pkgName   string // proto package（默认从已有 proto 扫描）
	goPkgName string // go_package 路径（默认从已有 proto 扫描）
)

// DataCmd 从 xlsx 生成 json 数据文件
var DataCmd = &cobra.Command{
	Use:   "data",
	Short: "从xlsx生成json数据文件",
	RunE:  runData,
}

// ProtoCmd 从 xlsx 生成 proto 定义文件
var ProtoCmd = &cobra.Command{
	Use:   "proto",
	Short: "从xlsx生成proto定义文件",
	RunE:  runProto,
}

// CodeCmd 从 xlsx 生成 Go 查询代码(.gen.go)
var CodeCmd = &cobra.Command{
	Use:   "code",
	Short: "从xlsx生成Go查询代码(.gen.go)",
	RunE:  runCode,
}

func init() {
	// data 命令
	DataCmd.Flags().StringVarP(&xlsxDir, "xlsx", "x", ".", "xlsx配置目录")
	DataCmd.Flags().StringVarP(&dstDir, "output", "o", ".", "数据输出目录")
	DataCmd.MarkFlagRequired("xlsx")
	DataCmd.MarkFlagRequired("output")

	// proto 命令
	ProtoCmd.Flags().StringVarP(&xlsxDir, "xlsx", "x", "", "xlsx文件或目录")
	ProtoCmd.Flags().StringVarP(&dstDir, "output", "o", "", "proto文件输出目录")
	ProtoCmd.Flags().StringVarP(&pkgName, "pkg", "", "", "proto package名（输出目录无已有proto时必填）")
	ProtoCmd.Flags().StringVarP(&goPkgName, "gopkg", "", "", "go_package路径（输出目录无已有proto时必填）")
	ProtoCmd.MarkFlagRequired("xlsx")
	ProtoCmd.MarkFlagRequired("output")

	// code 命令
	CodeCmd.Flags().StringVarP(&xlsxDir, "xlsx", "x", ".", "xlsx配置目录")
	CodeCmd.Flags().StringVarP(&dstDir, "output", "o", ".", "代码输出目录")
	CodeCmd.MarkFlagRequired("xlsx")
	CodeCmd.MarkFlagRequired("output")
}

func saveFile(dir, filename string, data []byte) error {
	return os.WriteFile(filepath.Join(dir, filename), data, 0o644)
}

// runProto 生成 proto 定义文件：扫描已有 proto → 读 xlsx → 解析 → 生成。
func runProto(cmd *cobra.Command, args []string) error {
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	// 1. 扫描已有 proto 文件（获取 package 信息与外部类型）
	registry := domain.NewProtoRegistry()
	if err := infra.ScanProtoDir(dstDir, registry); err != nil {
		return fmt.Errorf("扫描proto失败: %w", err)
	}
	// 输出目录无已有 proto 时，用命令行参数兜底；仍缺失则报错，避免生成无效 proto
	if registry.Pkg == "" {
		registry.Pkg = pkgName
	}
	if registry.GoPkg == "" {
		registry.GoPkg = goPkgName
	}
	if registry.Pkg == "" || registry.GoPkg == "" {
		return fmt.Errorf("无法确定proto package信息：输出目录(%s)无已有proto，请指定 --pkg 与 --gopkg", dstDir)
	}

	// 2. 读取 xlsx
	fmt.Printf("[OK] 扫描xlsx: %s\n", xlsxDir)
	tables := infra.ReadTables(xlsxDir)

	// 3. 解析
	if len(tables) > 0 {
		ctx := domain.NewParseContext()
		ctx.Registry = registry
		for _, t := range tables {
			ctx.ParseTable(t)
		}

		// 4. 生成
		if err := app.GenProto(ctx, dstDir, saveFile); err != nil {
			return fmt.Errorf("生成proto失败: %w", err)
		}
		fmt.Printf("[OK] 完成! %s\n", dstDir)
	} else {
		fmt.Printf("[OK] 没有合法配置表需要处理!\n")
	}
	return nil
}

// runData 生成 json 数据文件：读取 xlsx → 解析 → 生成。
func runData(cmd *cobra.Command, args []string) error {
	// 1. 读取 xlsx
	fmt.Printf("[OK] 扫描xlsx: %s\n", xlsxDir)
	tables := infra.ReadTables(xlsxDir)

	// 2. 解析
	if len(tables) > 0 {
		ctx := domain.NewParseContext()
		for _, t := range tables {
			ctx.ParseTable(t)
		}

		// 3. 生成
		if err := os.MkdirAll(dstDir, 0o755); err != nil {
			return fmt.Errorf("创建输出目录失败: %w", err)
		}
		if err := app.GenJSON(ctx, dstDir, saveFile); err != nil {
			return fmt.Errorf("生成json失败: %w", err)
		}
		fmt.Printf("[OK] 完成! %s\n", dstDir)
	} else {
		fmt.Printf("[OK] 没有合法配置表需要处理!\n")
	}
	return nil
}

// runCode 生成 Go 查询代码：读取 xlsx → 解析 → 生成（无 proto 编译依赖，枚举由 @enum 表判断）。
func runCode(cmd *cobra.Command, args []string) error {
	// 1. 读取 xlsx
	fmt.Printf("[OK] 扫描xlsx: %s\n", xlsxDir)
	tables := infra.ReadTables(xlsxDir)

	// 2. 解析
	if len(tables) > 0 {
		ctx := domain.NewParseContext()
		for _, t := range tables {
			ctx.ParseTable(t)
		}

		// 3. 生成
		if err := os.MkdirAll(dstDir, 0o755); err != nil {
			return fmt.Errorf("创建输出目录失败: %w", err)
		}
		if err := app.GenCode(ctx, dstDir, saveFile); err != nil {
			return fmt.Errorf("生成代码失败: %w", err)
		}
		fmt.Printf("[OK] 完成! %s\n", dstDir)
	} else {
		fmt.Printf("[OK] 没有合法酏表需要处理!\n")
	}
	return nil
}
