package auth

import "context"

type ConfigProvider interface {
	GetConfig(ctx context.Context) (Config, error)
}

type StaticConfigProvider struct {
	Config Config
}

func (p StaticConfigProvider) GetConfig(_ context.Context) (Config, error) {
	return p.Config, nil
}
