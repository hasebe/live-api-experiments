'use client';

import { useState } from 'react';
import { useLiveApi } from '@/hooks/useLiveApi';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';

export default function Home() {
  const { isConnected, isRecording, connect, disconnect, startRecording, stopRecording } = useLiveApi();
  const [error, setError] = useState<string | null>(null);
  const [backend, setBackend] = useState<'dotnet' | 'go'>('dotnet');

  const handleConnect = () => {
    try {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const host = backend === 'go' ? 'localhost:8080' : 'localhost:5093';
      const url = `${protocol}//${host}/ws`;
      connect(url);
      setError(null);
    } catch (e) {
      setError('Failed to connect');
    }
  };

  return (
    <div className="min-h-screen bg-neutral-50 flex items-center justify-center p-4 dark:bg-neutral-900">
      <Card className="w-full max-w-md shadow-xl border-neutral-200 dark:border-neutral-800">
        <CardHeader>
          <CardTitle className="text-center text-2xl font-bold text-neutral-900 dark:text-neutral-50">
            Live API Demo
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          {/* Backend Switcher */}
          <div className="flex flex-col items-center gap-2">
            <span className="text-xs font-semibold uppercase tracking-wider text-neutral-500 dark:text-neutral-400">
              Backend
            </span>
            <div className="flex bg-neutral-100 dark:bg-neutral-800 p-1 rounded-lg border border-neutral-200 dark:border-neutral-700">
              <button
                onClick={() => setBackend('dotnet')}
                disabled={isConnected}
                className={`px-4 py-1.5 text-sm font-medium rounded-md transition-all ${backend === 'dotnet'
                    ? 'bg-white dark:bg-neutral-700 shadow-sm text-neutral-900 dark:text-neutral-100'
                    : 'text-neutral-500 hover:text-neutral-700 dark:text-neutral-400 dark:hover:text-neutral-200'
                  } ${isConnected ? 'opacity-50 cursor-not-allowed' : ''}`}
              >
                .NET (5093)
              </button>
              <button
                onClick={() => setBackend('go')}
                disabled={isConnected}
                className={`px-4 py-1.5 text-sm font-medium rounded-md transition-all ${backend === 'go'
                    ? 'bg-white dark:bg-neutral-700 shadow-sm text-neutral-900 dark:text-neutral-100'
                    : 'text-neutral-500 hover:text-neutral-700 dark:text-neutral-400 dark:hover:text-neutral-200'
                  } ${isConnected ? 'opacity-50 cursor-not-allowed' : ''}`}
              >
                Go (8080)
              </button>
            </div>
          </div>

          <div className="flex flex-col space-y-4">
            {!isConnected ? (
              <Button onClick={handleConnect} className="w-full font-medium transition-all hover:scale-[1.02]">
                Connect to Server
              </Button>
            ) : (
              <Button variant="destructive" onClick={disconnect} className="w-full transition-all hover:scale-[1.02]">
                Disconnect
              </Button>
            )}

            <div className="grid grid-cols-2 gap-4">
              <Button
                onClick={startRecording}
                disabled={!isConnected || isRecording}
                variant={isRecording ? "secondary" : "default"}
                className="w-full"
              >
                Start Stream
              </Button>
              <Button
                onClick={stopRecording}
                disabled={!isRecording}
                variant="outline"
                className="w-full"
              >
                Stop Stream
              </Button>
            </div>
          </div>

          <div className="text-center text-sm font-medium text-neutral-500 dark:text-neutral-400 p-2 bg-neutral-100 dark:bg-neutral-800 rounded-md">
            Status:
            <span className={`ml-2 ${isConnected ? 'text-green-600 dark:text-green-400' : 'text-neutral-500'}`}>
              {isConnected ? (isRecording ? "Streaming Audio..." : "Connected") : "Disconnected"}
            </span>
          </div>

          {error && (
            <div className="text-center text-sm text-red-500 bg-red-50 p-2 rounded-md">
              {error}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
