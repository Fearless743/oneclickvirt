package user

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	// 基础字段
	ID        uint           `json:"id" gorm:"primarykey"`                     // 用户主键ID
	UUID      string         `json:"uuid" gorm:"uniqueIndex;not null;size:36"` // 用户唯一标识符
	CreatedAt time.Time      `json:"createdAt"`                                // 用户创建时间
	UpdatedAt time.Time      `json:"updatedAt"`                                // 用户信息更新时间
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`                           // 软删除时间

	// 基本信息
	Username string `json:"username" gorm:"uniqueIndex;not null;size:64"` // 用户名（唯一，用于登录）
	Password string `json:"-" gorm:"not null;size:128"`                   // 密码哈希（不返回给前端）
	Nickname string `json:"nickname" gorm:"size:64"`                      // 用户昵称（显示名称）
	Email    string `json:"email" gorm:"size:128"`                        // 邮箱地址
	Phone    string `json:"phone" gorm:"size:32"`                         // 手机号码
	Telegram string `json:"telegram" gorm:"size:64"`                      // Telegram用户名
	QQ       string `json:"qq" gorm:"size:32"`                            // QQ号码
	Avatar   string `json:"avatar" gorm:"size:255"`                       // 头像图片路径

	// 状态和权限
	Status   int    `json:"status" gorm:"default:1"`              // 用户状态：0=禁用（不可登录），1=正常
	Level    int    `json:"level" gorm:"default:1"`               // 用户等级，用于权限控制
	UserType string `json:"userType" gorm:"default:user;size:16"` // 用户类型：user, admin, super_admin等

	// 配额管理（传统系统兼容字段）
	UsedQuota  int `json:"usedQuota" gorm:"default:0"`  // 已使用配额
	TotalQuota int `json:"totalQuota" gorm:"default:0"` // 总配额限制

	// 流量管理（MB为单位）
	UsedTraffic    int64      `json:"usedTraffic" gorm:"default:0"`        // 当月已使用流量（MB）
	TotalTraffic   int64      `json:"totalTraffic" gorm:"default:0"`       // 当月流量配额（MB），根据用户等级自动设置
	TrafficResetAt *time.Time `json:"trafficResetAt"`                      // 流量重置时间
	TrafficLimited bool       `json:"trafficLimited" gorm:"default:false"` // 是否因流量超限被限制

	// 资源限制（根据用户等级自动设置，避免每次查询配置）
	MaxInstances int `json:"maxInstances" gorm:"default:1"`   // 最大实例数
	MaxCPU       int `json:"maxCPU" gorm:"default:1"`         // 最大CPU核心数
	MaxMemory    int `json:"maxMemory" gorm:"default:512"`    // 最大内存（MB）
	MaxDisk      int `json:"maxDisk" gorm:"default:10240"`    // 最大磁盘空间（MB）
	MaxBandwidth int `json:"maxBandwidth" gorm:"default:100"` // 最大带宽（Mbps）

	// 其他信息
	InviteCode  string     `json:"inviteCode" gorm:"size:32"` // 注册时使用的邀请码
	LastLoginAt *time.Time `json:"lastLoginAt"`               // 最后登录时间
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	u.UUID = uuid.New().String()
	return nil
}

// UserRole 用户角色关联表
type UserRole struct {
	UserID uint `gorm:"primarykey" json:"user_id"`
	RoleID uint `gorm:"primarykey" json:"role_id"`
}

// VerifyCode 验证码模型
type VerifyCode struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	Email     string    `json:"email" gorm:"size:100"`
	Phone     string    `json:"phone" gorm:"size:20"`
	Target    string    `json:"target" gorm:"size:128"`
	Code      string    `json:"code" gorm:"size:10;not null"`
	Type      string    `json:"type" gorm:"size:20;not null"`
	Used      bool      `json:"used" gorm:"default:false"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
}

// PasswordReset 密码重置模型
type PasswordReset struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	UserUUID  string    `json:"user_uuid" gorm:"size:36;not null"`
	Token     string    `json:"token" gorm:"size:64;not null;uniqueIndex"`
	Used      bool      `json:"used" gorm:"default:false"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
}

// TrafficRecord 流量记录表 - 用于跟踪用户在Provider上的流量使用
type TrafficRecord struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// 关联信息
	UserID     uint `json:"userId" gorm:"not null;index"`     // 用户ID
	ProviderID uint `json:"providerId" gorm:"not null;index"` // Provider ID
	InstanceID uint `json:"instanceId" gorm:"index"`          // 实例ID（可能为空，因为实例可能被删除）

	// 时间范围
	Year  int `json:"year" gorm:"not null;index"`  // 年份
	Month int `json:"month" gorm:"not null;index"` // 月份

	// 流量统计（MB为单位）
	TrafficIn  int64 `json:"trafficIn" gorm:"default:0"`  // 入站流量（MB）
	TrafficOut int64 `json:"trafficOut" gorm:"default:0"` // 出站流量（MB）
	TotalUsed  int64 `json:"totalUsed" gorm:"default:0"`  // 总使用流量（MB）

	// vnstat相关
	InterfaceName string     `json:"interfaceName" gorm:"size:32"` // 网络接口名称
	LastSyncAt    *time.Time `json:"lastSyncAt"`                   // 最后同步时间
}
