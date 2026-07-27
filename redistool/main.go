package main

import (
	"flag"

	futil "github.com/hechh/library/base/fileutil"
	"github.com/hechh/tools/redistool/app"
	"github.com/hechh/tools/redistool/domain"
	"github.com/hechh/tools/redistool/infra"
)

func main() {
	var src, dst string
	flag.StringVar(&src, "src", "", ".pb.go源码文件目录")
	flag.StringVar(&dst, "dst", "", "生成代码输出目录")
	flag.Parse()

	// 1. 创建领域上下文（聚合根）
	ctx := &domain.ParseContext{}

	// 2. 创建AST解析器（基础设施层）
	astParser := infra.NewASTParser(ctx)

	// 3. 扫描并解析所有.pb.go文件
	files, err := futil.Glob(src, ".*\\.pb\\.go", true)
	if err != nil {
		panic("scan source files failed: " + err.Error())
	}

	if err := futil.ParseFiles(astParser, files...); err != nil {
		panic("parse proto files failed: " + err.Error())
	}

	// 4. 构建模板（基础设施层）
	strTpl, err := infra.BuildStringTemplate()
	if err != nil {
		panic("build string template failed: " + err.Error())
	}
	hashTpl, err := infra.BuildHashTemplate()
	if err != nil {
		panic("build hash template failed: " + err.Error())
	}

	// 5. 创建领域服务+应用服务（DI组装）
	generator := domain.NewGenerator(strTpl, hashTpl)
	svc := app.NewService(generator)

	// 6. 执行生成，输出到文件系统
	writer := infra.NewFileWriter(dst)
	if err := svc.Run(ctx, writer); err != nil {
		panic("code generation failed: " + err.Error())
	}
}
