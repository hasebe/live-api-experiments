import { useState, useRef, useCallback, useEffect } from 'react';

export function useLiveApi() {
  const [isConnected, setIsConnected] = useState(false);
  const [isRecording, setIsRecording] = useState(false);
  const socketRef = useRef<WebSocket | null>(null);

  // Audio Playback Refs
  const audioContextRef = useRef<AudioContext | null>(null);
  const nextPlayTimeRef = useRef<number>(0);
  const audioQueueRef = useRef<Float32Array[]>([]);
  const isPlayingRef = useRef<boolean>(false);
  const gainNodeRef = useRef<GainNode | null>(null);

  // Audio Recording Refs
  const processorRef = useRef<AudioWorkletNode | null>(null);
  const sourceRef = useRef<MediaStreamAudioSourceNode | null>(null);
  const recordingGainRef = useRef<GainNode | null>(null);
  const recordingContextRef = useRef<AudioContext | null>(null);

  // Initialize Audio Context for Playback
  const ensureAudioContext = useCallback(() => {
    if (!audioContextRef.current) {
      // 24kHz is typical for Gemini, but we should let the browser decide or config it if needed
      // Ideally matching the response sample rate (24kHz)
      audioContextRef.current = new AudioContext({ sampleRate: 24000 });
      nextPlayTimeRef.current = audioContextRef.current.currentTime;

      const gainNode = audioContextRef.current.createGain();
      gainNode.connect(audioContextRef.current.destination);
      gainNodeRef.current = gainNode;
    } else if (audioContextRef.current.state === 'suspended') {
      audioContextRef.current.resume();
    }
  }, []);

  const scheduleNextChunk = useCallback(() => {
    const ctx = audioContextRef.current;
    if (!ctx || audioQueueRef.current.length === 0) {
      isPlayingRef.current = false;
      return;
    }

    isPlayingRef.current = true;
    const chunk = audioQueueRef.current.shift()!;

    // Create buffer
    const buffer = ctx.createBuffer(1, chunk.length, 24000);
    buffer.getChannelData(0).set(chunk);

    const source = ctx.createBufferSource();
    source.buffer = buffer;
    source.connect(gainNodeRef.current!);

    // Schedule play
    // Ensure we don't play in the past
    // If nextPlayTime is behind currentTime, catch up to currentTime + small buffer
    if (nextPlayTimeRef.current < ctx.currentTime) {
      nextPlayTimeRef.current = ctx.currentTime;
    }

    source.start(nextPlayTimeRef.current);
    nextPlayTimeRef.current += buffer.duration;

    // Use onended to check for more chunks or just aggressive scheduling?
    // Aggressive scheduling is often smoother for streaming.
    // We can just loop until queue is empty or we have scheduled enough ahead.
    // For simplicity, let's schedule everything in the queue immediately when it's available.
    scheduleNextChunk();
  }, []);

  const enqueueAudio = useCallback((base64Audio: string) => {
    ensureAudioContext();

    try {
      const binaryString = window.atob(base64Audio);
      const len = binaryString.length;
      const bytes = new Uint8Array(len);
      for (let i = 0; i < len; i++) {
        bytes[i] = binaryString.charCodeAt(i);
      }

      // PCM16 -> Float32
      const pcm16 = new Int16Array(bytes.buffer);
      const float32 = new Float32Array(pcm16.length);
      for (let i = 0; i < pcm16.length; i++) {
        float32[i] = pcm16[i] / 32768.0;
      }

      audioQueueRef.current.push(float32);

      if (!isPlayingRef.current) {
        scheduleNextChunk();
      } else {
        // If we are already playing, scheduleNextChunk will be called recursively/looping
        // but we need to trigger it if the queue was exhausted effectively
        // Actually, just calling scheduleNextChunk() is safe because it pulls from queue
        scheduleNextChunk();
      }

    } catch (e) {
      console.error('Audio decode error', e);
    }
  }, [ensureAudioContext, scheduleNextChunk]);

  const stopPlayback = useCallback(() => {
    // Clear queue
    audioQueueRef.current = [];
    isPlayingRef.current = false;

    // Stop currently scheduled sources?
    // It's hard to stop individual scheduled sources without keeping track of them.
    // Easiest is to suspend context or disconnect gain node temporarily, or close/recreate context.
    // For now, let's just accept we might hear a split second of remaining buffer or implement a "stop all" if valuable.
    // Re-creating AudioContext is nuclear but effective for "interrupted".
    if (audioContextRef.current) {
      const ctx = audioContextRef.current;
      audioContextRef.current = null; // Nullify immediately to prevent double-close
      nextPlayTimeRef.current = 0;

      if (ctx.state !== 'closed') {
        ctx.close();
      }
    }
  }, []);

  const connect = useCallback((url: string) => {
    const socket = new WebSocket(url);
    socket.binaryType = 'arraybuffer';

    socket.onopen = () => {
      console.log('Connected to Backend');
      setIsConnected(true);
      // Prepare audio for playback
      ensureAudioContext();
    };

    socket.onmessage = async (event) => {
      let data: any;
      try {
        if (typeof event.data === 'string') {
          data = JSON.parse(event.data);
        } else if (event.data instanceof ArrayBuffer) {
          const text = new TextDecoder().decode(event.data);
          data = JSON.parse(text);
        }
      } catch (e) {
        console.error('Failed to parse message', e);
        return;
      }

      // Check for Interruption
      if (data?.serverContent?.interrupted) {
        console.log('Interruption signal received');
        stopPlayback();
        return;
      }



      // Handle Audio
      if (data?.serverContent?.modelTurn?.parts) {
        for (const part of data.serverContent.modelTurn.parts) {
          if (part.inlineData && part.inlineData.mimeType.startsWith('audio/')) {
            enqueueAudio(part.inlineData.data);
          }
        }
      }
    };

    socket.onclose = () => {
      console.log('Disconnected');
      setIsConnected(false);
      stopPlayback();
    };

    socketRef.current = socket;
  }, [ensureAudioContext, enqueueAudio, stopPlayback]);

  const startRecording = useCallback(async () => {
    if (!socketRef.current || socketRef.current.readyState !== WebSocket.OPEN) {
      console.error('Socket not connected');
      return;
    }

    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      const ctx = new AudioContext({ sampleRate: 16000 });
      recordingContextRef.current = ctx;

      await ctx.audioWorklet.addModule('/audio-processor.js');

      const source = ctx.createMediaStreamSource(stream);
      const processor = new AudioWorkletNode(ctx, 'audio-recorder-processor');
      const gain = ctx.createGain();
      gain.gain.value = 0; // Mute input

      processor.port.onmessage = (e) => {
        const inputData = e.data;
        const pcm16 = new Int16Array(inputData.length);
        for (let i = 0; i < inputData.length; i++) {
          const s = Math.max(-1, Math.min(1, inputData[i]));
          pcm16[i] = s < 0 ? s * 0x8000 : s * 0x7FFF;
        }
        if (socketRef.current?.readyState === WebSocket.OPEN) {
          socketRef.current.send(pcm16.buffer);
        }
      };

      source.connect(processor);
      processor.connect(gain);
      gain.connect(ctx.destination);

      sourceRef.current = source;
      processorRef.current = processor;
      recordingGainRef.current = gain;
      setIsRecording(true);
    } catch (err) {
      console.error('Error starting audio', err);
    }
  }, []);

  const stopRecording = useCallback(() => {
    sourceRef.current?.disconnect();
    sourceRef.current?.mediaStream.getTracks().forEach(t => t.stop());
    processorRef.current?.disconnect();

    if (recordingContextRef.current && recordingContextRef.current.state !== 'closed') {
      recordingContextRef.current.close();
    }
    recordingContextRef.current = null;
    setIsRecording(false);
  }, []);

  const disconnect = useCallback(() => {
    stopRecording();
    socketRef.current?.close();
    stopPlayback();
    setIsConnected(false);
  }, [stopRecording, stopPlayback]);

  useEffect(() => {
    return () => {
      disconnect();
    };
  }, [disconnect]);

  return { isConnected, isRecording, connect, disconnect, startRecording, stopRecording };
}
