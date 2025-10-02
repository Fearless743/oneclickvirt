package traffic

import (
	"fmt"
	"time"

	"oneclickvirt/global"
	dashboardModel "oneclickvirt/model/dashboard"
	monitoringModel "oneclickvirt/model/monitoring"
	"oneclickvirt/model/user"

	"go.uber.org/zap"
)

// UserTrafficService 用户流量服务 - 提供基于vnStat的流量查询
type UserTrafficService struct {
	limitService *LimitService
}

// NewUserTrafficService 创建用户流量服务
func NewUserTrafficService() *UserTrafficService {
	return &UserTrafficService{
		limitService: NewLimitService(),
	}
}

// GetUserTrafficOverview 获取用户流量概览
func (s *UserTrafficService) GetUserTrafficOverview(userID uint) (map[string]interface{}, error) {
	// 获取用户信息
	var u user.User
	if err := global.APP_DB.First(&u, userID).Error; err != nil {
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}

	// 自动同步TotalTraffic为maxTraffic（如TotalTraffic为0时）
	if u.TotalTraffic == 0 {
		// 从等级配置中获取流量限额
		levelLimits, exists := global.APP_CONFIG.Quota.LevelLimits[u.Level]
		if exists && levelLimits.MaxTraffic > 0 {
			u.TotalTraffic = levelLimits.MaxTraffic
		}
	}

	// 获取基于vnStat的流量使用情况
	vnstatData, err := s.limitService.GetUserTrafficUsageWithVnStat(userID)
	if err != nil {
		global.APP_LOG.Warn("获取vnStat流量数据失败，返回基础信息",
			zap.Uint("userID", userID),
			zap.Error(err))

		// 降级到基础数据
		return map[string]interface{}{
			"user_id":             userID,
			"current_month_usage": u.UsedTraffic,
			"total_limit":         u.TotalTraffic,
			"usage_percent":       float64(u.UsedTraffic) / float64(u.TotalTraffic) * 100,
			"is_limited":          u.TrafficLimited,
			"reset_time":          u.TrafficResetAt,
			"data_source":         "legacy",
			"vnstat_available":    false,
		}, nil
	}

	// 强制同步total_limit为maxTraffic（如TotalTraffic为0时）
	if u.TotalTraffic > 0 {
		vnstatData["total_limit"] = u.TotalTraffic
	}

	// 数据源标识
	vnstatData["data_source"] = "vnstat"
	vnstatData["vnstat_available"] = true

	return vnstatData, nil
}

// GetInstanceTrafficDetail 获取实例流量详情
func (s *UserTrafficService) GetInstanceTrafficDetail(userID, instanceID uint) (map[string]interface{}, error) {
	// 验证用户权限
	if !s.hasInstanceAccess(userID, instanceID) {
		return nil, fmt.Errorf("用户无权限访问该实例")
	}

	// 获取实例的网络接口列表
	interfaces, err := s.getVnStatInterfaces(instanceID)
	if err != nil {
		global.APP_LOG.Warn("获取实例网络接口失败", zap.Error(err))
		interfaces = []*monitoringModel.VnStatInterface{}
	}

	// 格式化数据
	result := map[string]interface{}{
		"instance_id": instanceID,
		"interfaces":  interfaces,
	}

	return result, nil
}

// hasInstanceAccess 检查用户是否有实例访问权限
func (s *UserTrafficService) hasInstanceAccess(userID, instanceID uint) bool {
	var count int64
	err := global.APP_DB.Table("instances").
		Where("id = ? AND user_id = ?", instanceID, userID).
		Count(&count).Error
	if err != nil {
		return false
	}
	return count > 0
}

// getVnStatInterfaces 获取实例的vnStat接口列表
func (s *UserTrafficService) getVnStatInterfaces(instanceID uint) ([]*monitoringModel.VnStatInterface, error) {
	var interfaces []*monitoringModel.VnStatInterface
	err := global.APP_DB.Where("instance_id = ?", instanceID).Find(&interfaces).Error
	return interfaces, err
}

