package chunker

import (
	"crypto/sha256"
	"encoding/hex"
)

// Chunker 定义文件分块接口
type Chunker interface {
	// Chunk 将数据分割成块，返回每个块的 hash
	Chunk(data []byte) ([]string, error)

	// ChunkSize 返回固定块大小（仅用于固定大小分块）
	ChunkSize() int
}

// FixedSizeChunker 使用固定大小的分块器
// 简单有效，但对文件插入/删除敏感（可能导致块对齐错位）
type FixedSizeChunker struct {
	blockSize int
}

// NewFixedSizeChunker 创建固定大小分块器
// 典型块大小：4KB, 8KB, 16KB
func NewFixedSizeChunker(blockSize int) *FixedSizeChunker {
	if blockSize <= 0 {
		blockSize = 8192 // 默认 8KB
	}
	return &FixedSizeChunker{blockSize: blockSize}
}

// Chunk 将数据分割成固定大小的块
func (c *FixedSizeChunker) Chunk(data []byte) ([]string, error) {
	if len(data) == 0 {
		return []string{}, nil
	}

	var hashes []string
	for i := 0; i < len(data); i += c.blockSize {
		end := i + c.blockSize
		if end > len(data) {
			end = len(data)
		}

		chunk := data[i:end]
		hash := sha256.Sum256(chunk)
		hashes = append(hashes, hex.EncodeToString(hash[:]))
	}

	return hashes, nil
}

// ChunkSize 返回块大小
func (c *FixedSizeChunker) ChunkSize() int {
	return c.blockSize
}

// ============ Content-Defined Chunking (CDC) 实现 ============
// 基于内容的分块器：通过内容特征点而非固定位置分块
// 优点：文件中部修改仅影响相邻块，不会导致全文块重排

// CDCChunker 使用内容定义的分块（简化的 Rabin 指纹实现）
type CDCChunker struct {
	minSize    int // 最小块大小
	maxSize    int // 最大块大小
	avgSize    int // 平均块大小
	windowSize int // 滑动窗口大小
}

// NewCDCChunker 创建 CDC 分块器
// 参数建议：minSize=2KB, avgSize=8KB, maxSize=64KB
func NewCDCChunker(minSize, avgSize, maxSize int) *CDCChunker {
	if minSize <= 0 {
		minSize = 2048
	}
	if avgSize <= 0 {
		avgSize = 8192
	}
	if maxSize <= 0 {
		maxSize = 65536
	}
	if minSize >= avgSize || avgSize >= maxSize {
		minSize = 2048
		avgSize = 8192
		maxSize = 65536
	}

	return &CDCChunker{
		minSize:    minSize,
		avgSize:    avgSize,
		maxSize:    maxSize,
		windowSize: 64, // 滑动窗口，用于计算指纹
	}
}

// Chunk 使用 CDC 算法分块
func (c *CDCChunker) Chunk(data []byte) ([]string, error) {
	if len(data) == 0 {
		return []string{}, nil
	}

	var hashes []string
	var pos int

	for pos < len(data) {
		// 确定块的起始位置
		chunkStart := pos
		chunkEnd := pos + c.minSize

		// 扫描直到找到分界点（简化：每 avgSize 字节检查一次）
		for chunkEnd < pos+c.maxSize && chunkEnd < len(data) {
			// 计算当前窗口的简单哈希值
			if c.isChunkBoundary(data, chunkEnd) {
				break
			}
			chunkEnd++
		}

		// 确保块不超过最大大小
		if chunkEnd > pos+c.maxSize {
			chunkEnd = pos + c.maxSize
		}
		if chunkEnd > len(data) {
			chunkEnd = len(data)
		}

		// 计算块的哈希
		chunk := data[chunkStart:chunkEnd]
		hash := sha256.Sum256(chunk)
		hashes = append(hashes, hex.EncodeToString(hash[:]))

		pos = chunkEnd
	}

	return hashes, nil
}

// isChunkBoundary 简化的分界点检测
// 在实际应用中应使用 Rabin Fingerprint 或类似算法
func (c *CDCChunker) isChunkBoundary(data []byte, pos int) bool {
	if pos < c.minSize || pos > len(data) {
		return false
	}

	// 简化实现：每 avgSize 字节检查一次
	// 实际 CDC 应基于数据内容特征
	return (pos-c.minSize)%c.avgSize == 0
}

// ChunkSize 返回平均块大小
func (c *CDCChunker) ChunkSize() int {
	return c.avgSize
}

// ============ 文件指纹计算（用于文件去重） ============

// ComputeFileMerkleHash 计算文件的 Merkle 哈希
// 所有块的哈希按顺序拼接后再哈希一次
func ComputeFileMerkleHash(blockHashes []string) (string, error) {
	if len(blockHashes) == 0 {
		emptyHash := sha256.Sum256([]byte{})
		return hex.EncodeToString(emptyHash[:]), nil
	}

	// 将所有块哈希拼接
	var combined string
	for _, h := range blockHashes {
		combined += h
	}

	// 计算最终哈希
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:]), nil
}
