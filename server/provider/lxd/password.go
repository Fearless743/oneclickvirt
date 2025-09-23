package lxd

import (
	"context"
	"fmt"

	"oneclickvirt/utils"
)

// SetInstancePassword 设置实例密码
func (l *LXDProvider) SetInstancePassword(ctx context.Context, instanceID, password string) error {
	if !l.connected {
		return fmt.Errorf("provider not connected")
	}

	// 根据执行规则判断使用哪种方式
	if l.shouldUseAPI() {
		if err := l.apiSetInstancePassword(ctx, instanceID, password); err == nil {
			return nil
		} else {
			// 检查是否可以回退到SSH
			if !l.shouldFallbackToSSH() {
				return fmt.Errorf("API调用失败且不允许回退到SSH: %w", err)
			}
		}
	}

	// 如果执行规则不允许使用SSH，则返回错误
	if !l.shouldUseSSH() {
		return fmt.Errorf("执行规则不允许使用SSH")
	}

	return l.sshSetInstancePassword(ctx, instanceID, password)
}

// ResetInstancePassword 重置实例密码
func (l *LXDProvider) ResetInstancePassword(ctx context.Context, instanceID string) (string, error) {
	if !l.connected {
		return "", fmt.Errorf("provider not connected")
	}

	// 密码重置通常只通过SSH进行，因为需要进入实例内部
	// 如果执行规则不允许使用SSH，则返回错误
	if !l.shouldUseSSH() {
		return "", fmt.Errorf("执行规则不允许使用SSH，无法重置实例密码")
	}

	// 生成随机密码
	newPassword := l.generateRandomPassword()

	// 设置新密码
	err := l.SetInstancePassword(ctx, instanceID, newPassword)
	if err != nil {
		return "", err
	}

	return newPassword, nil
}

// generateRandomPassword 生成随机密码（仅包含数字和大小写英文字母，长度不低于8位）
func (l *LXDProvider) generateRandomPassword() string {
	return utils.GenerateInstancePassword()
}
