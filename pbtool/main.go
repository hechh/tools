package main

import (
	"flag"

	futil "github.com/hechh/library/base/fileutil"
	"github.com/hechh/tools/pbtool/internal"
)

func main() {
	var src string
	flag.StringVar(&src, "src", "", ".pb.go文件目录")
	flag.Parse()

	files, err := futil.Glob(src, ".*\\.pb\\.go", true)
	if err != nil {
		panic(err)
	}

	parser := &internal.Parser{}
	if err := futil.ParseFiles(parser, files...); err != nil {
		panic(err)
	}
	// 生成文件
	if err := parser.Gen(src); err != nil {
		panic(err)
	}
}
