package infra

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hechh/tools/xlsxtool/domain"

	"github.com/bufbuild/protocompile"
)

// ScanProtoDir 扫描proto目录,文本解析填充注册表
func ScanProtoDir(protoDir string, registry *domain.ProtoRegistry) error {
	pkgRegex := regexp.MustCompile(`^\s*package\s+([^;]+);`)
	gopkgRegex := regexp.MustCompile(`^\s*option\s+go_package\s*=\s*"([^"]+)"\s*;`)
	structRegex := regexp.MustCompile(`^\s*message\s+(\w+)\s*\{?`)
	enumRegex := regexp.MustCompile(`^\s*enum\s+(\w+)\s*\{?`)

	return filepath.Walk(protoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".proto") {
			return nil
		}
		return scanProtoFile(path, pkgRegex, gopkgRegex, structRegex, enumRegex, registry)
	})
}

func scanProtoFile(filename string, pkgRe, gopkgRe, structRe, enumRe *regexp.Regexp, registry *domain.ProtoRegistry) error {
	fp, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer fp.Close()

	scanner := bufio.NewScanner(fp)
	var filePkg, fileGoPkg string
	for scanner.Scan() {
		line := scanner.Text()

		if m := pkgRe.FindStringSubmatch(line); m != nil {
			filePkg = strings.TrimSpace(m[1])
		}
		if m := gopkgRe.FindStringSubmatch(line); m != nil {
			fileGoPkg = m[1]
		}
		if m := enumRe.FindStringSubmatch(line); m != nil {
			registry.Add(m[1], &domain.Proto{
				Type: m[1], Pkg: filePkg, GoPkg: fileGoPkg,
				Filename: filepath.Base(filename), IsEnum: true,
			})
		}
		if m := structRe.FindStringSubmatch(line); m != nil {
			registry.Add(m[1], &domain.Proto{
				Type: m[1], Pkg: filePkg, GoPkg: fileGoPkg,
				Filename: filepath.Base(filename),
			})
		}
	}
	registry.SetPkgInfo(filePkg, fileGoPkg)
	return scanner.Err()
}

// CompileProtoDir 编译proto目录中的所有文件到registry
func CompileProtoDir(protoDir string, registry *domain.ProtoRegistry, extraImports ...string) error {
	var files []string
	err := filepath.Walk(protoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".proto") {
			return nil
		}
		rel, _ := filepath.Rel(protoDir, path)
		files = append(files, rel)
		return nil
	})
	if err != nil || len(files) == 0 {
		return err
	}

	importPaths := append([]string{protoDir}, extraImports...)
	compiler := protocompile.Compiler{
		Resolver:       &protocompile.SourceResolver{ImportPaths: importPaths},
		SourceInfoMode: protocompile.SourceInfoNone,
	}
	results, err := compiler.Compile(context.Background(), files...)
	if err != nil {
		return err
	}
	for _, r := range results {
		if err := registry.Files.RegisterFile(r); err != nil {
			return err
		}
	}
	return nil
}
