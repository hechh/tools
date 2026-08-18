// Package cmd 定义 xlsxtool 的全部子命令（proto / data / code）。
// 参考 astock 项目的 cobra 组织方式：包级 var 定义子命令，init() 中设置 flags。
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hechh/library/base/fileutil"
	"github.com/hechh/tools/xlsxtool/app"
	"github.com/hechh/tools/xlsxtool/domain"
	"github.com/hechh/tools/xlsxtool/infra"
	"github.com/spf13/cobra"
)

// 共享 flag 变量（各命令按需绑定）
var (
	xlsxDir      string   // xlsx 文件或目录
	protoDir     string   // proto 协议目录
	dstDir       string   // 输出目录
	extraImports []string // 额外的 proto import 路径
)

var (
	// ProtoCmd 从 xlsx 生成 proto 定义文件
	ProtoCmd = &cobra.Command{
		Use:   "proto",
		Short: "从xlsx生成proto定义文件",
		RunE:  runProto,
	}

	// DataCmd 从 xlsx 生成 protobuf 文本数据文件(.conf)
	DataCmd = &cobra.Command{
		Use:   "data",
		Short: "从xlsx生成protobuf文本数据文件(.conf)",
		RunE:  runData,
	}

	// CodeCmd 从 xlsx 生成 Go 查询代码(.gen.go)
	CodeCmd = &cobra.Command{
		Use:   "code",
		Short: "从xlsx生成Go查询代码(.gen.go)",
		RunE:  runCode,
	}
)

func init() {
	// proto 命令
	ProtoCmd.Flags().StringVarP(&xlsxDir, "xlsx", "x", "", "xlsx文件或目录")
	ProtoCmd.Flags().StringVarP(&dstDir, "output", "o", "", "proto文件输出目录")
	ProtoCmd.MarkFlagRequired("xlsx")
	ProtoCmd.MarkFlagRequired("output")

	// data 命令
	DataCmd.Flags().StringVarP(&xlsxDir, "xlsx", "x", ".", "xlsx配置目录")
	DataCmd.Flags().StringVarP(&protoDir, "proto", "p", ".", "proto协议目录")
	DataCmd.Flags().StringVarP(&dstDir, "output", "o", ".", "数据输出目录")
	DataCmd.Flags().StringSliceVarP(&extraImports, "import", "i", nil, "额外的proto import路径(可多次指定)")

	// code 命令
	CodeCmd.Flags().StringVarP(&xlsxDir, "xlsx", "x", ".", "xlsx配置目录")
	CodeCmd.Flags().StringVarP(&protoDir, "proto", "p", ".", "proto协议目录")
	CodeCmd.Flags().StringVarP(&dstDir, "output", "o", ".", "数据输出目录")
	CodeCmd.Flags().StringSliceVarP(&extraImports, "import", "i", nil, "额外的proto import路径(可多次指定)")
}

// runProto 生成 proto 定义文件：创建输出目录 → 扫描已有 proto → 读 xlsx → 解析 → 生成。
func runProto(cmd *cobra.Command, args []string) error {
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	// 1. 扫描已有 proto 文件
	registry := domain.NewProtoRegistry()
	if err := infra.ScanProtoDir(dstDir, registry); err != nil {
		return fmt.Errorf("扫描proto失败: %w", err)
	}

	// 2. 读取 xlsx
	fmt.Printf("[OK] 扫描xlsx: %s\n", xlsxDir)
	tables := infra.ReadTables(xlsxDir)

	// 3. 解析
	if len(tables) > 0 {
		ctx := domain.NewParseContext(registry)
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

// runData 生成 .conf 数据文件：加载并编译 proto → 读 xlsx → 解析 → 生成。
func runData(cmd *cobra.Command, args []string) error {
	// 1. 加载 proto
	fmt.Printf("[OK] 加载proto: %s\n", protoDir)
	registry := domain.NewProtoRegistry()
	if err := infra.ScanProtoDir(protoDir, registry); err != nil {
		return fmt.Errorf("扫描proto失败: %w", err)
	}
	if err := infra.CompileProtoDir(protoDir, registry, extraImports...); err != nil {
		return fmt.Errorf("编译proto失败: %w", err)
	}

	// 2. 读取 xlsx
	fmt.Printf("[OK] 扫描xlsx: %s\n", xlsxDir)
	tables := infra.ReadTables(xlsxDir)

	// 3. 解析
	if len(tables) > 0 {
		ctx := domain.NewParseContext(registry)
		for _, t := range tables {
			ctx.ParseTable(t)
		}

		// 4. 生成
		if err := os.MkdirAll(dstDir, 0o755); err != nil {
			return fmt.Errorf("创建输出目录失败: %w", err)
		}
		if err := app.GenData(ctx, dstDir, saveFile); err != nil {
			return fmt.Errorf("生成数据失败: %w", err)
		}
		fmt.Printf("[OK] 完成! %s\n", dstDir)
	} else {
		fmt.Printf("[OK] 没有合法配置表需要处理!\n")
	}
	return nil
}

// runCode 生成 Go 查询代码(.gen.go)：加载并编译 proto → 读 xlsx → 解析 → 生成。
func runCode(cmd *cobra.Command, args []string) error {
	// 1. 加载 proto
	fmt.Printf("[OK] 加载proto: %s\n", protoDir)
	registry := domain.NewProtoRegistry()
	if err := infra.ScanProtoDir(protoDir, registry); err != nil {
		return fmt.Errorf("扫描proto失败: %w", err)
	}
	if err := infra.CompileProtoDir(protoDir, registry, extraImports...); err != nil {
		return fmt.Errorf("编译proto失败: %w", err)
	}

	// 2. 读取 xlsx
	fmt.Printf("[OK] 扫描xlsx: %s\n", xlsxDir)
	tables := infra.ReadTables(xlsxDir)

	// 3. 解析
	if len(tables) > 0 {
		ctx := domain.NewParseContext(registry)
		for _, t := range tables {
			ctx.ParseTable(t)
		}

		// 4. 生成
		if err := os.MkdirAll(dstDir, 0o755); err != nil {
			return fmt.Errorf("创建输出目录失败: %w", err)
		}
		if err := app.GenCode(ctx, dstDir, fileutil.Save); err != nil {
			return fmt.Errorf("生成代码失败: %w", err)
		}
		fmt.Printf("[OK] 完成! %s\n", dstDir)
	} else {
		fmt.Printf("[OK] 没有合法配置表需要处理!\n")
	}
	return nil
}

// saveFile 直接写文件（权限 0o644）
func saveFile(dir, filename string, data []byte) error {
	return os.WriteFile(filepath.Join(dir, filename), data, 0o644)
}
