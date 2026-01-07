import React, { useState, useEffect, useRef } from 'react';
import ByteMD from 'byte-md';
import 'byte-md/dist/index.css';
import { uploadFile } from '../../api/file';

/**
 * Markdown在线实时编辑器组件
 * 集成byte-md，支持边写边存和图片粘贴上传
 * 
 * @param {string} value - 编辑器初始内容
 * @param {function} onChange - 内容变化回调
 * @param {function} onSave - 保存内容回调
 * @param {string} resourceId - 关联的资源ID
 */
const MarkdownEditor = ({ value, onChange, onSave, resourceId }) => {
  const [content, setContent] = useState(value || '');
  const [isSaving, setIsSaving] = useState(false);
  const debounceRef = useRef(null);
  const editorRef = useRef(null);

  // 处理内容变化
  const handleChange = (newContent) => {
    setContent(newContent);
    onChange?.(newContent);
    
    // 清除之前的debounce
    if (debounceRef.current) {
      clearTimeout(debounceRef.current);
    }
    
    // 设置新的debounce
    setIsSaving(true);
    debounceRef.current = setTimeout(() => {
      handleSave(newContent);
    }, 1000);
  };

  // 保存内容
  const handleSave = async (contentToSave) => {
    setIsSaving(false);
    try {
      await onSave(contentToSave);
    } catch (error) {
      console.error('保存失败:', error);
    }
  };

  // 处理粘贴事件
  const handlePaste = async (event) => {
    const items = event.clipboardData?.items;
    if (!items) return;

    for (let i = 0; i < items.length; i++) {
      if (items[i].type.indexOf('image') !== -1) {
        event.preventDefault();
        const file = items[i].getAsFile();
        if (file) {
          await handleImageUpload(file);
        }
      }
    }
  };

  // 处理图片上传
  const handleImageUpload = async (file) => {
    try {
      const formData = new FormData();
      formData.append('file', file);
      formData.append('resourceId', resourceId);

      const response = await uploadFile(formData);
      const imageUrl = response.data.url;
      
      // 插入Markdown图片语法
      const editor = editorRef.current;
      if (editor) {
        const currentContent = content;
        const cursorPos = editor.getSelection();
        const newContent = 
          currentContent.substring(0, cursorPos.start) + 
          `![${file.name}](${imageUrl})` + 
          currentContent.substring(cursorPos.end);
        
        setContent(newContent);
        onChange?.(newContent);
      }
    } catch (error) {
      console.error('图片上传失败:', error);
    }
  };

  useEffect(() => {
    setContent(value || '');
  }, [value]);

  return (
    <div className="markdown-editor">
      <div className="editor-header flex justify-between items-center mb-2">
        <div className="status text-sm text-gray-500">
          {isSaving ? '保存中...' : '已保存'}
        </div>
        <div className="toolbar">
          {/* 工具栏按钮 */}
        </div>
      </div>
      
      <div 
        ref={editorRef}
        onPaste={handlePaste}
      >
        <ByteMD.Editor
          value={content}
          onChange={handleChange}
          config={{
            autoHeight: true,
            placeholder: '开始编辑...',
          }}
        />
      </div>
      
      <div className="preview-pane mt-4">
        <ByteMD.Preview value={content} />
      </div>
    </div>
  );
};

export default MarkdownEditor;