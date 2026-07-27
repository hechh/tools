package infra

import (
	"path/filepath"

	futil "github.com/hechh/library/base/fileutil"
)

// FileWriter 文件系统输出适配器，实现 domain.OutputPort 接口
type FileWriter struct {
	dstDir string
}

// NewFileWriter 创建文件写入器
func NewFileWriter(dstDir string) *FileWriter {
	return &FileWriter{dstDir: dstDir}
}

// Write 实现 OutputPort 接口
func (w *FileWriter) Write(filename string, content []byte) error {
	cleanPath := filepath.ToSlash(filename)
	return futil.Save(filepath.Join(w.dstDir, cleanPath), content)
}
