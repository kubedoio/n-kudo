import { useNavigate } from 'react-router-dom'
import { Loader2, ExternalLink } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { useExecutions } from '@/api/hooks'
import { formatRelativeTime, truncateId } from '@/lib/utils'

interface ExecutionListProps {
    siteId: string
}

function statusVariant(state: string) {
    switch (state?.toUpperCase()) {
        case 'SUCCEEDED': return 'success' as const
        case 'FAILED': return 'destructive' as const
        case 'IN_PROGRESS': case 'PENDING': return 'warning' as const
        default: return 'outline' as const
    }
}

export default function ExecutionList({ siteId }: ExecutionListProps) {
    const { data: executions, isLoading } = useExecutions(siteId)
    const navigate = useNavigate()

    if (isLoading) {
        return (
            <div className="flex items-center justify-center py-8">
                <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
            </div>
        )
    }
    if (!executions || executions.length === 0) {
        return <p className="text-sm text-muted-foreground text-center py-8">No recent executions.</p>
    }

    return (
        <ScrollArea className="max-h-[calc(100vh-340px)]">
            <table className="w-full text-sm" aria-label="Executions table">
                <thead>
                    <tr className="border-b text-xs text-muted-foreground">
                        <th className="text-left py-2 px-2 font-medium">Execution ID</th>
                        <th className="text-left py-2 px-2 font-medium">Operation</th>
                        <th className="text-left py-2 px-2 font-medium">Status</th>
                        <th className="text-left py-2 px-2 font-medium hidden md:table-cell">VM</th>
                        <th className="text-right py-2 px-2 font-medium"></th>
                    </tr>
                </thead>
                <tbody>
                    {executions.map((exec) => (
                        <tr
                            key={exec.id}
                            className="border-b hover:bg-accent/50 cursor-pointer transition-colors"
                            onClick={() => navigate(`/executions/${exec.id}`)}
                        >
                            <td className="py-2 px-2 font-mono text-xs">{truncateId(exec.id)}</td>
                            <td className="py-2 px-2">{exec.operation_type || '—'}</td>
                            <td className="py-2 px-2">
                                <Badge variant={statusVariant(exec.state)} className="text-[10px]">{exec.state}</Badge>
                            </td>
                            <td className="py-2 px-2 hidden md:table-cell font-mono text-xs">{truncateId(exec.vm_id)}</td>
                            <td className="py-2 px-2 text-right">
                                <ExternalLink className="inline h-3.5 w-3.5 text-muted-foreground" />
                            </td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </ScrollArea>
    )
}
