package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/go-sql-driver/mysql"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
)

var (
	sqlDir = flag.String("dir", "./sql", "SQL migrations directory")
)

func main() {
	common.Init()
	logger.SetupLogger()
	flag.Parse()

	logger.SysLog("Starting database migration...")

	// 初始化数据库连接
	model.InitDB()
	defer func() {
		err := model.CloseDB()
		if err != nil {
			logger.FatalLog("failed to close database: " + err.Error())
		}
	}()

	// 执行 SQL 迁移文件
	files, err := filepath.Glob(filepath.Join(*sqlDir, "*.sql"))
	if err != nil {
		logger.FatalLog("failed to find SQL files: " + err.Error())
	}

	if len(files) == 0 {
		logger.SysLog("No SQL migration files found in " + *sqlDir)
		return
	}

	for _, file := range files {
		logger.SysLog("Executing migration: " + file)

		content, err := os.ReadFile(file)
		if err != nil {
			logger.FatalLog("failed to read SQL file: " + err.Error())
		}

		// 执行 SQL
		err = model.DB.Exec(string(content)).Error
		if err != nil {
			logger.FatalLog(fmt.Sprintf("failed to execute %s: %v", file, err))
		}

		logger.SysLog("✓ Completed: " + filepath.Base(file))
	}

	logger.SysLog("✓ All migrations completed successfully!")

	// 初始化模型定价缓存
	err = model.InitModelPricingCache()
	if err != nil {
		logger.Warn(context.Background(), "failed to init model pricing cache: "+err.Error())
	} else {
		logger.SysLog("✓ Model pricing cache initialized")
	}
}
