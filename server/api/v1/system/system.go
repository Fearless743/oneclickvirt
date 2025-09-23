package system

import (
	"encoding/json"
	"net/http"
	"oneclickvirt/service/provider"
	"strconv"

	"oneclickvirt/global"
	"oneclickvirt/model/admin"
	"oneclickvirt/model/common"
	adminInstance "oneclickvirt/service/admin/instance"
	adminProvider "oneclickvirt/service/admin/provider"
	adminSystem "oneclickvirt/service/admin/system"
	adminUser "oneclickvirt/service/admin/user"
	"oneclickvirt/service/task"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GetSystemConfig 获取系统配置
// @Summary 获取系统配置
// @Description 获取系统的配置列表
// @Tags 系统管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Success 200 {object} common.Response{data=object} "获取成功"
// @Failure 400 {object} common.Response "参数错误"
// @Failure 500 {object} common.Response "获取失败"
// @Router /admin/config [get]
func GetSystemConfig(c *gin.Context) {
	var req admin.SystemConfigListRequest

	// 使用请求处理服务处理参数
	requestProcessService := provider.RequestProcessService{}
	if err := requestProcessService.ProcessSystemConfigListRequest(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, common.Response{
			Code: 400,
			Msg:  "参数错误",
		})
		return
	}

	systemService := adminSystem.NewService()
	configs, total, err := systemService.GetSystemConfigList(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.Response{
			Code: 500,
			Msg:  "获取系统配置失败",
		})
		return
	}
	c.JSON(http.StatusOK, common.Response{
		Code: 200,
		Msg:  "获取成功",
		Data: map[string]interface{}{
			"list":  configs,
			"total": total,
		},
	})
}

