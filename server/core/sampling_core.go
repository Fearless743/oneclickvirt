package core

import (
	"sync"
	"time"

	"go.uber.org/zap/zapcore"
)

// SamplingCore 是一个包装的zapcore.Core，用于控制日志采样
type SamplingCore struct {
	zapcore.Core
	samplers map[string]*sampler
	mu       sync.RWMutex
}

// sampler 用于控制特定消息的采样频率
type sampler struct {
	interval    time.Duration
	lastLog     time.Time
	skipCount   int64
	maxSkipLogs int64
}

// NewSamplingCore 创建一个新的采样核心
func NewSamplingCore(core zapcore.Core) *SamplingCore {
	return &SamplingCore{
		Core:     core,
		samplers: make(map[string]*sampler),
	}
}

// Check 检查是否应该记录该日志
func (s *SamplingCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	// 对于错误和致命级别的日志，总是记录
	if entry.Level >= zapcore.ErrorLevel {
		return s.Core.Check(entry, ce)
	}

	// 对于调试和信息级别的日志，进行采样
	if s.shouldSample(entry.Message, entry.Level) {
		return s.Core.Check(entry, ce)
	}

	return ce
}

// shouldSample 判断是否应该记录该消息
func (s *SamplingCore) shouldSample(message string, level zapcore.Level) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	samp, exists := s.samplers[message]

	if !exists {
		// 为新消息创建采样器
		interval := s.getSamplingInterval(level, message)
		s.samplers[message] = &sampler{
			interval:    interval,
			lastLog:     now,
			maxSkipLogs: s.getMaxSkipLogs(level),
		}
		return true
	}

	// 检查是否达到采样间隔
	if now.Sub(samp.lastLog) >= samp.interval {
		samp.lastLog = now
		if samp.skipCount > 0 {
			// 如果跳过了一些日志，重置计数
			samp.skipCount = 0
		}
		return true
	}

	// 检查是否超过最大跳过次数
	samp.skipCount++
	if samp.skipCount >= samp.maxSkipLogs {
		samp.lastLog = now
		samp.skipCount = 0
		return true
	}

	return false
}

// getSamplingInterval 根据日志级别和消息内容获取采样间隔
func (s *SamplingCore) getSamplingInterval(level zapcore.Level, message string) time.Duration {
	// 对于不同的消息设置不同的采样间隔
	switch {
	case contains(message, "存储目录创建成功"):
		return 10 * time.Second // 存储目录相关消息每10秒最多记录一次
	case contains(message, "监控数据"):
		return 30 * time.Second // 监控数据每30秒最多记录一次
	case contains(message, "流量数据"):
		return 1 * time.Minute // 流量数据每分钟最多记录一次
	case contains(message, "Task"):
		return 5 * time.Second // 任务相关消息每5秒最多记录一次
	case level == zapcore.DebugLevel:
		return 5 * time.Second // Debug级别消息每5秒最多记录一次
	case level == zapcore.InfoLevel:
		return 2 * time.Second // Info级别消息每2秒最多记录一次
	default:
		return 1 * time.Second
	}
}

// getMaxSkipLogs 获取最大跳过日志数
func (s *SamplingCore) getMaxSkipLogs(level zapcore.Level) int64 {
	switch level {
	case zapcore.DebugLevel:
		return 10 // Debug级别最多跳过10次
	case zapcore.InfoLevel:
		return 5 // Info级别最多跳过5次
	default:
		return 3
	}
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		len(substr) > 0 &&
		findSubstring(s, substr)
}

// findSubstring 查找子串
func findSubstring(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// CleanupOldSamplers 定期清理旧的采样器
func (s *SamplingCore) CleanupOldSamplers() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for message, samp := range s.samplers {
		// 如果采样器超过1小时没有使用，删除它
		if now.Sub(samp.lastLog) > time.Hour {
			delete(s.samplers, message)
		}
	}
}
