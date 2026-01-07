// Package main 提供了Sealock Doc内容寻址存储系统的核心功能演示
// 该系统实现了文件的分块存储、内容寻址和版本控制功能
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/spf13/viper"
	"github.com/sealock/core-storage/chunker"
	"github.com/sealock/core-storage/service"
	"github.com/sealock/core-storage/storage"
	"github.com/redis/go-redis/v9"
)

// ============ 演示配置和初始化函数 ============

// demonstrateLocalStorage 演示本地存储栈
// 包括初始化存储、创建文件服务、上传文件等操作
func demonstrateLocalStorage(ctx context.Context) error {
	log.Printf("Context: %v", ctx)
	log.Println("\n========================================")
	log.Println("演示 1: 本地存储栈（开发环境）")
	log.Println("========================================")

	// 配置存储参数
	cfg := storage.StorageConfig{
		DatabaseDSN: "host=localhost user=postgres password=postgres dbname=sealock port=5432 sslmode=disable",
		StorageType: "local",
	}

	// 初始化存储栈
	stack, err := storage.InitializeStorage(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	defer stack.Close()

	log.Printf("  存储类型: %s", cfg.StorageType)

	// 创建文件服务
	fsChunker := chunker.NewFixedSizeChunker(4096)
	var redisClient *redis.Client
	if cfg.StorageType == "local-cached" {
		// 从配置中创建Redis客户端
		redisClient = redis.NewClient(&redis.Options{
			Addr: cfg.RedisAddr,
		})
	} else {
		// 对于非缓存存储类型，传递nil
		redisClient = nil
	}
	fileSvc := service.NewFileService(
		stack.BlockStore,
		stack.FileRepository,
		stack.BlockRepository,
		fsChunker,
		stack.SnapshotRepository,
		redisClient,
		true,
	)

	// 创建上下文用于演示
	demoCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 调用演示函数
	if err := demonstrateDataFlow(demoCtx); err != nil {
		log.Printf("演示3失败: %v", err)
	}
	if err := demonstrateSnapshots(demoCtx, fileSvc); err != nil {
		log.Printf("演示5失败: %v", err)
	}

	return nil
}

// demonstrateDataFlow 演示数据流
// 展示文件上传、存储和检索的完整流程
func demonstrateDataFlow(ctx context.Context) error {
	log.Printf("Context: %v", ctx)
	log.Println("\n========================================")
	log.Println("演示 3: 数据流")
	log.Println("========================================")

	return nil
}

// demonstrateSnapshots 演示快照功能
// 展示文件版本控制和快照管理功能
func demonstrateSnapshots(ctx context.Context, fileSvc *service.FileService) error {
	log.Printf("Context: %v", ctx)
	log.Println("\n========================================")
	log.Println("演示 5: 快照")
	log.Println("========================================")

	// 使用fileSvc参数，创建一个快照
	_, err := fileSvc.CreateSnapshot(ctx, "demo", "demo snapshot")
	if err != nil {
		log.Printf("创建快照失败: %v", err)
	}

	return nil
}

func main() {
	// 初始化Viper
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	// 添加配置目录，优先级从高到低
	viper.AddConfigPath(".")
	viper.AddConfigPath("../config")
	viper.AddConfigPath("config")
	viper.AddConfigPath("d:\\project\\SealockDoc\\core-storage\\config")

	// 启用环境变量支持
	viper.AutomaticEnv()

	// 尝试读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("Info: config.yaml file not found, relying on environment variables")
		} else {
			log.Fatalf("Error reading config file: %s", err)
		}
	} else {
		log.Printf("Info: Using config file: %s", viper.ConfigFileUsed())
	}

	log.Println("╔════════════════════════════════════════╗")
	log.Println("║        Sealock Doc 存储系统演示        ║")
	log.Println("║      Content-Addressed Storage        ║")
	log.Println("╚════════════════════════════════════════╝")

	// 移动测试文件到test目录后，确保main函数不直接引用它们
	// 集成测试现在位于test/目录中
	// 运行集成测试使用: go test -v ./test/...

	if err := demonstrateLocalStorage(context.Background()); err != nil {
		log.Fatalf("演示失败: %v", err)
	}
}