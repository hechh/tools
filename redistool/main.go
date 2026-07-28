package main

import (
	"path/filepath"

	"github.com/hechh/library/base/fileutil"
	"github.com/hechh/tools/redistool/app"
	"github.com/hechh/tools/redistool/domain"
	"github.com/hechh/tools/redistool/infra"
	"github.com/spf13/cobra"
)

func main() {
	var src, dst string

	cmd := &cobra.Command{
		Use:   "redistool",
		Short: "Redis 数据访问层代码生成器",
		Long: "扫描 .pb.go 中 @dbtool 注解，自动生成 Redis String/Hash 的 CRUD 函数" +
			"及可选的 Cache 层。核心理念：proto 中声明 → 工具生成 → 业务直接调用。",
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. 创建领域上下文（聚合根）
			ctx := &domain.ParseContext{}

			// 2. 创建AST解析器（基础设施层）
			astParser := infra.NewASTParser(ctx)

			// 3. 扫描并解析所有.pb.go文件
			files, err := fileutil.Glob(filepath.Join(src, "*.pb.go"), true)
			if err != nil {
				return err
			}
			if err := fileutil.ParseFiles(astParser, files...); err != nil {
				return err
			}

			// 4. 构建模板（基础设施层）
			strTpl, err := infra.BuildStringTemplate()
			if err != nil {
				return err
			}
			hashTpl, err := infra.BuildHashTemplate()
			if err != nil {
				return err
			}

			// 5. 创建领域服务+应用服务（DI组装）
			generator := domain.NewGenerator(strTpl, hashTpl)
			svc := app.NewService(generator)

			// 6. 执行生成，输出到文件系统
			writer := infra.NewFileWriter(dst)
			return svc.Run(ctx, writer)
		},
	}

	cmd.Flags().StringVarP(&src, "src", "s", "", ".pb.go源码文件目录")
	cmd.Flags().StringVarP(&dst, "dst", "d", "", "生成代码输出目录")
	cmd.MarkFlagRequired("src")
	cmd.MarkFlagRequired("dst")

	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
