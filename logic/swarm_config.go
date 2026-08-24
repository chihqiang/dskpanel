package logic

import (
	"context"
	"errors"

	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
)

// SwarmSecretItem Secret 列表项。
type SwarmSecretItem struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ListSecrets Secret 列表。
func (l *SwarmLogic) ListSecrets(ctx context.Context) ([]SwarmSecretItem, error) {
	cli, close, err := l.newClient(ctx)
	if err != nil {
		return nil, err
	}
	defer close()

	res, err := cli.SecretList(ctx, client.SecretListOptions{})
	if err != nil {
		return nil, err
	}
	items := make([]SwarmSecretItem, 0, len(res.Items))
	for _, s := range res.Items {
		items = append(items, SwarmSecretItem{
			ID:        s.ID,
			Name:      s.Spec.Annotations.Name,
			CreatedAt: s.Meta.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: s.Meta.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return items, nil
}

// CreateSecret 创建 Secret。
func (l *SwarmLogic) CreateSecret(ctx context.Context, name, data string) error {
	if name == "" {
		return errors.New("Secret 名称不能为空")
	}
	cli, close, err := l.newClient(ctx)
	if err != nil {
		return err
	}
	defer close()

	spec := swarm.SecretSpec{}
	spec.Annotations.Name = name
	spec.Data = []byte(data)
	_, err = cli.SecretCreate(ctx, client.SecretCreateOptions{Spec: spec})
	return err
}

// RemoveSecret 删除 Secret。
func (l *SwarmLogic) RemoveSecret(ctx context.Context, id string) error {
	cli, close, err := l.newClient(ctx)
	if err != nil {
		return err
	}
	defer close()
	_, err = cli.SecretRemove(ctx, id, client.SecretRemoveOptions{})
	return err
}

// SwarmConfigItem Config 列表项。
type SwarmConfigItem struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ListConfigs Config 列表。
func (l *SwarmLogic) ListConfigs(ctx context.Context) ([]SwarmConfigItem, error) {
	cli, close, err := l.newClient(ctx)
	if err != nil {
		return nil, err
	}
	defer close()

	res, err := cli.ConfigList(ctx, client.ConfigListOptions{})
	if err != nil {
		return nil, err
	}
	items := make([]SwarmConfigItem, 0, len(res.Items))
	for _, c := range res.Items {
		items = append(items, SwarmConfigItem{
			ID:        c.ID,
			Name:      c.Spec.Annotations.Name,
			CreatedAt: c.Meta.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: c.Meta.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return items, nil
}

// CreateConfig 创建 Config。
func (l *SwarmLogic) CreateConfig(ctx context.Context, name, data string) error {
	if name == "" {
		return errors.New("Config 名称不能为空")
	}
	cli, close, err := l.newClient(ctx)
	if err != nil {
		return err
	}
	defer close()

	spec := swarm.ConfigSpec{}
	spec.Annotations.Name = name
	spec.Data = []byte(data)
	_, err = cli.ConfigCreate(ctx, client.ConfigCreateOptions{Spec: spec})
	return err
}

// RemoveConfig 删除 Config。
func (l *SwarmLogic) RemoveConfig(ctx context.Context, id string) error {
	cli, close, err := l.newClient(ctx)
	if err != nil {
		return err
	}
	defer close()
	_, err = cli.ConfigRemove(ctx, id, client.ConfigRemoveOptions{})
	return err
}

// SecretDetail Secret 详情。
// 注意：Docker API 出于安全考虑，Secret 内容（解密后的 data）不通过 inspect 返回，
// 因此这里只返回元信息，不包含 data。
type SecretDetail struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Labels    map[string]string `json:"labels"`
	CreatedAt string            `json:"created_at"`
	UpdatedAt string            `json:"updated_at"`
}

// InspectSecret Secret 详情（仅元信息，Docker 不返回 Secret 明文）。
func (l *SwarmLogic) InspectSecret(ctx context.Context, id string) (*SecretDetail, error) {
	cli, close, err := l.newClient(ctx)
	if err != nil {
		return nil, err
	}
	defer close()

	res, err := cli.SecretInspect(ctx, id, client.SecretInspectOptions{})
	if err != nil {
		return nil, err
	}
	s := res.Secret
	return &SecretDetail{
		ID:        s.ID,
		Name:      s.Spec.Annotations.Name,
		Labels:    s.Spec.Annotations.Labels,
		CreatedAt: s.Meta.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: s.Meta.UpdatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// ConfigDetail Config 详情（含数据）。
type ConfigDetail struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Data      string            `json:"data"`
	Labels    map[string]string `json:"labels"`
	CreatedAt string            `json:"created_at"`
	UpdatedAt string            `json:"updated_at"`
}

// InspectConfig Config 详情。
func (l *SwarmLogic) InspectConfig(ctx context.Context, id string) (*ConfigDetail, error) {
	cli, close, err := l.newClient(ctx)
	if err != nil {
		return nil, err
	}
	defer close()

	res, err := cli.ConfigInspect(ctx, id, client.ConfigInspectOptions{})
	if err != nil {
		return nil, err
	}
	c := res.Config
	return &ConfigDetail{
		ID:        c.ID,
		Name:      c.Spec.Annotations.Name,
		Data:      string(c.Spec.Data),
		Labels:    c.Spec.Annotations.Labels,
		CreatedAt: c.Meta.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: c.Meta.UpdatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}
