import { useState } from 'react'
import SitesPanel from '@/components/SitesPanel'
import ActionsPanel from '@/components/ActionsPanel'
import LivePanel from '@/components/LivePanel'
import InternetAccessBubble from '@/components/InternetAccessBubble'
import VpnStatusBubble from '@/components/VpnStatusBubble'
import { useCreatePlan } from '@/api/hooks'
import type { Site, MicroVM } from '@/api/types'

export default function DashboardPage() {
    const [activeSite, setActiveSite] = useState<Site | null>(null)
    const [selectedVM, setSelectedVM] = useState<MicroVM | null>(null)
    const createPlan = useCreatePlan(activeSite?.id || '')

    const handleVMAction = async (vm: MicroVM, op: 'START' | 'STOP' | 'DELETE') => {
        if (!activeSite) return
        try {
            await createPlan.mutateAsync({
                idempotency_key: `${op.toLowerCase()}-${vm.id}-${Date.now()}`,
                actions: [{ operation: op, vm_id: vm.id }],
            })
        } catch {
            // handled by interceptor
        }
    }

    return (
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-[280px_220px_1fr_160px] xl:grid-cols-[300px_240px_1fr_180px]">
            {/* Left: Sites */}
            <div className="min-h-[400px]">
                <SitesPanel activeSiteId={activeSite?.id ?? null} onSelectSite={setActiveSite} />
            </div>

            {/* Middle: Actions */}
            <div className="min-h-[400px]">
                <ActionsPanel siteId={activeSite?.id ?? null} selectedVM={selectedVM} />
            </div>

            {/* Main: Live */}
            <div className="min-h-[400px]">
                <LivePanel
                    siteId={activeSite?.id ?? null}
                    selectedVM={selectedVM}
                    onSelectVM={setSelectedVM}
                    onVMAction={handleVMAction}
                />
            </div>

            {/* Right: Bubbles */}
            <div className="space-y-4">
                <InternetAccessBubble />
                <VpnStatusBubble siteId={activeSite?.id ?? null} />
            </div>
        </div>
    )
}
