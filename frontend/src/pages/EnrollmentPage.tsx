import { useState, useEffect } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Key, Copy, Check, Loader2, CheckCircle2, Clock } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { useCreateEnrollmentToken, useHosts } from '@/api/hooks'
import { copyToClipboard } from '@/lib/utils'

export default function EnrollmentPage() {
    const [searchParams] = useSearchParams()
    const [tenantId, setTenantId] = useState(localStorage.getItem('nkudo_tenant_id') || '')
    const [siteId, setSiteId] = useState(searchParams.get('siteId') || '')
    const [ttl, setTtl] = useState('900')
    const [token, setToken] = useState('')
    const [copied, setCopied] = useState<string | null>(null)
    const createToken = useCreateEnrollmentToken(tenantId)

    const baseUrl = localStorage.getItem('nkudo_api_url') || 'https://localhost:8443'
    const { data: hosts } = useHosts(siteId)
    const hasOnlineHost = hosts?.some(h => h.agent_state === 'online')

    useEffect(() => {
        const stored = localStorage.getItem('nkudo_tenant_id')
        if (stored && !tenantId) setTenantId(stored)
    }, [])

    const handleGenerate = async () => {
        if (!tenantId || !siteId) return
        try {
            const result = await createToken.mutateAsync({
                site_id: siteId,
                expires_in_seconds: parseInt(ttl) || 900,
            })
            setToken(result.token)
        } catch {
            // error handled
        }
    }

    const handleCopy = async (text: string, label: string) => {
        await copyToClipboard(text)
        setCopied(label)
        setTimeout(() => setCopied(null), 2000)
    }

    const enrollCmd = `sudo ./bin/edge enroll --control-plane ${baseUrl} --token ${token || '<TOKEN>'}`
    const runCmd = `sudo ./bin/edge run --control-plane ${baseUrl}`

    return (
        <div className="max-w-2xl mx-auto space-y-6">
            <div>
                <h1 className="text-lg font-semibold">Enrollment Helper</h1>
                <p className="text-sm text-muted-foreground">Generate enrollment tokens and onboard edge agents.</p>
            </div>

            {/* Step 1: Config */}
            <Card>
                <CardHeader><CardTitle>1. Configuration</CardTitle></CardHeader>
                <CardContent className="space-y-3">
                    <div className="space-y-1.5">
                        <Label htmlFor="enroll-tenant">Tenant ID</Label>
                        <Input id="enroll-tenant" value={tenantId} onChange={(e) => setTenantId(e.target.value)} placeholder="UUID" className="font-mono text-xs" />
                    </div>
                    <div className="space-y-1.5">
                        <Label htmlFor="enroll-site">Site ID</Label>
                        <Input id="enroll-site" value={siteId} onChange={(e) => setSiteId(e.target.value)} placeholder="UUID" className="font-mono text-xs" />
                    </div>
                    <div className="space-y-1.5">
                        <Label htmlFor="enroll-ttl">Token TTL (seconds)</Label>
                        <Input id="enroll-ttl" type="number" value={ttl} onChange={(e) => setTtl(e.target.value)} />
                    </div>
                </CardContent>
            </Card>

            {/* Step 2: Generate Token */}
            <Card>
                <CardHeader><CardTitle>2. Generate Enrollment Token</CardTitle></CardHeader>
                <CardContent className="space-y-3">
                    <Button onClick={handleGenerate} disabled={!tenantId || !siteId || createToken.isPending}>
                        {createToken.isPending ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Key className="mr-2 h-4 w-4" />}
                        Create Enrollment Token
                    </Button>
                    {createToken.isError && <p className="text-sm text-destructive">Failed to create token.</p>}
                    {token && (
                        <div className="rounded-md border bg-muted/50 p-3 space-y-2">
                            <div className="flex items-center justify-between">
                                <span className="text-xs font-semibold text-muted-foreground">Token</span>
                                <Button size="sm" variant="ghost" onClick={() => handleCopy(token, 'token')}>
                                    {copied === 'token' ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
                                </Button>
                            </div>
                            <pre className="text-xs font-mono break-all select-all">{token}</pre>
                        </div>
                    )}
                </CardContent>
            </Card>

            {/* Step 3: Commands */}
            <Card>
                <CardHeader><CardTitle>3. Run on Edge Host</CardTitle></CardHeader>
                <CardContent className="space-y-3">
                    <div className="rounded-md border bg-muted/50 p-3">
                        <div className="flex items-center justify-between mb-1">
                            <span className="text-xs font-semibold text-muted-foreground">Enroll</span>
                            <Button size="sm" variant="ghost" onClick={() => handleCopy(enrollCmd, 'enroll')}>
                                {copied === 'enroll' ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
                            </Button>
                        </div>
                        <pre className="text-xs font-mono break-all select-all">{enrollCmd}</pre>
                    </div>
                    <div className="rounded-md border bg-muted/50 p-3">
                        <div className="flex items-center justify-between mb-1">
                            <span className="text-xs font-semibold text-muted-foreground">Run Agent</span>
                            <Button size="sm" variant="ghost" onClick={() => handleCopy(runCmd, 'run')}>
                                {copied === 'run' ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
                            </Button>
                        </div>
                        <pre className="text-xs font-mono break-all select-all">{runCmd}</pre>
                    </div>
                </CardContent>
            </Card>

            {/* Step 4: Wait for heartbeat */}
            <Card>
                <CardHeader><CardTitle>4. Wait for Heartbeat</CardTitle></CardHeader>
                <CardContent>
                    {!siteId ? (
                        <p className="text-sm text-muted-foreground">Enter a Site ID above to monitor.</p>
                    ) : hasOnlineHost ? (
                        <div className="flex items-center gap-2 text-emerald-500">
                            <CheckCircle2 className="h-5 w-5" />
                            <span className="text-sm font-medium">Agent online! First host detected.</span>
                        </div>
                    ) : (
                        <div className="flex items-center gap-2 text-muted-foreground">
                            <Clock className="h-5 w-5 animate-pulse" />
                            <span className="text-sm">Waiting for first heartbeat…</span>
                            <Badge variant="outline" className="text-[10px]">polling every 10s</Badge>
                        </div>
                    )}
                </CardContent>
            </Card>
        </div>
    )
}
