package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	futil "github.com/hechh/library/base/fileutil"
	"github.com/hechh/tools/xlsxtool/app"
	"github.com/hechh/tools/xlsxtool/domain"
	"github.com/hechh/tools/xlsxtool/infra"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	switch os.Args[1] {
	case "proto":
		runProto()
	case "data":
		runData()
	case "code":
		runCode()
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func runProto() {
	var input, output string
	cmd := flag.NewFlagSet("proto", flag.ExitOnError)
	cmd.StringVar(&input, "x", "", "xlsx文件或目录")
	cmd.StringVar(&output, "o", "", "proto文件输出目录")
	cmd.Parse(os.Args[2:])

	if input == "" || output == "" {
		cmd.Usage()
		os.Exit(1)
	}
	if err := os.MkdirAll(output, 0o755); err != nil {
		exit("创建输出目录失败: %v", err)
	}

	// 1. 扫描已有proto文件
	registry := domain.NewProtoRegistry()
	if err := infra.ScanProtoDir(output, registry); err != nil {
		exit("扫描proto失败: %v", err)
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
			exit("生成proto失败: %v", err)
		}
		fmt.Printf("[OK] 完成! %s\n", output)
	} else {
		fmt.Printf("[OK] 没有合法配置表需要处理!\n")
	}
}

func runData() {
	var xlsxDir, protoDir, dstDir string
	var extraImports arrayFlags
	cmd := flag.NewFlagSet("data", flag.ExitOnError)
	cmd.StringVar(&xlsxDir, "x", ".", "xlsx配置目录")
	cmd.StringVar(&protoDir, "p", ".", "proto协议目录")
	cmd.StringVar(&dstDir, "o", ".", "数据输出目录")
	cmd.Var(&extraImports, "i", "额外的proto import路径(可多次指定)")
	cmd.Parse(os.Args[2:])

	// 1. 加载proto
	fmt.Printf("[OK] 加载proto: %s\n", protoDir)
	registry := domain.NewProtoRegistry()
	if err := infra.ScanProtoDir(protoDir, registry); err != nil {
		exit("扫描proto失败: %v", err)
	}
	if err := infra.CompileProtoDir(protoDir, registry, extraImports...); err != nil {
		exit("编译proto失败: %v", err)
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
			exit("创建输出目录失败: %v", err)
		}
		if err := app.GenData(ctx, dstDir, saveFile); err != nil {
			exit("生成数据失败: %v", err)
		}
		fmt.Printf("[OK] 完成! %s\n", dstDir)
	} else {
		fmt.Printf("[OK] 没有合法配置表需要处理!\n")
	}
}

func runCode() {
	var xlsxDir, protoDir, dstDir string
	var extraImports arrayFlags
	cmd := flag.NewFlagSet("code", flag.ExitOnError)
	cmd.StringVar(&xlsxDir, "x", ".", "xlsx配置目录")
	cmd.StringVar(&protoDir, "p", ".", "proto协议目录")
	cmd.StringVar(&dstDir, "o", ".", "数据输出目录")
	cmd.Var(&extraImports, "i", "额外的proto import路径(可多次指定)")
	cmd.Parse(os.Args[2:])

	// 1. 加载proto
	fmt.Printf("[OK] 加载proto: %s\n", protoDir)
	registry := domain.NewProtoRegistry()
	if err := infra.ScanProtoDir(protoDir, registry); err != nil {
		exit("扫描proto失败: %v", err)
	}
	if err := infra.CompileProtoDir(protoDir, registry, extraImports...); err != nil {
		exit("编译proto失败: %v", err)
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
			exit("创建输出目录失败: %v", err)
		}
		if err := app.GenCode(ctx, dstDir, futil.Save); err != nil {
			exit("生成代码失败: %v", err)
		}
		fmt.Printf("[OK] 完成! %s\n", dstDir)
	} else {
		fmt.Printf("[OK] 没有合法配置表需要处理!\n")
	}
}

type arrayFlags []string

func (a *arrayFlags) String() string     { return strings.Join(*a, " ") }
func (a *arrayFlags) Set(v string) error { *a = append(*a, v); return nil }

func printUsage() {
	fmt.Println("xlsxtool - Excel配置转Proto/数据文件工具")
	fmt.Println()
	fmt.Println("用法: xlsxtool <command> [选项]")
	fmt.Println()
	fmt.Println("命令:")
	fmt.Println("  proto    从xlsx生成proto定义文件")
	fmt.Println("  data     从xlsx生成protobuf文本数据文件")
	fmt.Println()
	fmt.Println("  proto: xlsxtool proto -i <xlsx> -o <output>")
	fmt.Println("  data:  xlsxtool data -xlsx <dir> -proto <dir> -dst <dir>")
}

func saveFile(dir, filename string, data []byte) error {
	return os.WriteFile(filepath.Join(dir, filename), data, 0o644)
}

func exit(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
