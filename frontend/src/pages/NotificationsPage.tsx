import { useNavigate } from 'react-router-dom'
import { Bell, ExternalLink, Loader2 } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { useExecutions } from '@/api/hooks'
import { formatRelativeTime, truncateId } from '@/lib/utils'

export default function NotificationsPage() {
    // Use a synthetic approach: show recent executions as notifications
    // In a real app this would come from an events/audit endpoint
    const tenantId = localStorage.getItem('nkudo_tenant_id') || ''
    const navigate = useNavigate()

    // We need at least one siteId—this is a simplification.
    // In practice, we'd aggregate across all sites.
    // For MVP-1, we show a notice if no site context is set.

    return (
        <div className="max-w-2xl mx-auto space-y-4">
            <div className="flex items-center gap-2">
                <Bell className="h-5 w-5 text-muted-foreground" />
                <h1 className="text-lg font-semibold">Notifications</h1>
            </div>

            <Card>
                <CardHeader><CardTitle>Recent Activity</CardTitle></CardHeader>
                <CardContent>
                    <p className="text-sm text-muted-foreground">
                        Notifications are sourced from execution events. Navigate to a site's Executions tab or the Dashboard to see recent activity.
                    </p>
                    <div className="mt-4 space-y-2">
                        <NotificationHint />
                    </div>
                </CardContent>
            </Card>
        </div>
    )
}

function NotificationHint() {
    return (
        <div className="flex items-start gap-3 rounded-md border p-3">
            <div className="h-2 w-2 mt-2 rounded-full bg-primary shrink-0" />
            <div className="flex-1">
                <p className="text-sm font-medium">Execution events</p>
                <p className="text-xs text-muted-foreground mt-0.5">
                    Visit the Dashboard and select a site to view recent executions and plans. In MVP-2, real-time notifications will be pushed via SSE.
                </p>
            </div>
            <Badge variant="outline" className="text-[10px] shrink-0">MVP-2</Badge>
        </div>
    )
}