// UpdateSystemConfig 更新系统配置
// @Summary 更新系统配置
// @Description 更新系统的配置参数，支持单个配置项和批量配置
// @Tags 系统管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body admin.UpdateSystemConfigRequest true "更新配置请求参数"
// @Success 200 {object} common.Response "更新成功"
// @Failure 400 {object} common.Response "参数错误"
// @Failure 500 {object} common.Response "更新失败"
// @Router /admin/config [put]
func UpdateSystemConfig(c *gin.Context) {
	// 先读取原始JSON数据
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, common.Response{
			Code: 400,
			Msg:  "无法读取请求体",
		})
		return
	}

	// 尝试解析为批量配置请求
	var batchReq admin.BatchUpdateSystemConfigRequest
	if err := json.Unmarshal(body, &batchReq); err == nil && batchReq.Config != nil {
		// 处理批量配置更新
		systemService := adminSystem.NewService()
		if err := systemService.UpdateSystemConfigBatch(batchReq); err != nil {
			c.JSON(http.StatusInternalServerError, common.Response{
				Code: 500,
				Msg:  err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, common.Response{
			Code: 200,
			Msg:  "批量更新系统配置成功",
		})
		return
	}

	// 回退到单个配置项更新
	var req admin.UpdateSystemConfigRequest
	if err := json.Unmarshal(body, &req); err != nil {
		c.JSON(http.StatusBadRequest, common.Response{
			Code: 400,
			Msg:  "参数错误",
		})
		return
	}
	systemService := adminSystem.NewService()
	err = systemService.UpdateSystemConfig(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.Response{
			Code: 500,
			Msg:  err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, common.Response{
		Code: 200,
		Msg:  "更新系统配置成功",
	})
}

func GetAnnouncement(c *gin.Context) {
	// 获取查询参数
	announcementType := c.Query("type") // homepage, topbar 或者为空获取所有
	systemService := adminSystem.NewService()
	announcements, err := systemService.GetActiveAnnouncements(announcementType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.Response{
			Code: 500,
			Msg:  "获取公告列表失败",
		})
		return
	}
	c.JSON(http.StatusOK, common.Response{
		Code: 200,
		Msg:  "获取成功",
		Data: announcements,
	})
}

func GetUsers(c *gin.Context) {
	var req admin.UserListRequest

	// 使用请求处理服务处理参数
	requestProcessService := provider.RequestProcessService{}
	if err := requestProcessService.ProcessUserListRequest(c, &req); err != nil {
		common.ResponseWithError(c, common.NewError(common.CodeValidationError, "参数错误"))
		return
	}

	userService := adminUser.NewService()
	users, total, err := userService.GetUserList(req)
	if err != nil {
		common.ResponseWithError(c, common.NewError(common.CodeInternalError, "获取用户列表失败"))
		return
	}
	common.ResponseSuccessWithPagination(c, users, total, req.Page, req.PageSize)
}

func GetProviders(c *gin.Context) {
	providerIDStr := c.Param("id")
	providerID, err := strconv.ParseUint(providerIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.Response{
			Code: 400,
			Msg:  "无效的Provider ID",
		})
		return
	}
	providerService := adminProvider.NewService()
	status, err := providerService.GetProviderStatus(uint(providerID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.Response{
			Code: 500,
			Msg:  "获取状态失败: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, common.Response{
		Code: 200,
		Msg:  "获取状态成功",
		Data: status,
	})
}

func UpdateProviderStatus(c *gin.Context) {
	// 从URL路径参数获取ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.Response{
			Code: 400,
			Msg:  "无效的Provider ID",
		})
		return
	}
	var req admin.UpdateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		global.APP_LOG.Error("UpdateProvider参数绑定失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, common.Response{
			Code: 400,
			Msg:  "参数错误: " + err.Error(),
		})
		return
	}
	// 设置ID从URL参数
	req.ID = uint(id)
	providerService := adminProvider.NewService()
	if err := providerService.UpdateProvider(req); err != nil {
		c.JSON(http.StatusInternalServerError, common.Response{
			Code: 500,
			Msg:  err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, common.Response{
		Code: 200,
		Msg:  "更新提供商成功",
	})
}

func GetAllInstances(c *gin.Context) {
	var req admin.InstanceListRequest

	// 使用请求处理服务处理参数
	requestProcessService := provider.RequestProcessService{}
	if err := requestProcessService.ProcessInstanceListRequest(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, common.Response{
			Code: 400,
			Msg:  "参数错误",
		})
		return
	}

	instanceService := adminInstance.NewService(task.GetTaskService())
	instances, total, err := instanceService.GetInstanceList(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.Response{
			Code: 500,
			Msg:  "获取实例列表失败",
		})
		return
	}
	c.JSON(http.StatusOK, common.Response{
		Code: 200,
		Msg:  "获取成功",
		Data: map[string]interface{}{
			"list":  instances,
			"total": total,
		},
	})
}

func AdminInstanceAction(c *gin.Context) {
	instanceIDStr := c.Param("id")
	instanceID, err := strconv.ParseUint(instanceIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.Response{
			Code: 400,
			Msg:  "无效的实例ID",
		})
		return
	}

	var req struct {
		Action string `json:"action" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.Response{
			Code: 400,
			Msg:  "参数错误",
		})
		return
	}

	// 调用管理员实例操作服务
	adminReq := admin.InstanceActionRequest{
		Action: req.Action,
	}

	instanceService := adminInstance.NewService(task.GetTaskService())
	if err := instanceService.InstanceAction(uint(instanceID), adminReq); err != nil {
		c.JSON(http.StatusInternalServerError, common.Response{
			Code: 500,
			Msg:  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, common.Response{
		Code: 200,
		Msg:  "操作任务已创建，请查看任务列表了解进度",
	})
}

// GetProviderMonitoring 获取节点监控数据
// @Summary 获取节点监控数据
// @Description 获取节点的监控和性能数据
// @Tags 系统监控
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} common.Response{data=object} "获取成功"
// @Router /admin/monitoring/provider [get]
func GetProviderMonitoring(c *gin.Context) {
	c.JSON(http.StatusOK, common.Response{
		Code: 200,
		Msg:  "获取成功",
		Data: map[string]interface{}{
			"provider": []interface{}{},
		},
	})
}
