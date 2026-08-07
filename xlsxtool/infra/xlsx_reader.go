package infra

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hechh/tools/xlsxtool/domain"

	"github.com/xuri/excelize/v2"
)

// ReadTables 从目录读取所有xlsx文件的表格数据
func ReadTables(xlsxDir string) []*domain.Table {
	var tables []*domain.Table

	filepath.Walk(xlsxDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".xlsx") {
			return nil
		}
		ts, err := readFile(path)
		if err != nil {
			fmt.Printf("[Error] 解析%s失败 \n", filepath.Base(path))
			return nil
		}
		tables = append(tables, ts...)
		return nil
	})
	return tables
}

// readFile 读取单个xlsx文件的生成表
func readFile(filename string) ([]*domain.Table, error) {
	fp, err := excelize.OpenFile(filename)
	if err != nil {
		return nil, err
	}
	defer fp.Close()

	rows, err := fp.GetRows("生成表")
	if err != nil {
		return nil, err
	}

	var tables []*domain.Table
	for _, lines := range rows {
		for _, str := range lines {
			if len(str) <= 0 {
				continue
			}
			vals := strings.Split(str, "|")
			switch {
			case strings.HasPrefix(strings.ToLower(str), "@enum|"):
				sheetRows, err := fp.GetRows(vals[1])
				if err != nil {
					return nil, err
				}
				tables = append(tables, &domain.Table{
					Sheet: vals[1], Rows: sheetRows, Token: 1,
				})
			case strings.HasPrefix(strings.ToLower(str), "@struct|"):
				sheet, typ := vals[1], vals[1]
				if pos := strings.Index(vals[1], "@"); pos >= 0 {
					sheet, typ = vals[1][:pos], vals[1][pos+1:]
				}
				sheetRows, err := fp.GetRows(sheet)
				if err != nil {
					return nil, err
				}
				tables = append(tables, &domain.Table{
					Sheet: sheet, Type: typ,
					Rules: vals[2:], Rows: sheetRows, Token: 2,
				})
			case strings.HasPrefix(strings.ToLower(str), "@struct:col|"):
				sheet, typ := vals[1], vals[1]
				if pos := strings.Index(vals[1], "@"); pos >= 0 {
					sheet, typ = vals[1][:pos], vals[1][pos+1:]
				}
				cols, err := fp.GetCols(sheet)
				if err != nil {
					return nil, err
				}
				tables = append(tables, &domain.Table{
					Sheet: sheet, Type: typ,
					Rules: vals[2:], Rows: cols, Token: 3,
				})
			}
		}
	}
	return tables, nil
}
