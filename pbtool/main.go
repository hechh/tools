package main

import (
	"path/filepath"

	"github.com/hechh/library/base/fileutil"
	"github.com/hechh/tools/pbtool/internal"
	"github.com/spf13/cobra"
)

func main() {
	var src string
	cmd := &cobra.Command{
		Use:   "pbtool",
		Short: ".pb.go 辅助方法生成器",
		Long: "扫描 .pb.go 文件，自动为 protobuf message 生成 SetRspHead/GetRspHead、" +
			"ToDB/FromDB、ConfigS 只读包装等辅助方法。",
		RunE: func(cmd *cobra.Command, args []string) error {
			files, err := fileutil.Glob(filepath.Join(src, "*.pb.go"), true)
			if err != nil {
				return err
			}

			parser := &internal.Parser{}
			if err := fileutil.ParseFiles(parser, files...); err != nil {
				return err
			}
			return parser.Gen(src)
		},
	}

	cmd.Flags().StringVarP(&src, "src", "s", "", ".pb.go文件目录")
	cmd.MarkFlagRequired("src")

	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
