package workspace

import (
	"context"
	"fmt"

	"github.com/hsldymq/gomount/internal/config"
	"github.com/hsldymq/gomount/internal/drivers"
)

// Manager 工作区管理器
type Manager struct {
	cfg           *config.Config
	driverManager *drivers.Manager
}

// NewManager 创建工作区管理器
func NewManager(cfg *config.Config, driverManager *drivers.Manager) *Manager {
	return &Manager{
		cfg:           cfg,
		driverManager: driverManager,
	}
}

// MountWorkspace 挂载整个工作区
func (m *Manager) MountWorkspace(ctx context.Context, workspaceName string) (*OperationResult, error) {
	workspace, err := m.findWorkspace(workspaceName)
	if err != nil {
		return nil, err
	}

	result := &OperationResult{
		WorkspaceName: workspaceName,
		Total:         len(workspace.Mounts),
		Succeeded:     []string{},
		Failed:        make(map[string]error),
	}

	for _, mountName := range workspace.Mounts {
		if err := m.driverManager.Mount(ctx, mountName); err != nil {
			result.Failed[mountName] = err
		} else {
			result.Succeeded = append(result.Succeeded, mountName)
		}
	}

	return result, nil
}

// UnmountWorkspace 卸载整个工作区
func (m *Manager) UnmountWorkspace(ctx context.Context, workspaceName string) (*OperationResult, error) {
	workspace, err := m.findWorkspace(workspaceName)
	if err != nil {
		return nil, err
	}

	result := &OperationResult{
		WorkspaceName: workspaceName,
		Total:         len(workspace.Mounts),
		Succeeded:     []string{},
		Failed:        make(map[string]error),
	}

	for _, mountName := range workspace.Mounts {
		if err := m.driverManager.Unmount(ctx, mountName); err != nil {
			result.Failed[mountName] = err
		} else {
			result.Succeeded = append(result.Succeeded, mountName)
		}
	}

	return result, nil
}

// GetWorkspaceStatus 获取工作区状态
func (m *Manager) GetWorkspaceStatus(ctx context.Context, workspaceName string) (*StatusResult, error) {
	workspace, err := m.findWorkspace(workspaceName)
	if err != nil {
		return nil, err
	}

	result := &StatusResult{
		WorkspaceName: workspaceName,
		Mounts:        make(map[string]*drivers.MountStatus),
	}

	for _, mountName := range workspace.Mounts {
		status, err := m.driverManager.Status(ctx, mountName)
		if err != nil {
			result.Mounts[mountName] = &drivers.MountStatus{
				Mounted: false,
				Message: fmt.Sprintf("Error: %v", err),
			}
		} else {
			result.Mounts[mountName] = status
		}
	}

	return result, nil
}

// ListWorkspaces 列出所有工作区
func (m *Manager) ListWorkspaces() []config.Workspace {
	if m.cfg.Workspaces == nil {
		return []config.Workspace{}
	}
	return m.cfg.Workspaces
}

// GetWorkspace 获取指定工作区
func (m *Manager) GetWorkspace(name string) (*config.Workspace, error) {
	return m.findWorkspace(name)
}

// findWorkspace 查找工作区
func (m *Manager) findWorkspace(name string) (*config.Workspace, error) {
	for i := range m.cfg.Workspaces {
		if m.cfg.Workspaces[i].Name == name {
			return &m.cfg.Workspaces[i], nil
		}
	}
	return nil, fmt.Errorf("workspace not found: %s", name)
}

// OperationResult 操作结果
type OperationResult struct {
	WorkspaceName string
	Total         int
	Succeeded     []string
	Failed        map[string]error
}

// SuccessCount 返回成功数量
func (r *OperationResult) SuccessCount() int {
	return len(r.Succeeded)
}

// FailureCount 返回失败数量
func (r *OperationResult) FailureCount() int {
	return len(r.Failed)
}

// IsSuccess 返回是否全部成功
func (r *OperationResult) IsSuccess() bool {
	return len(r.Failed) == 0
}

// StatusResult 状态结果
type StatusResult struct {
	WorkspaceName string
	Mounts        map[string]*drivers.MountStatus
}

// MountedCount 返回已挂载数量
func (r *StatusResult) MountedCount() int {
	count := 0
	for _, status := range r.Mounts {
		if status.Mounted {
			count++
		}
	}
	return count
}
