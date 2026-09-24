package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hyperits/gosuite/logger"

	"github.com/castorworks/castor/internal/infrastructure/bootstrap"
	"github.com/castorworks/castor/internal/infrastructure/config"
	"github.com/castorworks/castor/internal/infrastructure/database"
	infraI18n "github.com/castorworks/castor/internal/infrastructure/i18n"
)

func main() {
	// 加载配置
	configPath := os.Getenv("CASTOR_CONFIG_FILE")
	if configPath == "" {
		configPath = "/etc/castor/config.toml"
	}
	config.MustLoad(configPath)

	// 初始化日志
	logger.Init(config.C.Logger.ToLoggerConfig())

	if len(os.Args) > 1 {
		if len(os.Args) != 2 || os.Args[1] != "init-db" {
			logger.Errorf("%s", infraI18n.CLIMessage("DeployCommandUsage"))
			os.Exit(2)
		}
		os.Exit(runInitDB())
	}

	// 初始化服务
	server, err := InitServer()
	if err != nil {
		logger.Fatalf("Error init server: %v", err.Error())
		return
	}

	server.Start()
}

func runInitDB() int {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	db, err := database.NewPostgres()
	if err != nil {
		logger.Errorf("%s: %v", infraI18n.CLIMessage("DeployInitFailed"), err)
		return 1
	}
	defer func() {
		if closeErr := database.Close(db); closeErr != nil {
			logger.Warnf("close database: %v", closeErr)
		}
	}()
	result, err := bootstrap.Initialize(ctx, db)
	if err != nil {
		logger.Errorf("%s: %v", infraI18n.CLIMessage("DeployInitFailed"), err)
		return 1
	}
	if result.GeneratedAdminPassword != "" {
		// Written straight to the terminal instead of the structured log so the secret
		// does not end up in log files or log aggregation.
		fmt.Fprintf(os.Stderr, "\n%s\n  username: system\n  password: %s\n\n",
			infraI18n.CLIMessage("DeployGeneratedAdminPassword"), result.GeneratedAdminPassword)
	}
	logger.Infof("%s", infraI18n.CLIMessage("DeployInitComplete"))
	return 0
}
