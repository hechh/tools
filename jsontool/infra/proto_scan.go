package infra

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hechh/tools/jsontool/domain"
)

// ScanProtoDir 扫描proto目录,文本解析填充注册表（无 protobuf 依赖）
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
