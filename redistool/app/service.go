package app

import (
	"fmt"

	"github.com/hechh/tools/redistool/domain"
)

// Service 应用服务：编排 解析→生成→输出 的完整流程
type Service struct {
	generator *domain.Generator
}

// NewService 创建应用服务（DI注入）
func NewService(gen *domain.Generator) *Service {
	return &Service{generator: gen}
}

// Run 执行完整流程
func (s *Service) Run(ctx *domain.ParseContext, out domain.OutputPort) error {
	if err := s.generator.GenerateAll(ctx, out); err != nil {
		return fmt.Errorf("code generation failed: %w", err)
	}
	return nil
}
