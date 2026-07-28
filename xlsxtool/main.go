package main

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

func main() {
	root := &cobra.Command{
		Use:   "xlsxtool",
		Short: "Excel配置表 → Proto / 数据 / 查询代码一站式生成工具",
		Long: "xlsxtool 是 Excel 配置表转 Proto 定义、.conf 数据文件、" +
			"以及 Go 查询代码的一站式生成工具。",
	}

	root.AddCommand(newProtoCmd())
	root.AddCommand(newDataCmd())
	root.AddCommand(newCodeCmd())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func newProtoCmd() *cobra.Command {
	var input, output string

	cmd := &cobra.Command{
		Use:   "proto",
		Short: "从xlsx生成proto定义文件",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := os.MkdirAll(output, 0o755); err != nil {
				return fmt.Errorf("创建输出目录失败: %w", err)
			}

			// 1. 扫描已有proto文件
			registry := domain.NewProtoRegistry()
			if err := infra.ScanProtoDir(output, registry); err != nil {
				return fmt.Errorf("扫描proto失败: %w", err)
			}

			// 2. 读取xlsx
			fmt.Printf("[OK] 扫描xlsx: %s\n", input)
			tables := infra.ReadTables(input)

			// 3. 解析
			if len(tables) > 0 {
				ctx := domain.NewParseContext(registry)
				for _, t := range tables {
					ctx.ParseTable(t)
				}

				// 4. 生成
				if err := app.GenProto(ctx, output, saveFile); err != nil {
					return fmt.Errorf("生成proto失败: %w", err)
				}
				fmt.Printf("[OK] 完成! %s\n", output)
			} else {
				fmt.Printf("[OK] 没有合法配置表需要处理!\n")
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&input, "xlsx", "x", "", "xlsx文件或目录")
	cmd.Flags().StringVarP(&output, "output", "o", "", "proto文件输出目录")
	cmd.MarkFlagRequired("xlsx")
	cmd.MarkFlagRequired("output")

	return cmd
}

func newDataCmd() *cobra.Command {
	var xlsxDir, protoDir, dstDir string
	var extraImports []string

	cmd := &cobra.Command{
		Use:   "data",
		Short: "从xlsx生成protobuf文本数据文件(.conf)",
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. 加载proto
			fmt.Printf("[OK] 加载proto: %s\n", protoDir)
			registry := domain.NewProtoRegistry()
			if err := infra.ScanProtoDir(protoDir, registry); err != nil {
				return fmt.Errorf("扫描proto失败: %w", err)
			}
			if err := infra.CompileProtoDir(protoDir, registry, extraImports...); err != nil {
				return fmt.Errorf("编译proto失败: %w", err)
			}

			// 2. 读取xlsx
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
		},
	}

	cmd.Flags().StringVarP(&xlsxDir, "xlsx", "x", ".", "xlsx配置目录")
	cmd.Flags().StringVarP(&protoDir, "proto", "p", ".", "proto协议目录")
	cmd.Flags().StringVarP(&dstDir, "output", "o", ".", "数据输出目录")
	cmd.Flags().StringSliceVarP(&extraImports, "import", "i", nil, "额外的proto import路径(可多次指定)")

	return cmd
}

func newCodeCmd() *cobra.Command {
	var xlsxDir, protoDir, dstDir string
	var extraImports []string

	cmd := &cobra.Command{
		Use:   "code",
		Short: "从xlsx生成Go查询代码(.gen.go)",
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. 加载proto
			fmt.Printf("[OK] 加载proto: %s\n", protoDir)
			registry := domain.NewProtoRegistry()
			if err := infra.ScanProtoDir(protoDir, registry); err != nil {
				return fmt.Errorf("扫描proto失败: %w", err)
			}
			if err := infra.CompileProtoDir(protoDir, registry, extraImports...); err != nil {
				return fmt.Errorf("编译proto失败: %w", err)
			}

			// 2. 读取xlsx
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
		},
	}

	cmd.Flags().StringVarP(&xlsxDir, "xlsx", "x", ".", "xlsx配置目录")
	cmd.Flags().StringVarP(&protoDir, "proto", "p", ".", "proto协议目录")
	cmd.Flags().StringVarP(&dstDir, "output", "o", ".", "数据输出目录")
	cmd.Flags().StringSliceVarP(&extraImports, "import", "i", nil, "额外的proto import路径(可多次指定)")

	return cmd
}

func saveFile(dir, filename string, data []byte) error {
	return os.WriteFile(filepath.Join(dir, filename), data, 0o644)
}
