'use client';

import { use, useState, useEffect, useRef } from 'react';
import { useQuery } from '@tanstack/react-query';
import { authStore } from '@/store/auth';
import {
    Terminal,
    CheckCircle2,
    XCircle,
    Loader2,
    Clock,
    Copy,
    Pause,
    Play,
    ArrowLeft
} from 'lucide-react';
import { cn } from '@/lib/utils';
import Link from 'next/link';

export default function ExecutionDetailPage({ params: paramsPromise }: { params: Promise<{ executionId: string }> }) {
    const params = use(paramsPromise);
    const executionId = params.executionId;
    const client = authStore.getState().client;
    const [isPaused, setIsPaused] = useState(false);
    const [autoScroll, setAutoScroll] = useState(true);
    const scrollRef = useRef<HTMLDivElement>(null);

    const { data: logsData, isLoading } = useQuery({
        queryKey: ['logs', executionId],
        queryFn: () => client?.listExecutionLogs(executionId),
        enabled: !!client && !isPaused,
        refetchInterval: 2000,
    });

    const logs = logsData?.logs || [];

    useEffect(() => {
        if (autoScroll && scrollRef.current) {
            scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
        }
    }, [logs, autoScroll]);

    return (
        <div className="flex flex-col h-full space-y-6">
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                    <Link href="/sites" className="rounded-full p-2 hover:bg-slate-200 transition-colors">
                        <ArrowLeft className="h-5 w-5 text-slate-500" />
                    </Link>
                    <div>
                        <h1 className="text-2xl font-bold text-slate-900">Execution Logs</h1>
                        <p className="text-sm text-slate-500 font-mono">{executionId}</p>
                    </div>
                </div>
                <div className="flex items-center gap-2">
                    <button
                        onClick={() => setIsPaused(!isPaused)}
                        className={cn(
                            "flex items-center gap-2 rounded-lg border px-4 py-2 text-sm font-semibold shadow-sm transition-colors",
                            isPaused ? "bg-indigo-600 text-white hover:bg-indigo-500" : "bg-white text-slate-700 hover:bg-slate-50"
                        )}
                    >
                        {isPaused ? <Play className="h-4 w-4" /> : <Pause className="h-4 w-4" />}
                        {isPaused ? 'Resume Polling' : 'Pause Logs'}
                    </button>
                    <button className="flex items-center gap-2 rounded-lg bg-white border px-4 py-2 text-sm font-semibold text-slate-700 shadow-sm hover:bg-slate-50">
                        <Copy className="h-4 w-4" />
                        Copy Logs
                    </button>
                </div>
            </div>

            <div className="flex-1 flex flex-col rounded-2xl bg-slate-900 shadow-2xl overflow-hidden border border-slate-800">
                <div className="flex items-center justify-between bg-slate-800 px-4 py-2">
                    <div className="flex items-center gap-4">
                        <div className="flex items-center gap-2">
                            <div className="h-2 w-2 rounded-full bg-emerald-400 animate-pulse" />
                            <span className="text-xs font-bold text-slate-400 uppercase tracking-widest">Live Terminal</span>
                        </div>
                        <span className="text-xs text-slate-500">|</span>
                        <span className="text-xs text-slate-400">{logs.length} events logged</span>
                    </div>
                    <div className="flex items-center gap-4 text-xs">
                        <div className="flex items-center gap-1 text-slate-400">
                            <Clock className="h-3 w-3" />
                            <span>Started {logs.length > 0 ? new Date(logs[0].timestamp).toLocaleTimeString() : '-'}</span>
                        </div>
                    </div>
                </div>

                <div
                    ref={scrollRef}
                    className="flex-1 overflow-y-auto p-6 font-mono text-xs sm:text-sm leading-relaxed scroll-smooth custom-scrollbar"
                >
                    {isLoading && logs.length === 0 ? (
                        <div className="flex h-full items-center justify-center text-slate-500">
                            <Loader2 className="h-6 w-6 animate-spin mr-2" />
                            Initializing log stream...
                        </div>
                    ) : logs.length === 0 ? (
                        <div className="flex h-full items-center justify-center text-slate-500 italic">
                            Waiting for agent activity...
                        </div>
                    ) : (
                        <div className="space-y-1">
                            {logs.map((log, i) => (
                                <div key={i} className="flex gap-4 group">
                                    <span className="shrink-0 text-slate-600 select-none">[{new Date(log.timestamp).toLocaleTimeString()}]</span>
                                    <span className={cn(
                                        "shrink-0 font-bold w-12",
                                        log.level === 'ERROR' ? "text-red-400" :
                                            log.level === 'WARN' ? "text-amber-400" : "text-indigo-400"
                                    )}>{log.level}</span>
                                    <span className="text-slate-300 break-all">{log.message}</span>
                                </div>
                            ))}
                        </div>
                    )}
                </div>

                <div className="bg-slate-800/50 px-4 py-2 flex items-center justify-between border-t border-slate-700">
                    <div className="flex items-center gap-2">
                        <input
                            type="checkbox" id="autoscroll"
                            checked={autoScroll} onChange={(e) => setAutoScroll(e.target.checked)}
                            className="h-3 w-3 rounded bg-slate-700 border-slate-600 text-indigo-500"
                        />
                        <label htmlFor="autoscroll" className="text-xs text-slate-400 select-none">Auto-scroll to end</label>
                    </div>
                    <div className="text-[10px] text-slate-500 uppercase tracking-widest">
                        n-kudo-edge v1.0.0
                    </div>
                </div>
            </div>

            <style jsx global>{`
        .custom-scrollbar::-webkit-scrollbar {
          width: 8px;
        }
        .custom-scrollbar::-webkit-scrollbar-track {
          background: #0f172a;
        }
        .custom-scrollbar::-webkit-scrollbar-thumb {
          background: #334155;
          border-radius: 4px;
        }
        .custom-scrollbar::-webkit-scrollbar-thumb:hover {
          background: #475569;
        }
      `}</style>
        </div>
    );
}
