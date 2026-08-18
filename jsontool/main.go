// jsontool 是 Excel 配置表转 JSON 数据文件的生成工具，不依赖 .proto / .pb.go 编译链。
package main

import (
	"os"

	"github.com/hechh/tools/jsontool/cmd"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "jsontool",
	Short: "Excel配置表 → JSON 数据文件生成工具",
	Long: "jsontool 是 Excel 配置表转 JSON 数据文件的一站式生成工具，" +
		"不依赖 .proto / .pb.go 编译链，JSON 结构与 protobuf message 同构，可倒回 pb 使用。",
}

func init() {
	rootCmd.AddCommand(cmd.DataCmd)
	rootCmd.AddCommand(cmd.ProtoCmd)
	rootCmd.AddCommand(cmd.CodeCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
