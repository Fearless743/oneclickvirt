package config

type UpdateConfigRequest struct {
	Auth       AuthConfig       `json:"auth"`
	Quota      QuotaConfig      `json:"quota"`
	InviteCode InviteCodeConfig `json:"inviteCode"`
}

type AuthConfig struct {
	EnableEmail              bool   `json:"enableEmail"`
	EnableTelegram           bool   `json:"enableTelegram"`
	EnableQQ                 bool   `json:"enableQQ"`
	EnablePublicRegistration bool   `json:"enablePublicRegistration"` // 是否启用公开注册（无需邀请码）
	EmailSMTPHost            string `json:"emailSMTPHost"`
	EmailSMTPPort            int    `json:"emailSMTPPort"`
	EmailUsername            string `json:"emailUsername"`
	EmailPassword            string `json:"emailPassword"`
	TelegramBotToken         string `json:"telegramBotToken"`
	QQAppID                  string `json:"qqAppID"`
	QQAppKey                 string `json:"qqAppKey"`
}

type InviteCodeConfig struct {
	Enabled        bool `json:"enabled"`        // 是否启用邀请码系统
	Required       bool `json:"required"`       // 是否必须邀请码（兼容旧字段）
	DefaultMaxUses int  `json:"defaultMaxUses"` // 默认最大使用次数
}

type QuotaConfig struct {
	DefaultLevel int                    `json:"defaultLevel"`
	LevelLimits  map[int]LevelLimitInfo `json:"levelLimits"`
}

type LevelLimitInfo struct {
	MaxInstances int                    `json:"maxInstances"`
	MaxResources map[string]interface{} `json:"maxResources"`
	MaxTraffic   int64                  `json:"maxTraffic"` // 最大流量限制(MB)
}

// DatabaseConfig 数据库初始化配置
type DatabaseConfig struct {
	Type         string `json:"type" binding:"required"`
	Host         string `json:"host" binding:"required"`
	Port         int    `json:"port" binding:"required,min=1,max=65535"`
	Username     string `json:"username" binding:"required"`
	Password     string `json:"password"`
	Database     string `json:"database" binding:"required"`
	DatabaseName string `json:"databaseName"`
	SSLMode      string `json:"sslMode"`
}

// ConfigTaskContext 配置任务上下文
type ConfigTaskContext struct {
	RequestID    string                 `json:"requestId"`
	UserID       uint                   `json:"userId"`
	OriginalData map[string]interface{} `json:"originalData"`
	TargetData   map[string]interface{} `json:"targetData"`
}

// DailyLogConfig 日志轮转配置
type DailyLogConfig struct {
	BaseDir    string `json:"baseDir"`    // 基础日志目录
	FileName   string `json:"fileName"`   // 文件名
	MaxSize    int64  `json:"maxSize"`    // 最大文件大小(字节)
	MaxAge     int    `json:"maxAge"`     // 保留天数
	MaxBackups int    `json:"maxBackups"` // 最大备份数
	Compress   bool   `json:"compress"`   // 是否压缩历史日志
	LocalTime  bool   `json:"localTime"`  // 使用本地时间
}
