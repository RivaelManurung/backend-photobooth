package services

import (
	"context"
	"fmt"
)

type AIImageProvider interface {
	Generate(ctx context.Context, prompt string) ([]byte, error)
	IsEnabled() bool
}

type DisabledAIProvider struct{}

func (DisabledAIProvider) Generate(ctx context.Context, prompt string) ([]byte, error) {
	return nil, fmt.Errorf("ai image generation is disabled")
}

func (DisabledAIProvider) IsEnabled() bool {
	return false
}

type LocalComfyUIProvider struct {
	Endpoint string
	Enabled  bool
}

func (p LocalComfyUIProvider) Generate(ctx context.Context, prompt string) ([]byte, error) {
	if !p.Enabled {
		return nil, fmt.Errorf("local ai provider is disabled")
	}
	return nil, fmt.Errorf("local ComfyUI provider placeholder is not implemented")
}

func (p LocalComfyUIProvider) IsEnabled() bool {
	return p.Enabled
}
