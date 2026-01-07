// Web Worker for calculating SHA-256 hashes
// This worker calculates both the full file hash and individual chunk hashes

importScripts('https://cdn.jsdelivr.net/npm/crypto-js@4.1.1/crypto-js.js');

self.onmessage = async function(e) {
  const { file, chunkSize } = e.data;
  
  // Read the entire file to calculate the overall hash
  const reader = new FileReader();
  
  // Promise wrapper for FileReader onload
  const readFile = (file) => {
    return new Promise((resolve, reject) => {
      reader.onload = () => resolve(reader.result);
      reader.onerror = () => reject(reader.error);
      reader.readAsArrayBuffer(file);
    });
  };
  
  try {
    const arrayBuffer = await readFile(file);
    const uint8Array = new Uint8Array(arrayBuffer);
    
    // Calculate overall file hash
    const fileHash = CryptoJS.SHA256(CryptoJS.enc.Latin1.parse(uint8Array)).toString();
    
    // Calculate chunk hashes
    const totalChunks = Math.ceil(file.size / chunkSize);
    const chunkHashes = [];
    
    for (let i = 0; i < totalChunks; i++) {
      const start = i * chunkSize;
      const end = Math.min(start + chunkSize, file.size);
      const chunk = file.slice(start, end);
      
      const chunkReader = new FileReader();
      const chunkArrayBuffer = await new Promise((resolve, reject) => {
        chunkReader.onload = () => resolve(chunkReader.result);
        chunkReader.onerror = () => reject(chunkReader.error);
        chunkReader.readAsArrayBuffer(chunk);
      });
      
      const chunkUint8Array = new Uint8Array(chunkArrayBuffer);
      const chunkHash = CryptoJS.SHA256(CryptoJS.enc.Latin1.parse(chunkUint8Array)).toString();
      chunkHashes.push(chunkHash);
    }
    
    // Send results back to main thread
    self.postMessage({
      fileHash,
      chunkHashes,
      totalChunks,
      chunkSize
    });
  } catch (error) {
    self.postMessage({ error: error.message });
  }
};