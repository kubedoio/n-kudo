import { Shield, ShieldCheck, ShieldAlert, ShieldOff } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { useHosts } from '@/api/hooks'

interface VpnStatusBubbleProps {
    siteId: string | null
}

type VpnState = 'ready' | 'degraded' | 'not_configured' | 'unknown'

function deriveVpnState(hosts: { netbird_ready?: boolean }[]): VpnState {
    if (!hosts || hosts.length === 0) return 'not_configured'
    const readyCount = hosts.filter(h => h.netbird_ready === true).length
    if (readyCount === hosts.length) return 'ready'
    if (readyCount > 0) return 'degraded'
    return 'not_configured'
}

const icons: Record<VpnState, typeof Shield> = {
    ready: ShieldCheck,
    degraded: ShieldAlert,
    not_configured: ShieldOff,
    unknown: Shield,
}

const labels: Record<VpnState, string> = {
    ready: 'Ready',
    degraded: 'Degraded',
    not_configured: 'Not configured',
    unknown: 'Unknown',
}

const variants: Record<VpnState, 'success' | 'warning' | 'outline' | 'secondary'> = {
    ready: 'success',
    degraded: 'warning',
    not_configured: 'outline',
    unknown: 'secondary',
}

export default function VpnStatusBubble({ siteId }: VpnStatusBubbleProps) {
    const { data: hosts } = useHosts(siteId || '')
    const state: VpnState = siteId ? deriveVpnState(hosts || []) : 'unknown'
    const Icon = icons[state]

    return (
        <Card className="border-dashed">
            <CardContent className="p-4 text-center">
                <Icon className="h-8 w-8 mx-auto text-muted-foreground mb-2" />
                <p className="text-xs font-semibold uppercase text-muted-foreground">VPN (NetBird)</p>
                <Badge variant={variants[state]} className="mt-1 text-[10px]">{labels[state]}</Badge>
            </CardContent>
        </Card>
    )
}
