package config

import (
	"log"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config 是 WeLink 的完整配置结构体。
// 配置优先级：默认值 < config.yaml < 环境变量。
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Data     DataConfig     `yaml:"data"`
	Analysis AnalysisConfig `yaml:"analysis"`
	Ingest   IngestConfig   `yaml:"ingest"`
	Sync     SyncConfig     `yaml:"sync"`
	Decrypt  DecryptConfig  `yaml:"decrypt"`
	Runtime  RuntimeConfig  `yaml:"runtime"`
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

type IngestConfig struct {
	// Enabled 控制统一数据接入编排是否启用，默认 false（保持兼容原有纯分析模式）。
	Enabled bool `yaml:"enabled"`

	// SourceDataDir 原始数据源目录（待解密或上游同步目录）。
	SourceDataDir string `yaml:"source_data_dir"`

	// WorkDir 运行时工作目录（解密产物、临时文件等）。
	WorkDir string `yaml:"work_dir"`

	// AnalysisDataDir 分析读取目录。默认继承 Data.Dir。
	AnalysisDataDir string `yaml:"analysis_data_dir"`

	// Platform 运行平台标识（auto/windows/macos/linux）。
	Platform string `yaml:"platform"`
}

type SyncConfig struct {
	// Enabled 控制文件监听/自动刷新是否启用。
	Enabled bool `yaml:"enabled"`

	// WatchWAL 是否监听 wal/shm 变更。
	WatchWAL bool `yaml:"watch_wal"`

	// DebounceMs 变更去抖窗口（毫秒）。
	DebounceMs int `yaml:"debounce_ms"`

	// MaxWaitMs 批次变更最大等待时间（毫秒）。
	MaxWaitMs int `yaml:"max_wait_ms"`

	// EventBuffer SSE/事件订阅缓冲大小。
	EventBuffer int `yaml:"event_buffer"`
}

type DecryptConfig struct {
	// Enabled 控制解密能力是否启用。
	Enabled bool `yaml:"enabled"`

	// AutoStart 服务启动后是否自动开启解密任务。
	AutoStart bool `yaml:"auto_start"`

	// Provider 解密提供方（builtin/external）。
	Provider string `yaml:"provider"`

	// WindowsCommand/ MacCommand/ LinuxCommand 为平台默认解密命令。
	WindowsCommand string `yaml:"windows_command"`
	MacCommand     string `yaml:"mac_command"`
	LinuxCommand   string `yaml:"linux_command"`

	// PreserveWAL 解密输出后是否保留 wal/shm。
	PreserveWAL bool `yaml:"preserve_wal"`

	// TaskTimeoutSeconds 单次解密任务超时时间（秒）。
	TaskTimeoutSeconds int `yaml:"task_timeout_seconds"`
}

type RuntimeConfig struct {
	// EngineType 运行引擎类型（welink/windows/macos/hybrid）。
	EngineType string `yaml:"engine_type"`

	// MaxTaskRecords 运行时最多保留任务记录数。
	MaxTaskRecords int `yaml:"max_task_records"`

	// MaxLogRecords 运行时最多保留日志记录数。
	MaxLogRecords int `yaml:"max_log_records"`
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
		Ingest: IngestConfig{
			Enabled:         false,
			SourceDataDir:   "",
			WorkDir:         "./workdir",
			AnalysisDataDir: "",
			Platform:        "auto",
		},
		Sync: SyncConfig{
			Enabled:     false,
			WatchWAL:    true,
			DebounceMs:  1000,
			MaxWaitMs:   10000,
			EventBuffer: 128,
		},
		Decrypt: DecryptConfig{
			Enabled:            false,
			AutoStart:          false,
			Provider:           "builtin",
			WindowsCommand:     "",
			MacCommand:         "",
			LinuxCommand:       "",
			PreserveWAL:        false,
			TaskTimeoutSeconds: 120,
		},
		Runtime: RuntimeConfig{
			EngineType:     "welink",
			MaxTaskRecords: 200,
			MaxLogRecords:  1000,
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
	if v := firstNonEmptyEnv("WELINK_INGEST_ENABLED"); v != "" {
		cfg.Ingest.Enabled = parseBool(v, cfg.Ingest.Enabled)
	}
	if v := firstNonEmptyEnv("WELINK_SOURCE_DATA_DIR"); v != "" {
		cfg.Ingest.SourceDataDir = v
	}
	if v := firstNonEmptyEnv("WELINK_WORK_DIR"); v != "" {
		cfg.Ingest.WorkDir = v
	}
	if v := firstNonEmptyEnv("WELINK_ANALYSIS_DATA_DIR"); v != "" {
		cfg.Ingest.AnalysisDataDir = v
	}
	if v := firstNonEmptyEnv("WELINK_PLATFORM"); v != "" {
		cfg.Ingest.Platform = v
	}
	if v := firstNonEmptyEnv("WELINK_SYNC_ENABLED"); v != "" {
		cfg.Sync.Enabled = parseBool(v, cfg.Sync.Enabled)
	}
	if v := firstNonEmptyEnv("WELINK_SYNC_WATCH_WAL"); v != "" {
		cfg.Sync.WatchWAL = parseBool(v, cfg.Sync.WatchWAL)
	}
	if v := firstNonEmptyEnv("WELINK_SYNC_DEBOUNCE_MS"); v != "" {
		cfg.Sync.DebounceMs = parseInt(v, cfg.Sync.DebounceMs)
	}
	if v := firstNonEmptyEnv("WELINK_SYNC_MAX_WAIT_MS"); v != "" {
		cfg.Sync.MaxWaitMs = parseInt(v, cfg.Sync.MaxWaitMs)
	}
	if v := firstNonEmptyEnv("WELINK_SYNC_EVENT_BUFFER"); v != "" {
		cfg.Sync.EventBuffer = parseInt(v, cfg.Sync.EventBuffer)
	}
	if v := firstNonEmptyEnv("WELINK_DECRYPT_ENABLED"); v != "" {
		cfg.Decrypt.Enabled = parseBool(v, cfg.Decrypt.Enabled)
	}
	if v := firstNonEmptyEnv("WELINK_DECRYPT_AUTO_START"); v != "" {
		cfg.Decrypt.AutoStart = parseBool(v, cfg.Decrypt.AutoStart)
	}
	if v := firstNonEmptyEnv("WELINK_DECRYPT_PROVIDER"); v != "" {
		cfg.Decrypt.Provider = v
	}
	if v := firstNonEmptyEnv("WELINK_WINDOWS_DECRYPT_COMMAND"); v != "" {
		cfg.Decrypt.WindowsCommand = v
	}
	if v := firstNonEmptyEnv("WELINK_MAC_DECRYPT_COMMAND"); v != "" {
		cfg.Decrypt.MacCommand = v
	}
	if v := firstNonEmptyEnv("WELINK_LINUX_DECRYPT_COMMAND"); v != "" {
		cfg.Decrypt.LinuxCommand = v
	}
	if v := firstNonEmptyEnv("WELINK_DECRYPT_PRESERVE_WAL"); v != "" {
		cfg.Decrypt.PreserveWAL = parseBool(v, cfg.Decrypt.PreserveWAL)
	}
	if v := firstNonEmptyEnv("WELINK_DECRYPT_TIMEOUT_SECONDS"); v != "" {
		cfg.Decrypt.TaskTimeoutSeconds = parseInt(v, cfg.Decrypt.TaskTimeoutSeconds)
	}
	if v := firstNonEmptyEnv("WELINK_RUNTIME_ENGINE_TYPE"); v != "" {
		cfg.Runtime.EngineType = v
	}
	if v := firstNonEmptyEnv("WELINK_RUNTIME_MAX_TASK_RECORDS"); v != "" {
		cfg.Runtime.MaxTaskRecords = parseInt(v, cfg.Runtime.MaxTaskRecords)
	}
	if v := firstNonEmptyEnv("WELINK_RUNTIME_MAX_LOG_RECORDS"); v != "" {
		cfg.Runtime.MaxLogRecords = parseInt(v, cfg.Runtime.MaxLogRecords)
	}

	if cfg.Ingest.SourceDataDir == "" {
		cfg.Ingest.SourceDataDir = cfg.Data.Dir
	}
	if cfg.Ingest.AnalysisDataDir == "" {
		cfg.Ingest.AnalysisDataDir = cfg.Data.Dir
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

func parseBool(raw string, fallback bool) bool {
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return v
}

func parseInt(raw string, fallback int) int {
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}
