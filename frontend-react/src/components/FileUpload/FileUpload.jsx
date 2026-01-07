import { useState, useRef, useEffect } from 'react';

/**
 * 大文件分片上传组件
 * 支持拖拽上传、多文件并行处理、Web Worker计算SHA-256
 * 区分三种状态：计算哈希、秒传、上传分片
 */
export default function FileUpload() {
  const [files, setFiles] = useState([]);
  const [isDragging, setIsDragging] = useState(false);
  const fileInputRef = useRef(null);
  const workerRef = useRef(null);

  // 初始化Web Worker
  useEffect(() => {
    // 创建Worker实例，指向hashWorker.js
    workerRef.current = new Worker(new URL('./hashWorker.js', import.meta.url));

    // 处理Worker消息
    workerRef.current.onmessage = (e) => {
      if (e.data.error) {
        console.error('Worker error:', e.data.error);
        return;
      }
      
      // 获取正在处理的文件ID
      const processingFile = files.find(f => f.status === 'computing');
      if (!processingFile) return;
      
      setFiles(prev => prev.map(f => 
        f.id === processingFile.id 
          ? { ...f, status: 'checking', message: '正在检查秒传...' }
          : f
      ));

      // 检查秒传
      checkFileExistence(processingFile.id, e.data);
    };

    return () => {
      workerRef.current?.terminate();
    };
  }, [files]);

  // 处理文件选择
  const handleFileSelect = (selectedFiles) => {
    const newFiles = Array.from(selectedFiles).map(file => ({
      id: Date.now() + Math.random(),
      file,
      status: 'computing',
      progress: 0,
      message: '正在计算哈希...'
    }));

    setFiles(prev => [...prev, ...newFiles]);
    newFiles.forEach(processFile);
  };

  // 处理单个文件
  const processFile = (fileObj) => {
    const chunkSize = 4096; // 与后端保持一致的分片大小

    try {
      workerRef.current.postMessage({
        file: fileObj.file,
        chunkSize
      });
    } catch (error) {
      setFiles(prev => prev.map(f => 
        f.id === fileObj.id 
          ? { ...f, status: 'error', message: '计算哈希失败: ' + error.message } 
          : f
      ));
    }
  };

  // 检查文件是否存在（秒传）
  const checkFileExistence = async (fileId, hashData) => {
    try {
      const response = await fetch('/api/upload/check', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ fileHash: hashData.fileHash })
      });

      const result = await response.json();

      if (result.exists) {
        setFiles(prev => prev.map(f => 
          f.id === fileId 
            ? { 
                ...f, 
                status: 'success', 
                progress: 100,
                message: '秒传成功！' 
              } 
            : f
        ));
      } else {
        setFiles(prev => prev.map(f => 
          f.id === fileId 
            ? { 
                ...f, 
                status: 'uploading', 
                progress: 0,
                message: `正在上传分片 (0/${hashData.totalChunks})` 
              } 
            : f
        ));
        
        // 开始上传分片
        uploadChunks(fileId, hashData);
      }
    } catch (error) {
      setFiles(prev => prev.map(f => 
        f.id === fileId 
          ? { ...f, status: 'error', message: '检查失败: ' + error.message } 
          : f
      ));
    }
  };

  // 上传文件分片
  const uploadChunks = async (fileId, hashData) => {
    const fileObj = files.find(f => f.id === fileId);
    if (!fileObj) return;
    
    const { file } = fileObj;
    const { chunkHashes, totalChunks, chunkSize } = hashData;
    let uploadedChunks = 0;

    const uploadChunk = async (chunkIndex) => {
      const start = chunkIndex * chunkSize;
      const end = Math.min(file.size, start + chunkSize);
      const chunk = file.slice(start, end);

      const formData = new FormData();
      formData.append('chunk', chunk, `chunk-${chunkIndex}`);
      formData.append('fileHash', hashData.fileHash);
      formData.append('chunkIndex', chunkIndex);
      formData.append('chunkHash', chunkHashes[chunkIndex]);
      formData.append('totalChunks', totalChunks);

      try {
        const response = await fetch('/api/upload/chunk', {
          method: 'POST',
          body: formData
        });

        if (response.ok) {
          uploadedChunks++;
          const progress = Math.round((uploadedChunks / totalChunks) * 100);
          
          setFiles(prev => prev.map(f => 
            f.id === fileId 
              ? { 
                  ...f, 
                  progress,
                  message: `正在上传分片 (${uploadedChunks}/${totalChunks})` 
                } 
              : f
          ));

          if (uploadedChunks === totalChunks) {
            await finishUpload(fileId, hashData);
          }
        }
      } catch (error) {
        setFiles(prev => prev.map(f => 
          f.id === fileId 
            ? { ...f, status: 'error', message: `分片 ${chunkIndex} 上传失败` } 
            : f
        ));
      }
    };

    // 并行上传（限制并发数）
    const maxConcurrent = 3;
    const uploadQueue = [];

    for (let i = 0; i < totalChunks; i++) {
      uploadQueue.push(uploadChunk(i));
      
      if (uploadQueue.length >= maxConcurrent) {
        await Promise.race(uploadQueue);
        uploadQueue.shift();
      }
    }

    await Promise.all(uploadQueue);
  };

  // 完成上传
  const finishUpload = async (fileId, hashData) => {
    const fileObj = files.find(f => f.id === fileId);
    if (!fileObj) return;
    
    try {
      const response = await fetch('/api/upload/finish', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          fileHash: hashData.fileHash,
          fileName: fileObj.file.name,
          fileSize: fileObj.file.size,
          totalChunks: hashData.totalChunks
        })
      });

      if (response.ok) {
        setFiles(prev => prev.map(f => 
          f.id === fileId 
            ? { ...f, status: 'success', progress: 100, message: '上传成功！' } 
            : f
        ));
      }
    } catch (error) {
      setFiles(prev => prev.map(f => 
        f.id === fileId 
          ? { ...f, status: 'error', message: '完成上传失败: ' + error.message } 
          : f
      ));
    }
  };

  // 拖拽事件处理
  const handleDragOver = (e) => {
    e.preventDefault();
    setIsDragging(true);
  };

  const handleDragLeave = () => {
    setIsDragging(false);
  };

  const handleDrop = (e) => {
    e.preventDefault();
    setIsDragging(false);
    handleFileSelect(e.dataTransfer.files);
  };

  return (
    <div className="max-w-2xl mx-auto">
      {/* 拖拽区域 */}
      <div
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        className={`\n          border-2 border-dashed rounded-xl p-12 text-center cursor-pointer
          transition-all duration-300 ease-in-out
          ${isDragging 
            ? 'border-blue-500 bg-blue-50' 
            : 'border-gray-300 hover:border-blue-400 hover:bg-gray-50'}
        `}
        onClick={() => fileInputRef.current?.click()}
      >
        <input
          type="file"
          ref={fileInputRef}
          className="hidden"
          multiple
          onChange={(e) => handleFileSelect(e.target.files)}
        />
        <div className="flex flex-col items-center">
          <svg
            className="w-12 h-12 text-gray-400 mb-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth="2"
              d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 9.9L16 10l4-4m0 0L14 2m6 6l-4 4"
            ></path>
          </svg>
          <p className="text-gray-600 mb-2">
            {isDragging
              ? '释放以上传文件'
              : '点击或拖拽文件到此区域'}
          </p>
          <p className="text-sm text-gray-500">
            支持多文件上传，最大支持 10GB 文件
          </p>
        </div>
      </div>

      {/* 文件列表 */}
      {files.length > 0 && (
        <div className="mt-8">
          <h2 className="text-lg font-semibold mb-4">上传任务</h2>
          <ul className="space-y-4">
            {files.map((file) => (
              <li key={file.id} className="bg-gray-50 rounded-lg p-4">
                <div className="flex justify-between items-start mb-2">
                  <div>
                    <p className="font-medium truncate max-w-xs">
                      {file.file.name}
                    </p>
                    <p className="text-sm text-gray-500">
                      {(file.file.size / (1024 * 1024)).toFixed(2)} MB
                    </p>
                  </div>
                  <span
                    className={`\n                      px-2 py-1 rounded-full text-xs font-medium
                      ${
                        file.status === 'success'
                          ? 'bg-green-100 text-green-800'
                          : file.status === 'error'
                            ? 'bg-red-100 text-red-800'
                            : 'bg-blue-100 text-blue-800'
                      }
                    `}
                  >
                    {file.status === 'success'
                      ? '成功'
                      : file.status === 'error'
                        ? '失败'
                        : '处理中'}
                  </span>
                </div>

                {/* 进度条 */}
                <div className="mt-2">
                  <div className="flex justify-between text-xs text-gray-500 mb-1">
                    <span>{file.message}</span>
                    <span>{file.progress}%</span>
                  </div>
                  <div className="w-full bg-gray-200 rounded-full h-2.5">
                    <div
                      className={`\n                        h-2.5 rounded-full
                        ${
                          file.status === 'error'
                            ? 'bg-red-500'
                            : 'bg-blue-600'
                        }
                      `}
                      style={{ width: `${file.progress}%` }}
                    ></div>
                  </div>
                </div>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}