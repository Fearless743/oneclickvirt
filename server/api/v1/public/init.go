package public

import (
	"net/http"
	"oneclickvirt/service/auth"
	"oneclickvirt/service/resources"
	"oneclickvirt/service/system"
	"strconv"

	"oneclickvirt/global"
	"oneclickvirt/model/common"
	configModel "oneclickvirt/model/config"
	"oneclickvirt/source"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CheckInit 检查系统初始化状态
// @Summary 检查系统初始化状态
// @Description 检查系统是否需要进行初始化设置
// @Tags 系统初始化
// @Accept json
// @Produce json
// @Success 200 {object} common.Response{data=object} "检查成功"
// @Router /public/init/check [get]
func CheckInit(c *gin.Context) {
	var (
		message  = "前往初始化数据库"
		needInit = true
	)

	// 检查数据库连接是否存在
	if global.APP_DB == nil {
		message = "数据库未连接，需要初始化"
		needInit = true
		global.APP_LOG.Info("数据库连接为空，需要初始化")
	} else {
		// 验证数据库连接是否有效
		sqlDB, err := global.APP_DB.DB()
		if err != nil {
			message = "数据库连接无效，需要初始化"
			needInit = true
			global.APP_LOG.Warn("获取数据库连接失败", zap.Error(err))
		} else if err := sqlDB.Ping(); err != nil {
			message = "数据库连接测试失败，需要初始化"
			needInit = true
			global.APP_LOG.Warn("数据库连接ping失败", zap.Error(err))
		} else {
			// 使用服务层检查是否有用户数据
			systemStatsService := resources.SystemStatsService{}
			hasUsers, err := systemStatsService.CheckUserExists()
			if err != nil {
				message = "数据库查询失败，需要初始化"
				needInit = true
				global.APP_LOG.Warn("查询用户表失败", zap.Error(err))
			} else if !hasUsers {
				message = "未找到用户数据，需要初始化"
				needInit = true
				global.APP_LOG.Info("数据库中无用户数据，需要初始化")
			} else {
				message = "数据库无需初始化"
				needInit = false
				global.APP_LOG.Debug("系统已初始化")
			}
		}
	}

	global.APP_LOG.Debug("初始化状态检查",
		zap.Bool("needInit", needInit),
		zap.String("message", message))

	c.JSON(http.StatusOK, common.Success(gin.H{
		"needInit": needInit,
		"message":  message,
	}))
}

// TestDatabaseConnection 测试数据库连接
// @Summary 测试数据库连接
// @Description 测试数据库连接是否可用，用于初始化前验证数据库配置
// @Tags 系统初始化
// @Accept json
// @Produce json
// @Param request body object true "数据库连接参数"
// @Success 200 {object} common.Response "连接成功"
// @Failure 400 {object} common.Response "参数错误"
// @Failure 500 {object} common.Response "连接失败"
// @Router /public/test-db-connection [post]
func TestDatabaseConnection(c *gin.Context) {
	var req struct {
		Type     string `json:"type" binding:"required"`
		Host     string `json:"host" binding:"required"`
		Port     string `json:"port" binding:"required"`
		Database string `json:"database" binding:"required"`
		Username string `json:"username" binding:"required"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		global.APP_LOG.Warn("数据库连接测试参数错误", zap.Error(err))
		c.JSON(http.StatusBadRequest, common.Error("参数错误"))
		return
	}

	// 目前只支持MySQL
	if req.Type != "mysql" {
		c.JSON(http.StatusBadRequest, common.Error("暂时只支持MySQL数据库"))
		return
	}

	// 使用InitService测试连接
	initService := &system.InitService{}

	// 转换端口字符串为整数
	port, err := strconv.Atoi(req.Port)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.Error("端口格式错误"))
		return
	}

	dbConfig := configModel.DatabaseConfig{
		Type:     req.Type,
		Host:     req.Host,
		Port:     port,
		Database: req.Database,
		Username: req.Username,
		Password: req.Password,
	}

	if err := initService.TestDatabaseConnection(dbConfig); err != nil {
		global.APP_LOG.Warn("数据库连接测试失败",
			zap.String("host", req.Host),
			zap.String("port", req.Port),
			zap.String("database", req.Database),
			zap.String("username", req.Username),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, common.Error("数据库连接失败: "+err.Error()))
		return
	}

	global.APP_LOG.Info("数据库连接测试成功",
		zap.String("host", req.Host),
		zap.String("port", req.Port),
		zap.String("database", req.Database),
		zap.String("username", req.Username))

	c.JSON(http.StatusOK, common.Success("数据库连接测试成功"))
}

// InitSystem 初始化系统
// @Summary 初始化系统
// @Description 进行系统的初始化设置，创建管理员和默认用户
// @Tags 系统初始化
// @Accept json
// @Produce json
// @Param request body object true "初始化请求参数"
// @Success 200 {object} common.Response "初始化成功"
// @Failure 400 {object} common.Response "参数错误或系统已初始化"
// @Failure 500 {object} common.Response "初始化失败"
// @Router /public/init [post]
func InitSystem(c *gin.Context) {
	var req struct {
		Admin struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
			Email    string `json:"email" binding:"required"`
		} `json:"admin" binding:"required"`
		User struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
			Email    string `json:"email" binding:"required"`
		} `json:"user" binding:"required"`
		Database struct {
			Type     string `json:"type" binding:"required"`
			Host     string `json:"host"`
			Port     string `json:"port"`
			Database string `json:"database"`
			Username string `json:"username"`
			Password string `json:"password"`
		} `json:"database" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.Error("参数错误"))
		return
	}

	// 如果数据库已经初始化，使用服务层检查是否有用户数据
	if global.APP_DB != nil {
		systemStatsService := resources.SystemStatsService{}
		hasUsers, err := systemStatsService.CheckUserExists()
		if err != nil {
			c.JSON(http.StatusInternalServerError, common.Error("检查用户数据失败: "+err.Error()))
			return
		}
		if hasUsers {
			c.JSON(http.StatusBadRequest, common.Error("系统已初始化"))
			return
		}
	}

	// 初始化服务
	initService := &system.InitService{}

	// 转换端口字符串为整数
	port, err := strconv.Atoi(req.Database.Port)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.Error("数据库端口格式错误"))
		return
	}

	// 确保数据库和表结构
	dbConfig := configModel.DatabaseConfig{
		Type:     req.Database.Type,
		Host:     req.Database.Host,
		Port:     port,
		Database: req.Database.Database,
		Username: req.Database.Username,
		Password: req.Database.Password,
	}

	if err := initService.EnsureDatabase(dbConfig); err != nil {
		c.JSON(http.StatusInternalServerError, common.Error("数据库初始化失败: "+err.Error()))
		return
	}

	// 创建用户
	authService := auth.AuthService{}
	adminInfo := auth.UserInfo{
		Username: req.Admin.Username,
		Password: req.Admin.Password,
		Email:    req.Admin.Email,
	}
	userInfo := auth.UserInfo{
		Username: req.User.Username,
		Password: req.User.Password,
		Email:    req.User.Email,
	}
	if err := authService.InitSystemWithUsers(adminInfo, userInfo); err != nil {
		c.JSON(http.StatusInternalServerError, common.Error(err.Error()))
		return
	}

	// 初始化系统种子数据（角色、菜单、API、公告等）
	source.InitSeedData()

	// 初始化系统镜像
	source.SeedSystemImages()

	c.JSON(http.StatusOK, common.Success("系统初始化成功"))
}

// GetRegisterConfig 获取注册配置信息
// @Summary 获取注册配置信息
// @Description 获取注册页面所需的配置信息（不需要认证）
// @Tags 系统初始化
// @Accept json
// @Produce json
// @Success 200 {object} common.Response{data=object} "获取成功"
// @Router /public/register-config [get]
func GetRegisterConfig(c *gin.Context) {
	config := map[string]interface{}{
		"auth": map[string]interface{}{
			"enablePublicRegistration": global.APP_CONFIG.Auth.EnablePublicRegistration,
		},
		"inviteCode": map[string]interface{}{
			"enabled": global.APP_CONFIG.InviteCode.Enabled,
		},
	}
	c.JSON(http.StatusOK, common.Success(config))
}
