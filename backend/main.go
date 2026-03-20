/*
 * WeLink — 微信聊天数据分析平台
 * Copyright (C) 2026 runzhliu
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"welink/backend/config"
	"welink/backend/pkg/db"
	"welink/backend/pkg/seed"
	runtimepkg "welink/backend/runtime"
	"welink/backend/service"
	syncmgr "welink/backend/sync"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. 加载配置（默认值 < config.yaml < 环境变量）
	cfg := config.Load(resolveConfigPath())
	log.Printf("WeLink config: data_dir=%s msg_dir=%s port=%s timezone=%s workers=%d",
		cfg.Data.Dir, cfg.Data.MsgDir, cfg.Server.Port, cfg.Analysis.Timezone, cfg.Analysis.WorkerCount)

	analysisDir := cfg.Data.Dir
	if cfg.Ingest.AnalysisDataDir != "" {
		analysisDir = cfg.Ingest.AnalysisDataDir
	}

	// 2. 初始化数据库管理器（WELINK_DEMO_MODE / DEMO_MODE 时先生成示例数据）
	if isDemoMode() {
		demoDir := analysisDir
		log.Printf("[DEMO] Demo mode enabled, generating sample databases in %s", demoDir)
		if err := seed.Generate(demoDir); err != nil {
			log.Fatalf("Failed to generate demo databases: %v", err)
		}
	}

	dbMgr, err := db.NewDBManager(analysisDir)
	if err != nil {
		log.Fatalf("Init DB failed: %v", err)
	}

	// 3. 初始化服务层
	systemRT := newSystemRuntime(cfg)
	contactSvc := service.NewContactService(dbMgr, cfg)
	systemRT.reindex = func(from, to int64) {
		contactSvc.Reinitialize(from, to)
	}
	refreshAnalysis := func(stageSource bool, reason string) error {
		if stageSource {
			if err := systemRT.stageSourceSnapshot(); err != nil {
				systemRT.store.UpdateStatus(func(status *runtimepkg.RuntimeStatus) {
					status.LastError = err.Error()
				})
				return err
			}
		}
		if err := dbMgr.Reload(analysisDir); err != nil {
			systemRT.store.AppendLog("error", "db", err.Error(), nil)
			systemRT.store.UpdateStatus(func(status *runtimepkg.RuntimeStatus) {
				status.LastError = err.Error()
			})
			return err
		}
		systemRT.store.AppendLog("info", "analysis", "refresh scheduled: "+reason, nil)
		from, to := contactSvc.GetFilterRange()
		contactSvc.Reinitialize(from, to)
		return nil
	}
	systemRT.onSnapshotReady = func(reason string) error {
		return refreshAnalysis(false, reason)
	}
	systemRT.applyInitialAnalysisStatus(false, false, len(contactSvc.GetCachedStats()))
	reindexStart, reindexFinish := systemRT.bindReindexHooks(func() int { return len(contactSvc.GetCachedStats()) })
	contactSvc.SetReindexHooks(reindexStart, reindexFinish)

	syncRoot := analysisDir
	if systemRT.shouldStageSourceBeforeReindex() {
		syncRoot = cfg.Ingest.SourceDataDir
	}

	manager, syncErr := syncmgr.NewManager(syncmgr.ManagerOptions{
		Root:         syncRoot,
		Debounce:     time.Duration(cfg.Sync.DebounceMs) * time.Millisecond,
		PollInterval: time.Duration(maxInt(cfg.Sync.MaxWaitMs/5, 1000)) * time.Millisecond,
		WatchWAL:     cfg.Sync.WatchWAL,
		OnRevision: func(revision syncmgr.Revision) {
			systemRT.onDataRevision(revision, func() {
				if err := refreshAnalysis(systemRT.shouldStageSourceBeforeReindex(), "sync revision "+revision.ID); err != nil {
					systemRT.publishEvent("runtime.reindex.failed", map[string]any{
						"revision": revision.ID,
						"error":    err.Error(),
						"message":  err.Error(),
					})
				}
			})
		},
		OnError: func(syncErr error) {
			systemRT.store.AppendLog("error", "sync", syncErr.Error(), nil)
		},
	})
	if syncErr != nil {
		log.Printf("Failed to initialize sync manager: %v", syncErr)
	} else {
		systemRT.syncManager = manager
		if cfg.Sync.Enabled {
			if err := manager.Start(); err != nil {
				log.Printf("Failed to start sync manager: %v", err)
			}
		}
	}

	systemRT.startConfiguredDecrypt()

	// 4. 初始化 Gin 路由
	r := gin.Default()

	if cfg.Data.MsgDir != "" {
		if st, err := os.Stat(cfg.Data.MsgDir); err == nil && st.IsDir() {
			log.Printf("Serving WeChat media files from %s at /media", cfg.Data.MsgDir)
			r.StaticFS("/media", http.Dir(cfg.Data.MsgDir))
		} else if err != nil {
			log.Printf("Media dir unavailable: %s (%v)", cfg.Data.MsgDir, err)
		}
	}

	// 跨域设置
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	api := r.Group("/api")
	{
		systemRT.registerRoutes(api)
		withAnalysisData := func(handler gin.HandlerFunc) gin.HandlerFunc {
			return func(c *gin.Context) {
				if !dbMgr.Ready() {
					c.JSON(http.StatusServiceUnavailable, gin.H{"error": "analysis data not ready"})
					return
				}
				handler(c)
			}
		}

		// 极速获取缓存后的统计信息
		api.GET("/contacts/stats", withAnalysisData(func(c *gin.Context) {
			stats := contactSvc.GetCachedStats()
			c.JSON(http.StatusOK, stats)
		}))

		// 获取全局统计数据
		api.GET("/global", withAnalysisData(func(c *gin.Context) {
			c.JSON(http.StatusOK, contactSvc.GetGlobal())
		}))

		// 获取词云数据
		api.GET("/contacts/wordcloud", withAnalysisData(func(c *gin.Context) {
			uname := c.Query("username")
			if uname == "" {
				c.JSON(400, gin.H{"error": "username required"})
				return
			}
			includeMine := c.Query("include_mine") == "true"
			c.JSON(http.StatusOK, contactSvc.GetWordCloud(uname, includeMine))
		}))

		// 群聊列表
		api.GET("/groups", withAnalysisData(func(c *gin.Context) {
			c.JSON(http.StatusOK, contactSvc.GetGroups())
		}))

		// 群聊某天聊天记录
		api.GET("/groups/messages", withAnalysisData(func(c *gin.Context) {
			uname := c.Query("username")
			date := c.Query("date")
			if uname == "" || date == "" {
				c.JSON(400, gin.H{"error": "username and date required"})
				return
			}
			c.JSON(http.StatusOK, contactSvc.GetGroupDayMessages(uname, date))
		}))

		// 群聊深度画像
		api.GET("/groups/detail", withAnalysisData(func(c *gin.Context) {
			uname := c.Query("username")
			if uname == "" {
				c.JSON(400, gin.H{"error": "username required"})
				return
			}
			c.JSON(http.StatusOK, contactSvc.GetGroupDetail(uname))
		}))

		// 获取与联系人的共同群聊
		api.GET("/contacts/common-groups", withAnalysisData(func(c *gin.Context) {
			uname := c.Query("username")
			if uname == "" {
				c.JSON(400, gin.H{"error": "username required"})
				return
			}
			c.JSON(http.StatusOK, contactSvc.GetCommonGroups(uname))
		}))

		// 获取联系人深度分析（小时/周/日历/深夜/红包/主动率）
		api.GET("/contacts/detail", withAnalysisData(func(c *gin.Context) {
			uname := c.Query("username")
			if uname == "" {
				c.JSON(400, gin.H{"error": "username required"})
				return
			}
			c.JSON(http.StatusOK, contactSvc.GetContactDetail(uname))
		}))

		// 关系变化榜
		api.GET("/relations/overview", withAnalysisData(func(c *gin.Context) {
			c.JSON(http.StatusOK, contactSvc.GetRelationOverview())
		}))

		// 联系人关系档案
		api.GET("/relations/detail", withAnalysisData(func(c *gin.Context) {
			uname := c.Query("username")
			if uname == "" {
				c.JSON(400, gin.H{"error": "username required"})
				return
			}
			c.JSON(http.StatusOK, contactSvc.GetRelationDetail(uname))
		}))

		// 争议榜单
		api.GET("/controversy/overview", withAnalysisData(func(c *gin.Context) {
			c.JSON(http.StatusOK, contactSvc.GetControversyOverview())
		}))

		// 联系人争议详情
		api.GET("/controversy/detail", withAnalysisData(func(c *gin.Context) {
			uname := c.Query("username")
			if uname == "" {
				c.JSON(400, gin.H{"error": "username required"})
				return
			}
			c.JSON(http.StatusOK, contactSvc.GetControversyDetail(uname))
		}))

		// 某天的聊天记录（日历点击）
		api.GET("/contacts/messages", withAnalysisData(func(c *gin.Context) {
			uname := c.Query("username")
			date := c.Query("date") // "2024-03-15"
			if uname == "" || date == "" {
				c.JSON(400, gin.H{"error": "username and date required"})
				return
			}
			c.JSON(http.StatusOK, contactSvc.GetDayMessages(uname, date))
		}))

		// 联系人历史聊天时间线
		api.GET("/contacts/messages/history", withAnalysisData(func(c *gin.Context) {
			uname := c.Query("username")
			if uname == "" {
				c.JSON(400, gin.H{"error": "username required"})
				return
			}

			before := int64(0)
			if value := c.Query("before"); value != "" {
				if _, err := fmt.Sscanf(value, "%d", &before); err != nil {
					c.JSON(400, gin.H{"error": "before must be unix timestamp"})
					return
				}
			}

			limit := 200
			if value := c.Query("limit"); value != "" {
				if _, err := fmt.Sscanf(value, "%d", &limit); err != nil {
					c.JSON(400, gin.H{"error": "limit must be integer"})
					return
				}
			}
			c.JSON(http.StatusOK, contactSvc.GetMessageHistory(uname, before, limit))
		}))

		// 搜索联系人聊天记录
		api.GET("/contacts/search", withAnalysisData(func(c *gin.Context) {
			uname := c.Query("username")
			q := c.Query("q")
			if uname == "" || q == "" {
				c.JSON(400, gin.H{"error": "username and q required"})
				return
			}
			includeMine := c.Query("include_mine") == "true"
			c.JSON(http.StatusOK, contactSvc.SearchMessages(uname, q, includeMine))
		}))

		// 全量搜索聊天记录
		api.GET("/search/messages", withAnalysisData(func(c *gin.Context) {
			q := c.Query("q")
			if q == "" {
				c.JSON(400, gin.H{"error": "q required"})
				return
			}
			includeMine := c.Query("include_mine") == "true"
			limit := 200
			if _, err := fmt.Sscanf(c.DefaultQuery("limit", "200"), "%d", &limit); err != nil {
				limit = 200
			}
			c.JSON(http.StatusOK, contactSvc.SearchAllMessages(q, includeMine, limit))
		}))

		// 朋友圈查询（发帖 + 互动 + 索引）
		api.GET("/sns/search", withAnalysisData(func(c *gin.Context) {
			limit := 100
			if value := c.Query("limit"); value != "" {
				if _, err := fmt.Sscanf(value, "%d", &limit); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be integer"})
					return
				}
			}
			resp, err := contactSvc.SearchSNS(service.SnsSearchParams{
				Q:        c.Query("q"),
				Username: c.Query("username"),
				Kind:     c.DefaultQuery("kind", "all"),
				From:     c.Query("from"),
				To:       c.Query("to"),
				Limit:    limit,
			})
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, resp)
		}))

		api.GET("/media/chat-image", withAnalysisData(func(c *gin.Context) {
			username := c.Query("username")
			md5Value := c.Query("md5")
			size := c.DefaultQuery("size", "full")
			var ts int64
			if username == "" || md5Value == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "username and md5 required"})
				return
			}
			if _, err := fmt.Sscanf(c.Query("ts"), "%d", &ts); err != nil || ts <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "ts must be unix timestamp"})
				return
			}
			path, contentType, err := contactSvc.ResolveChatImage(username, ts, md5Value, size)
			if err != nil {
				switch {
				case errors.Is(err, service.ServiceErrImageNotFound):
					c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				case errors.Is(err, service.ServiceErrImageNeedsAES):
					c.JSON(http.StatusFailedDependency, gin.H{"error": err.Error()})
				default:
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				}
				return
			}
			c.Header("Content-Type", contentType)
			c.File(path)
		}))

		// 某月的文本消息（情感分析详情）
		api.GET("/contacts/messages/month", withAnalysisData(func(c *gin.Context) {
			uname := c.Query("username")
			month := c.Query("month") // "2024-03"
			if uname == "" || month == "" {
				c.JSON(400, gin.H{"error": "username and month required"})
				return
			}
			includeMine := c.Query("include_mine") == "true"
			c.JSON(http.StatusOK, contactSvc.GetMonthMessages(uname, month, includeMine))
		}))

		// 情感分析
		api.GET("/contacts/sentiment", withAnalysisData(func(c *gin.Context) {
			uname := c.Query("username")
			if uname == "" {
				c.JSON(400, gin.H{"error": "username required"})
				return
			}
			includeMine := c.Query("include_mine") == "true"
			c.JSON(http.StatusOK, contactSvc.GetSentimentAnalysis(uname, includeMine))
		}))

		// 时间范围过滤统计（from/to 为 Unix 秒时间戳）
		api.GET("/stats/filter", withAnalysisData(func(c *gin.Context) {
			var from, to int64
			fmt.Sscanf(c.Query("from"), "%d", &from)
			fmt.Sscanf(c.Query("to"), "%d", &to)
			result := contactSvc.AnalyzeWithFilter(from, to)
			if result == nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "analysis failed"})
				return
			}
			c.JSON(http.StatusOK, result)
		}))

		// 获取数据库管理信息
		api.GET("/databases", func(c *gin.Context) {
			c.JSON(http.StatusOK, dbMgr.GetDBInfos())
		})

		// 获取指定数据库的表列表
		api.GET("/databases/:dbName/tables", func(c *gin.Context) {
			dbName := c.Param("dbName")
			tables, err := dbMgr.GetTables(dbName)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, tables)
		})

		// 获取表结构
		api.GET("/databases/:dbName/tables/:tableName/schema", func(c *gin.Context) {
			dbName := c.Param("dbName")
			tableName := c.Param("tableName")
			cols, err := dbMgr.GetTableSchema(dbName, tableName)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, cols)
		})

		// 获取表数据（分页）
		api.GET("/databases/:dbName/tables/:tableName/data", func(c *gin.Context) {
			dbName := c.Param("dbName")
			tableName := c.Param("tableName")
			offset := 0
			limit := 50
			if v := c.Query("offset"); v != "" {
				fmt.Sscanf(v, "%d", &offset)
			}
			if v := c.Query("limit"); v != "" {
				fmt.Sscanf(v, "%d", &limit)
				if limit > 200 {
					limit = 200
				}
			}
			data, err := dbMgr.GetTableData(dbName, tableName, offset, limit)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, data)
		})

		// 初始化/重新索引（前端传入时间范围后调用）
		api.POST("/init", func(c *gin.Context) {
			var body struct {
				From int64 `json:"from"`
				To   int64 `json:"to"`
			}
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(400, gin.H{"error": "invalid body"})
				return
			}
			contactSvc.Reinitialize(body.From, body.To)
			c.JSON(200, gin.H{"status": "indexing"})
		})

		// 获取后端索引状态
		api.GET("/status", func(c *gin.Context) {
			c.JSON(200, systemRT.store.Status())
		})

		// 健康检查
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok", "db_connected": len(dbMgr.MessageDBs), "analysis_dir": analysisDir})
		})

		registerExportRoutes(api, contactSvc, func() bool { return dbMgr.Ready() })

		// OpenAPI 规范
		api.GET("/swagger.json", func(c *gin.Context) {
			c.Data(http.StatusOK, "application/json", swaggerSpec())
		})
	}

	// Swagger UI
	r.GET("/swagger", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/")
	})
	r.GET("/swagger/", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", swaggerUI())
	})

	log.Printf("WeLink Backend serving on :%s", cfg.Server.Port)
	r.Run(":" + cfg.Server.Port)
}

func resolveConfigPath() string {
	if path := os.Getenv("WELINK_CONFIG_PATH"); path != "" {
		return path
	}
	for _, candidate := range []string{"config.yaml", "../config.yaml"} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func isDemoMode() bool {
	return os.Getenv("WELINK_DEMO_MODE") == "true" || os.Getenv("DEMO_MODE") == "true"
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
