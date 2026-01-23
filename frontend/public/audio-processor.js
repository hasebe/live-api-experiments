class AudioRecorderProcessor extends AudioWorkletProcessor {
  constructor() {
    super();
    this.bufferSize = 2048; // Send roughly every 128ms at 16kHz? No, 2048/16000 = 128ms.
    // Actually WebAudio usually processes 128 frames per block.
    // We can buffer here or just send chunks.
    // Sending small chunks is fine for WebSocket.
    this._buffer = new Float32Array(this.bufferSize);
    this._bytesWritten = 0;
  }

  process(inputs, outputs, parameters) {
    const input = inputs[0];
    if (input && input.length > 0) {
      const channelData = input[0];
      
      // We could downsample here if needed, but let's assume context is set to 16kHz or we assume input is 16kHz.
      // Usually default is 44.1k or 48k. Downsampling is needed.
      // For simplicity, let's just send the raw float data and let the main thread or backend handle it?
      // Or just send every block.
      
      this.port.postMessage(channelData);
    }
    return true;
  }
}

registerProcessor('audio-recorder-processor', AudioRecorderProcessor);
