package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sealock/core-storage/service"
)

// UploadHandler 处理文件上传操作
// 实现基于内容寻址的断点续传功能
// 遵循RESTful设计，具有清晰的错误处理机制
// 使用Redis进行上传会话跟踪
type UploadHandler struct {
	service *service.FileService
}

// NewUploadHandler 创建新的UploadHandler实例
func NewUploadHandler(fileService *service.FileService) *UploadHandler {
	return &UploadHandler{service: fileService}
}

// CheckFileHandler 检查文件是否已存在于系统中
// 当内容哈希匹配时实现"秒传"功能
// GET /check?fileHash={sha256}
func (h *UploadHandler) CheckFileHandler(c *gin.Context) {
	fileHash := c.Query("fileHash")
	if fileHash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fileHash参数是必需的"})
		return
	}

	// 检查文件是否已存在于系统中
	fileNode, err := h.service.GetFileNodeByContentHash(context.Background(), fileHash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "检查文件存在性失败"})
		return
	}

	if fileNode != nil {
		// 文件已存在，返回现有文件信息
		c.JSON(http.StatusOK, gin.H{
			"exists": true,
			"file": map[string]interface{}{
				"id":   fileNode.ID,
				"name": fileNode.Name,
				"size": fileNode.Size,
				"hash": fileNode.ContentHash,
			},
		})
		return
	}

	// 文件不存在，准备上传
	c.JSON(http.StatusOK, gin.H{
		"exists":       false,
		"uploadId":     uuid.New().String(), // 生成上传会话ID
		"requiredChunks": []int{},           // 将根据文件大小填充
	})
}

// UploadChunkHandler 处理单个文件分片上传
// POST /upload/chunk
// 请求体:
// {
//   "uploadId": "...",
//   "chunkIndex": 0,
//   "totalChunks": 5,
//   "chunkHash": "...",
//   "fileHash": "..."
// }
// 文件数据以原始二进制形式在请求体中发送
func (h *UploadHandler) UploadChunkHandler(c *gin.Context) {
	var req struct {
		UploadID    string `json:"uploadId"`
		ChunkIndex  int    `json:"chunkIndex"`
		TotalChunks int    `json:"totalChunks"`
		ChunkHash   string `json:"chunkHash"`
		FileHash    string `json:"fileHash"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求格式"})
		return
	}

	// 验证分片索引
	if req.ChunkIndex < 0 || req.ChunkIndex >= req.TotalChunks {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的分片索引"})
		return
	}

	// 从请求体读取分片数据
	chunkData, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取分片数据失败"})
		return
	}

	// 验证分片哈希
	computedHash := fmt.Sprintf("%x", h.service.ComputeSHA256(chunkData))
	if computedHash != req.ChunkHash {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":    "分片哈希不匹配",
			"expected": req.ChunkHash,
			"actual":   computedHash,
		})
		return
	}

	// 临时存储分片
	if err := h.service.StoreTemporaryChunk(req.UploadID, req.ChunkIndex, chunkData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "存储分片失败"})
		return
	}

	// 在Redis中跟踪分片接收情况，用于会话管理
	if err := h.service.RecordChunkReceived(req.UploadID, req.ChunkIndex, req.TotalChunks); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "记录分片失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"chunkIndex": req.ChunkIndex,
		"status":     "uploaded",
	})
}

// FinishUploadHandler 完成文件上传过程
// POST /upload/finish
// 请求体:
// {
//   "uploadId": "...",
//   "fileName": "example.pdf",
//   "fileSize": 123456,
//   "fileHash": "...",
//   "chunkHashes": ["hash1", "hash2", ...]
// }
func (h *UploadHandler) FinishUploadHandler(c *gin.Context) {
	var req struct {
		UploadID    string   `json:"uploadId"`
		FileName    string   `json:"fileName"`
		FileSize    int64    `json:"fileSize"`
		FileHash    string   `json:"fileHash"`
		ChunkHashes []string `json:"chunkHashes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求格式"})
		return
	}

	// 验证所有分片是否都已接收
	missingChunks, err := h.service.GetMissingChunks(req.UploadID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "验证分片失败"})
		return
	}

	if len(missingChunks) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":       "缺少分片",
			"missing":     missingChunks,
			"totalChunks": len(req.ChunkHashes),
		})
		return
	}

	// 验证文件哈希
	reconstructedHash, err := h.service.ReconstructFileHash(req.UploadID, req.ChunkHashes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "验证文件哈希失败"})
		return
	}

	if reconstructedHash != req.FileHash {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         "文件哈希不匹配",
			"expected":      req.FileHash,
			"reconstructed": reconstructedHash,
		})
		return
	}

	// 创建最终的文件条目
	fileNode, err := h.service.CreateFileNode(
		context.Background(),
		req.FileName,
		req.FileSize,
		req.FileHash,
		req.ChunkHashes,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建文件条目失败: " + err.Error()})
		return
	}

	// 清理临时资源
	if err := h.service.CleanupUploadSession(req.UploadID); err != nil {
		// 记录清理错误但不使请求失败
		fmt.Printf("警告: 清理上传会话 %s 失败: %v\n", req.UploadID, err)
	}

	c.JSON(http.StatusOK, gin.H{
		"file": map[string]interface{}{
			"id":   fileNode.ID,
			"name": fileNode.Name,
			"size": fileNode.Size,
			"hash": fileNode.ContentHash,
		},
	})
}

// RegisterUploadRoutes 设置上传相关的路由
func RegisterUploadRoutes(r *gin.Engine, fileService *service.FileService) {
	handler := NewUploadHandler(fileService)

	uploadGroup := r.Group("/api/v1/upload")
	{
		uploadGroup.GET("/check", handler.CheckFileHandler)   // 检查文件是否存在
		uploadGroup.POST("/chunk", handler.UploadChunkHandler) // 上传文件分片
		uploadGroup.POST("/finish", handler.FinishUploadHandler) // 完成上传
	}
}