package infra

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/hechh/tools/redistool/domain"

	"github.com/iancoleman/strcase"
)

// ASTParser 基础设施层：AST源码解析器
// 职责：将Go源码中的 @dbtool 注解解析为领域模型(RedisString/RedisHash)
type ASTParser struct {
	ctx   *domain.ParseContext
	fset  *token.FileSet
	rules []string // 当前正在处理的注解规则列表
}

// NewASTParser 创建AST解析器
func NewASTParser(ctx *domain.ParseContext) *ASTParser {
	return &ASTParser{
		ctx:  ctx,
		fset: token.NewFileSet(),
	}
}

// GetContext 返回填充后的领域上下文（供外部获取解析结果）
func (p *ASTParser) GetContext() *domain.ParseContext { return p.ctx }

// Visit 实现ast.Visitor接口，遍历AST提取@dbtool注解定义的模型
func (p *ASTParser) Visit(n ast.Node) ast.Visitor {
	switch vv := n.(type) {
	case *ast.File:
		return p
	case *ast.GenDecl:
		return p.extractRules(vv)
	case *ast.TypeSpec:
		p.extractModels(vv)
		return nil
	}
	return nil
}

// extractRules 从GenDecl中提取@dbtool注解规则
func (p *ASTParser) extractRules(decl *ast.GenDecl) ast.Visitor {
	if decl.Doc == nil {
		return nil
	}
	p.rules = p.rules[:0]
	for _, comment := range decl.Doc.List {
		rule := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
		if strings.HasPrefix(rule, "@dbtool") {
			p.rules = append(p.rules, rule)
		}
	}
	if len(p.rules) > 0 {
		return p
	}
	return nil
}

// extractModels 从TypeSpec中根据规则构建领域模型
func (p *ASTParser) extractModels(spec *ast.TypeSpec) {
	if _, isStruct := spec.Type.(*ast.StructType); !isStruct {
		return
	}
	for _, rule := range p.rules {
		parts := strings.Split(rule, "|")
		switch ruleType(parts[0]) {
		case "string":
			if m := p.buildStringModel(parts, spec.Name.Name); m != nil {
				p.ctx.AddString(m)
			}
		case "hash":
			if m := p.buildHashModel(parts, spec.Name.Name); m != nil {
				p.ctx.AddHash(m)
			}
		}
	}
}

// countFmtVerbs 统计格式化串中 %s/%d 等格式占位符数量
func countFmtVerbs(fmtStr string) int {
	count := 0
	for i := 0; i < len(fmtStr); i++ {
		if fmtStr[i] == '%' && i+1 < len(fmtStr) {
			switch fmtStr[i+1] {
			case 's', 'd', 'f', 'v', 'x', 'X', 'o':
				count++
			}
		}
	}
	return count
}

// validateFormat 校验格式串中的占位符数量与参数列表是否匹配，不匹配返回错误
func validateFormat(format string, fields []*domain.Field, rule, structName string) error {
	if len(fields) == 0 {
		return nil // 无参数时跳过校验（纯静态key）
	}
	expected := countFmtVerbs(format)
	if expected != len(fields) {
		fmt.Printf("[redistool] 跳过无效规则: %s (结构体 %s): 格式串 %q 需要 %d 个参数但声明了 %d 个 (%v)\n",
			rule, structName, format, expected, len(fields), fieldNames(fields))
		return fmt.Errorf("format/args mismatch")
	}
	return nil
}

func fieldNames(fields []*domain.Field) []string {
	names := make([]string, len(fields))
	for i, f := range fields {
		names[i] = f.Name + "@" + f.Type
	}
	return names
}

// buildStringModel 构建String类型领域模型
// 格式: @dbtool:string|DbSpec|keyFormat
//
//	例: @dbtool:string|shards:uid@uint64|user_info:%d:%s
func (p *ASTParser) buildStringModel(parts []string, structName string) *domain.RedisString {
	dbType, dbName, shardField := ParseDbSpec(parts[1])
	format, keys := ParseFieldFormat(parts[2])

	// shards 模式必须有有效的分片参数（global 常量格式允许 shardField 为 nil）
	if dbType == domain.DbTypeShards && shardField == nil {
		fmt.Printf("[redistool] 跳过无效规则: %s (结构体 %s): shards 模式缺少有效的参数声明\n",
			parts[0], structName)
		return nil
	}

	if err := validateFormat(format, keys, parts[0], structName); err != nil {
		return nil
	}

	// 当 ShardField 与首个 Key 字段同名时统一名称，避免函数签名参数重复
	if shardField != nil && len(keys) > 0 && strings.EqualFold(shardField.Name, keys[0].Name) {
		keys[0].Name = shardField.Name // 统一为 ShardField 的命名
	}

	return &domain.RedisString{
		Pkg:        strcase.ToSnake(structName),
		Name:       structName,
		DbType:     dbType,
		DbName:     dbName,
		ShardField: shardField,
		Format:     format,
		Keys:       keys,
	}
}

// buildHashModel 构建Hash类型领域模型
// 必须为4段格式: @dbtool:hash|DbSpec|keyFmt|fieldFmt
//
//	例: @dbtool:hash|shards:uid@uint64|user_info:%d|Phone:%s
func (p *ASTParser) buildHashModel(parts []string, structName string) *domain.RedisHash {
	if len(parts) < 4 {
		fmt.Printf("[redistool] 跳过无效规则: %s (结构体 %s): Hash 类型必须使用4段格式 @dbtool:hash|DbSpec|keyFmt|fieldFmt\n",
			parts[0], structName)
		return nil
	}

	dbType, dbName, shardField := ParseDbSpec(parts[1])

	// shards 模式必须有有效的分片参数（global 常量格式允许 shardField 为 nil）
	if dbType == domain.DbTypeShards && shardField == nil {
		fmt.Printf("[redistool] 跳过无效规则: %s (结构体 %s): shards 模式缺少有效的参数声明\n",
			parts[0], structName)
		return nil
	}

	keyFmt, keys := ParseFieldFormat(parts[2])
	fieldFmt, fields := ParseFieldFormat(parts[3])

	if err := validateFormat(keyFmt, keys, parts[0], structName); err != nil {
		return nil
	}
	if err := validateFormat(fieldFmt, fields, parts[0], structName); err != nil {
		return nil
	}

	// 当 ShardField 与 Key 同名时统一名称，避免函数签名参数重复
	if shardField != nil && len(keys) > 0 && strings.EqualFold(shardField.Name, keys[0].Name) {
		keys[0].Name = shardField.Name // 统一为 ShardField 的命名
	}
	// 当 ShardField 与 Field 同名时统一名称，避免函数签名参数重复 + 大小写不一致
	if shardField != nil && len(fields) > 0 && strings.EqualFold(shardField.Name, fields[0].Name) {
		fields[0].Name = shardField.Name // 统一为 ShardField 的命名
	}

	return &domain.RedisHash{
		Pkg:        strcase.ToSnake(structName),
		Name:       structName,
		DbType:     dbType,
		DbName:     dbName,
		ShardField: shardField,
		KeyFmt:     keyFmt,
		Keys:       keys,
		FieldFmt:   fieldFmt,
		Fields:     fields,
	}
}

// ruleType 提取规则类型标识
func ruleType(s string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimPrefix(s, "@dbtool:")), ":cache")
}
