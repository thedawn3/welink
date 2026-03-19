package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// Config 是 WeLink 的完整配置结构体。
// 配置优先级：默认值 < config.yaml < 环境变量。
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Data     DataConfig     `yaml:"data"`
	Analysis AnalysisConfig `yaml:"analysis"`
}

type ServerConfig struct {
	// Port HTTP 监听端口，默认 8080。
	// 优先读取 WELINK_BACKEND_PORT，PORT 仅做向后兼容。
	Port string `yaml:"port"`
}

type DataConfig struct {
	// Dir 解密后的微信数据目录，默认 ./decrypted（本地开发）或 /app/data（Docker）。
	// 优先读取 WELINK_DATA_DIR，DATA_DIR 仅做向后兼容。
	Dir string `yaml:"dir"`

	// MsgDir 微信媒体资源目录，包含图片/视频/文件等大体积资源。
	// 为空时表示不挂载媒体资源访问。
	MsgDir string `yaml:"msg_dir"`
}

type AnalysisConfig struct {
	// Timezone 时区名称，默认 Asia/Shanghai（即 CST UTC+8）。
	// 影响消息时间的小时/星期分布统计。
	Timezone string `yaml:"timezone"`

	// LateNightStartHour 深夜开始小时（含），默认 0（即 0:00 起）。
	LateNightStartHour int `yaml:"late_night_start_hour"`

	// LateNightEndHour 深夜结束小时（不含），默认 5（即到 4:59 止）。
	LateNightEndHour int `yaml:"late_night_end_hour"`

	// SessionGapSeconds 判断新对话段的时间间隔（秒），默认 21600（6 小时）。
	SessionGapSeconds int64 `yaml:"session_gap_seconds"`

	// WorkerCount 并发分析联系人的 goroutine 数，默认 4。
	WorkerCount int `yaml:"worker_count"`

	// LateNightMinMessages 进入深夜排行榜所需的最少消息数，默认 100。
	LateNightMinMessages int64 `yaml:"late_night_min_messages"`

	// LateNightTopN 深夜排行榜保留前 N 名，默认 20。
	LateNightTopN int `yaml:"late_night_top_n"`

	// DefaultInitFrom 启动后自动初始化的开始时间（Unix 秒，0 表示不限）。
	// 设置此值后，服务启动即自动开始索引，无需前端手动点击"开始分析"。
	DefaultInitFrom int64 `yaml:"default_init_from"`

	// DefaultInitTo 启动后自动初始化的结束时间（Unix 秒，0 表示不限）。
	DefaultInitTo int64 `yaml:"default_init_to"`
}

// defaults 返回所有字段的默认值。
func defaults() Config {
	return Config{
		Server: ServerConfig{
			Port: "8080",
		},
		Data: DataConfig{
			Dir:    "./decrypted",
			MsgDir: "",
		},
		Analysis: AnalysisConfig{
			Timezone:             "Asia/Shanghai",
			LateNightStartHour:   0,
			LateNightEndHour:     5,
			SessionGapSeconds:    21600,
			WorkerCount:          4,
			LateNightMinMessages: 100,
			LateNightTopN:        20,
			DefaultInitFrom:      0,
			DefaultInitTo:        0,
		},
	}
}

// Load 加载配置，按以下优先级合并：
//  1. 默认值
//  2. config.yaml（若存在）
//  3. 环境变量 WELINK_*（旧 DATA_DIR / MSG_DIR / PORT 仅向后兼容）
func Load(configPath string) *Config {
	cfg := defaults()

	// 尝试加载 YAML 文件
	if configPath == "" {
		configPath = "config.yaml"
	}
	if data, err := os.ReadFile(configPath); err == nil {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			log.Printf("[CONFIG] Failed to parse %s: %v, using defaults", configPath, err)
		} else {
			log.Printf("[CONFIG] Loaded config from %s", configPath)
		}
	} else if !os.IsNotExist(err) {
		log.Printf("[CONFIG] Cannot read %s: %v, using defaults", configPath, err)
	}

	// 环境变量覆盖（向后兼容旧用法）
	if v := firstNonEmptyEnv("WELINK_DATA_DIR", "DATA_DIR"); v != "" {
		cfg.Data.Dir = v
	}
	if v := firstNonEmptyEnv("WELINK_MSG_DIR", "MSG_DIR"); v != "" {
		cfg.Data.MsgDir = v
	}
	if v := firstNonEmptyEnv("WELINK_BACKEND_PORT", "PORT"); v != "" {
		cfg.Server.Port = v
	}

	return &cfg
}

func firstNonEmptyEnv(keys ...string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return ""
}
