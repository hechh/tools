package domain

// OutputPort 文件输出端口（领域层定义，基础设施层实现）
// 遵循依赖倒置原则：领域层只依赖抽象，不依赖具体文件系统
type OutputPort interface {
	// Write 写入单个生成文件
	Write(filename string, content []byte) error
}
