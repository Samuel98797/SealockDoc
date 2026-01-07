import React from 'react';
import ReactDOM from 'react-dom/client';
import FileUpload from './components/FileUpload/FileUpload';

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(
  <React.StrictMode>
    <div className="p-8">
      <h1 className="text-2xl font-bold mb-6">Sealock Doc - 大文件上传</h1>
      <FileUpload />
    </div>
  </React.StrictMode>
);