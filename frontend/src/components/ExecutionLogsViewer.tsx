import { useState, useRef, useEffect } from 'react'
import { Loader2, Copy, Check, RefreshCw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { useExecutionLogs } from '@/api/hooks'
import { copyToClipboard, cn } from '@/lib/utils'

interface ExecutionLogsViewerProps {
    executionId: string
}

const severityColor: Record<string, string> = {
    DEBUG: 'text-muted-foreground',
    INFO: 'text-blue-400',
    WARN: 'text-amber-400',
    ERROR: 'text-red-400',
}

export default function ExecutionLogsViewer({ executionId }: ExecutionLogsViewerProps) {
    const [autoRefresh, setAutoRefresh] = useState(true)
    const [copied, setCopied] = useState(false)
    const { data: logs, isLoading, refetch } = useExecutionLogs(executionId, undefined, {
        refetchInterval: autoRefresh ? 2000 : false,
    })
    const bottomRef = useRef<HTMLDivElement>(null)

    useEffect(() => {
        bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
    }, [logs])

    const handleCopy = async () => {
        if (!logs) return
        const text = logs.map(l => `[${l.severity}] ${l.emitted_at} ${l.message}`).join('\n')
        await copyToClipboard(text)
        setCopied(true)
        setTimeout(() => setCopied(false), 2000)
    }

    return (
        <div className="space-y-3">
            {/* Controls */}
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                    <div className="flex items-center gap-2">
                        <Checkbox id="auto-refresh" checked={autoRefresh} onCheckedChange={(v) => setAutoRefresh(v === true)} />
                        <Label htmlFor="auto-refresh" className="text-xs">Auto refresh (2s)</Label>
                    </div>
                    <Button size="sm" variant="ghost" onClick={() => refetch()} aria-label="Refresh logs">
                        <RefreshCw className="h-3.5 w-3.5" />
                    </Button>
                </div>
                <Button size="sm" variant="outline" onClick={handleCopy} aria-label="Copy logs">
                    {copied ? <Check className="mr-1 h-3.5 w-3.5" /> : <Copy className="mr-1 h-3.5 w-3.5" />}
                    {copied ? 'Copied' : 'Copy logs'}
                </Button>
            </div>

            {/* Log output */}
            <ScrollArea className="h-[400px] rounded-md border bg-black/50 p-3">
                {isLoading && (
                    <div className="flex items-center justify-center py-8">
                        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
                    </div>
                )}
                {!isLoading && (!logs || logs.length === 0) && (
                    <p className="text-sm text-muted-foreground text-center py-8">No logs available.</p>
                )}
                <div className="font-mono text-xs space-y-0.5">
                    {logs?.map((log) => (
                        <div key={`${log.execution_id}-${log.sequence}`} className="flex gap-2">
                            <Badge variant="outline" className={cn('shrink-0 text-[9px] px-1 py-0 font-mono', severityColor[log.severity])}>
                                {log.severity}
                            </Badge>
                            <span className="text-muted-foreground shrink-0">{new Date(log.emitted_at).toLocaleTimeString()}</span>
                            <span className="text-foreground break-all">{log.message}</span>
                        </div>
                    ))}
                    <div ref={bottomRef} />
                </div>
            </ScrollArea>
        </div>
    )
}
