import { useState, useEffect } from 'react'
import { Settings, Save, CheckCircle2, XCircle } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { useHealth } from '@/api/hooks'

export default function ProfilePage() {
    const [apiUrl, setApiUrl] = useState(localStorage.getItem('nkudo_api_url') || '/api')
    const [tenantId, setTenantId] = useState(localStorage.getItem('nkudo_tenant_id') || '')
    const [adminKey, setAdminKey] = useState(localStorage.getItem('nkudo_admin_key') || '')
    const [apiKey, setApiKey] = useState(localStorage.getItem('nkudo_api_key') || '')
    const [saved, setSaved] = useState(false)
    const { data: health, isError } = useHealth()

    const handleSave = () => {
        localStorage.setItem('nkudo_api_url', apiUrl)
        localStorage.setItem('nkudo_tenant_id', tenantId)
        localStorage.setItem('nkudo_admin_key', adminKey)
        localStorage.setItem('nkudo_api_key', apiKey)
        setSaved(true)
        setTimeout(() => setSaved(false), 2000)
    }

    return (
        <div className="max-w-xl mx-auto space-y-6">
            <div className="flex items-center gap-2">
                <Settings className="h-5 w-5 text-muted-foreground" />
                <h1 className="text-lg font-semibold">Profile & Settings</h1>
            </div>

            {/* Connection Status */}
            <Card>
                <CardHeader><CardTitle>Connection Status</CardTitle></CardHeader>
                <CardContent>
                    <div className="flex items-center gap-2">
                        {health && !isError ? (
                            <>
                                <CheckCircle2 className="h-4 w-4 text-emerald-500" />
                                <span className="text-sm text-emerald-500 font-medium">Connected</span>
                                <Badge variant="success" className="text-[10px]">healthy</Badge>
                            </>
                        ) : (
                            <>
                                <XCircle className="h-4 w-4 text-destructive" />
                                <span className="text-sm text-destructive font-medium">Disconnected</span>
                                <Badge variant="destructive" className="text-[10px]">error</Badge>
                            </>
                        )}
                    </div>
                </CardContent>
            </Card>

            {/* Configuration */}
            <Card>
                <CardHeader><CardTitle>API Configuration</CardTitle></CardHeader>
                <CardContent className="space-y-4">
                    <div className="space-y-1.5">
                        <Label htmlFor="prof-api-url">API Base URL</Label>
                        <Input id="prof-api-url" value={apiUrl} onChange={(e) => setApiUrl(e.target.value)} placeholder="https://localhost:8443" className="font-mono text-xs" />
                        <p className="text-[11px] text-muted-foreground">
                            Use <code>/api</code> for dev proxy or the full URL for direct connection.
                        </p>
                    </div>

                    <Separator />

                    <div className="space-y-1.5">
                        <Label htmlFor="prof-tenant">Tenant ID</Label>
                        <Input id="prof-tenant" value={tenantId} onChange={(e) => setTenantId(e.target.value)} placeholder="UUID" className="font-mono text-xs" />
                    </div>

                    <div className="space-y-1.5">
                        <Label htmlFor="prof-admin">Admin Key (X-Admin-Key)</Label>
                        <Input id="prof-admin" type="password" value={adminKey} onChange={(e) => setAdminKey(e.target.value)} placeholder="dev-admin-key" />
                    </div>

                    <div className="space-y-1.5">
                        <Label htmlFor="prof-api">API Key (X-API-Key)</Label>
                        <Input id="prof-api" type="password" value={apiKey} onChange={(e) => setApiKey(e.target.value)} placeholder="nk_..." />
                    </div>

                    <Button onClick={handleSave} className="w-full">
                        {saved ? <CheckCircle2 className="mr-2 h-4 w-4" /> : <Save className="mr-2 h-4 w-4" />}
                        {saved ? 'Saved!' : 'Save Settings'}
                    </Button>
                </CardContent>
            </Card>
        </div>
    )
}