// GetUserInstancesTrafficSummary 获取用户所有实例的流量汇总
func (s *UserTrafficService) GetUserInstancesTrafficSummary(userID uint) (map[string]interface{}, error) {
	// 获取用户所有实例
	var instances []dashboardModel.InstanceSummary

	err := global.APP_DB.Table("instances").
		Select("id, name, status").
		Where("user_id = ?", userID).
		Find(&instances).Error

	if err != nil {
		return nil, fmt.Errorf("获取用户实例列表失败: %w", err)
	}

	result := map[string]interface{}{
		"user_id":        userID,
		"instance_count": len(instances),
		"instances":      []map[string]interface{}{},
	}

	var totalRx, totalTx, totalBytes int64
	instanceDetails := []map[string]interface{}{}

	// 获取当前月份
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	// 遍历每个实例获取流量数据
	for _, instance := range instances {
		// 获取实例的本月流量数据
		monthlyTraffic, err := s.limitService.service.getInstanceMonthlyTrafficFromVnStat(
			instance.ID,
			year, month,
		)
		if err != nil {
			global.APP_LOG.Warn("获取实例月度流量失败",
				zap.Uint("instanceID", instance.ID),
				zap.Error(err))
			continue
		}

		instanceDetail := map[string]interface{}{
			"id":              instance.ID,
			"name":            instance.Name,
			"status":          instance.Status,
			"monthly_traffic": monthlyTraffic,
		}

		// 累加总流量
		totalBytes += monthlyTraffic

		instanceDetails = append(instanceDetails, instanceDetail)
	}

	result["instances"] = instanceDetails
	result["total_traffic"] = map[string]interface{}{
		"rx":    totalRx,
		"tx":    totalTx,
		"total": totalBytes,
		"formatted": map[string]string{
			"rx":    FormatTrafficMB(totalRx),
			"tx":    FormatTrafficMB(totalTx),
			"total": FormatTrafficMB(totalBytes),
		},
	}

	return result, nil
}

// GetTrafficLimitStatus 获取流量限制状态
func (s *UserTrafficService) GetTrafficLimitStatus(userID uint) (map[string]interface{}, error) {
	// 检查用户流量限制状态
	isUserLimited, userReason, err := s.limitService.CheckUserTrafficLimitWithVnStat(userID)
	if err != nil {
		return nil, fmt.Errorf("检查用户流量限制失败: %w", err)
	}

	// 获取用户流量概览
	trafficOverview, err := s.GetUserTrafficOverview(userID)
	if err != nil {
		return nil, fmt.Errorf("获取流量概览失败: %w", err)
	}

	result := map[string]interface{}{
		"user_id":           userID,
		"is_user_limited":   isUserLimited,
		"user_limit_reason": userReason,
		"traffic_overview":  trafficOverview,
		"limited_instances": []map[string]interface{}{},
	}

	// 获取受限实例列表
	var limitedInstances []dashboardModel.LimitedInstanceSummary

	err = global.APP_DB.Table("instances").
		Where("user_id = ? AND traffic_limited = ?", userID, true).
		Find(&limitedInstances).Error

	if err != nil {
		global.APP_LOG.Warn("获取受限实例列表失败", zap.Error(err))
	} else {
		// 检查每个受限实例的Provider状态
		instanceDetails := []map[string]interface{}{}
		for _, instance := range limitedInstances {
			// 检查Provider是否也受限
			isProviderLimited, providerReason, providerErr := s.limitService.CheckProviderTrafficLimitWithVnStat(instance.ProviderID)

			instanceDetail := map[string]interface{}{
				"id":                    instance.ID,
				"name":                  instance.Name,
				"status":                instance.Status,
				"is_provider_limited":   isProviderLimited,
				"provider_limit_reason": providerReason,
			}

			if providerErr != nil {
				instanceDetail["provider_check_error"] = providerErr.Error()
			}

			instanceDetails = append(instanceDetails, instanceDetail)
		}
		result["limited_instances"] = instanceDetails
	}

	return result, nil
}
