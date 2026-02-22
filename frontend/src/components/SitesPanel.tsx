import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { MapPin, Plus, Wifi, WifiOff, Loader2 } from 'lucide-react'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { useSites } from '@/api/hooks'
import { formatRelativeTime } from '@/lib/utils'
import type { Site } from '@/api/types'
import AddSiteModal from './AddSiteModal'

interface SitesPanelProps {
    activeSiteId: string | null
    onSelectSite: (site: Site) => void
}

export default function SitesPanel({ activeSiteId, onSelectSite }: SitesPanelProps) {
    const [showModal, setShowModal] = useState(false)
    const tenantId = localStorage.getItem('nkudo_tenant_id') || ''
    const { data: sites, isLoading } = useSites(tenantId)
    const navigate = useNavigate()

    const handleClick = (site: Site) => {
        onSelectSite(site)
        navigate(`/sites/${site.id}`)
    }

    return (
        <>
            <Card className="h-full flex flex-col">
                <CardHeader className="flex-row items-center justify-between space-y-0 pb-2">
                    <CardTitle>Sites</CardTitle>
                    <Button size="sm" variant="outline" onClick={() => setShowModal(true)} aria-label="Add new site">
                        <Plus className="mr-1 h-3.5 w-3.5" />
                        Add new
                    </Button>
                </CardHeader>
                <CardContent className="flex-1 min-h-0">
                    <ScrollArea className="h-full max-h-[calc(100vh-220px)]">
                        {isLoading && (
                            <div className="flex items-center justify-center py-8">
                                <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
                            </div>
                        )}
                        {!isLoading && (!sites || sites.length === 0) && (
                            <p className="text-sm text-muted-foreground py-8 text-center">
                                {tenantId ? 'No sites found.' : 'Set Tenant ID in Profile settings first.'}
                            </p>
                        )}
                        <div className="space-y-1">
                            {sites?.map((site) => {
                                const isOnline = site.connectivity_state === 'online'
                                const isActive = activeSiteId === site.id
                                return (
                                    <button
                                        key={site.id}
                                        onClick={() => handleClick(site)}
                                        className={`w-full text-left rounded-md px-3 py-2.5 transition-colors hover:bg-accent ${isActive ? 'bg-accent ring-1 ring-primary/30' : ''}`}
                                        aria-label={`Site ${site.name}`}
                                        aria-current={isActive ? 'true' : undefined}
                                    >
                                        <div className="flex items-center justify-between">
                                            <div className="flex items-center gap-2 min-w-0">
                                                <MapPin className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                                                <span className="text-sm font-medium truncate">{site.name}</span>
                                            </div>
                                            {isOnline ? (
                                                <Badge variant="success" className="shrink-0 text-[10px]">
                                                    <Wifi className="mr-1 h-3 w-3" />online
                                                </Badge>
                                            ) : (
                                                <Badge variant="outline" className="shrink-0 text-[10px]">
                                                    <WifiOff className="mr-1 h-3 w-3" />offline
                                                </Badge>
                                            )}
                                        </div>
                                        <div className="mt-1 flex items-center gap-2 text-xs text-muted-foreground">
                                            {site.location_country_code && <span>{site.location_country_code}</span>}
                                            <span>Last seen: {formatRelativeTime(site.last_heartbeat_at)}</span>
                                        </div>
                                    </button>
                                )
                            })}
                        </div>
                    </ScrollArea>
                </CardContent>
            </Card>

            <AddSiteModal open={showModal} onClose={() => setShowModal(false)} tenantId={tenantId} />
        </>
    )
}
