package main

import (
	"os"

	"github.com/hechh/tools/xlsxtool/cmd"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "xlsxtool",
	Short: "Excel配置表 → Proto / 数据 / 查询代码一站式生成工具",
	Long: "xlsxtool 是 Excel 配置表转 Proto 定义、.conf 数据文件、" +
		"以及 Go 查询代码的一站式生成工具。",
}

func init() {
	rootCmd.AddCommand(cmd.ProtoCmd)
	rootCmd.AddCommand(cmd.DataCmd)
	rootCmd.AddCommand(cmd.CodeCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
