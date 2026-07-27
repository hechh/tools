package domain

import (
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// ProtoRegistry Proto类型注册表
type ProtoRegistry struct {
	Files *protoregistry.Files
	Types map[string]*Proto
	Pkg   string
	GoPkg string
}

// NewProtoRegistry 创建注册表
func NewProtoRegistry() *ProtoRegistry {
	return &ProtoRegistry{
		Files: &protoregistry.Files{},
		Types: make(map[string]*Proto),
	}
}

// Add 添加类型定义
func (r *ProtoRegistry) Add(name string, p *Proto) {
	r.Types[name] = p
}

// Get 获取类型定义
func (r *ProtoRegistry) Get(name string) (*Proto, bool) {
	p, ok := r.Types[name]
	return p, ok
}

// SetPkgInfo 设置package信息(从首个扫描的proto文件提取)
func (r *ProtoRegistry) SetPkgInfo(pkg, goPkg string) {
	if pkg != "" && r.Pkg == "" {
		r.Pkg = pkg
	}
	if goPkg != "" && r.GoPkg == "" {
		r.GoPkg = goPkg
	}
}

// FindMessage 查找消息描述符
func (r *ProtoRegistry) FindMessage(name string) protoreflect.MessageDescriptor {
	p, ok := r.Types[name]
	if !ok {
		return nil
	}
	fullName := protoreflect.FullName(p.Pkg + "." + name)
	desc, err := r.Files.FindDescriptorByName(fullName)
	if err != nil {
		return nil
	}
	msg, ok := desc.(protoreflect.MessageDescriptor)
	if !ok {
		return nil
	}
	return msg
}

// FindEnum 查找枚举描述符
func (r *ProtoRegistry) FindEnum(name string) protoreflect.EnumDescriptor {
	p, ok := r.Types[name]
	if !ok {
		return nil
	}
	fullName := protoreflect.FullName(p.Pkg + "." + name)
	desc, err := r.Files.FindDescriptorByName(fullName)
	if err != nil {
		return nil
	}
	enum, ok := desc.(protoreflect.EnumDescriptor)
	if !ok {
		return nil
	}
	return enum
}
