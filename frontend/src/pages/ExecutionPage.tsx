import { useParams, Link } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import ExecutionLogsViewer from '@/components/ExecutionLogsViewer'

export default function ExecutionPage() {
    const { executionId } = useParams<{ executionId: string }>()
    const eid = executionId || ''

    return (
        <div className="space-y-4 max-w-4xl mx-auto">
            {/* Header */}
            <div className="flex items-center gap-3">
                <Link to="/">
                    <Button variant="ghost" size="icon" aria-label="Back to dashboard">
                        <ArrowLeft className="h-4 w-4" />
                    </Button>
                </Link>
                <div>
                    <h1 className="text-lg font-semibold">Execution Details</h1>
                    <div className="flex items-center gap-2">
                        <p className="text-xs font-mono text-muted-foreground">{eid}</p>
                        <Badge variant="outline" className="text-[10px]">Live</Badge>
                    </div>
                </div>
            </div>

            {/* Execution summary */}
            <Card>
                <CardHeader><CardTitle>Summary</CardTitle></CardHeader>
                <CardContent>
                    <div className="grid grid-cols-2 gap-4 text-sm">
                        <div>
                            <span className="text-muted-foreground">Execution ID</span>
                            <p className="font-mono text-xs mt-0.5 break-all">{eid}</p>
                        </div>
                    </div>
                </CardContent>
            </Card>

            {/* Logs */}
            <Card>
                <CardHeader><CardTitle>Logs</CardTitle></CardHeader>
                <CardContent>
                    <ExecutionLogsViewer executionId={eid} />
                </CardContent>
            </Card>
        </div>
    )
}
