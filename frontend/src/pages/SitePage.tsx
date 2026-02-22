import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { ArrowLeft, Loader2, Play, Square, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { ScrollArea } from '@/components/ui/scroll-area'
import { useHosts, useVMs, useExecutions, useCreatePlan } from '@/api/hooks'
import { formatRelativeTime, truncateId, formatBytes } from '@/lib/utils'
import type { MicroVM } from '@/api/types'
import ExecutionList from '@/components/ExecutionList'

export default function SitePage() {
    const { siteId } = useParams<{ siteId: string }>()
    const sid = siteId || ''
    const { data: hosts, isLoading: hostsLoading } = useHosts(sid)
    const { data: vms, isLoading: vmsLoading } = useVMs(sid)
    const createPlan = useCreatePlan(sid)
    const [selectedVM, setSelectedVM] = useState<MicroVM | null>(null)

    const handleVMAction = async (vm: MicroVM, op: 'START' | 'STOP' | 'DELETE') => {
        try {
            await createPlan.mutateAsync({
                idempotency_key: `${op.toLowerCase()}-${vm.id}-${Date.now()}`,
                actions: [{ operation: op, vm_id: vm.id }],
            })
        } catch {
            // handled
        }
    }

    return (
        <div className="space-y-4">
            {/* Header */}
            <div className="flex items-center gap-3">
                <Link to="/">
                    <Button variant="ghost" size="icon" aria-label="Back to dashboard">
                        <ArrowLeft className="h-4 w-4" />
                    </Button>
                </Link>
                <div>
                    <h1 className="text-lg font-semibold">Site Detail</h1>
                    <p className="text-xs font-mono text-muted-foreground">{sid}</p>
                </div>
            </div>

            <Tabs defaultValue="hosts">
                <TabsList>
                    <TabsTrigger value="hosts">Hosts</TabsTrigger>
                    <TabsTrigger value="vms">MicroVMs</TabsTrigger>
                    <TabsTrigger value="executions">Executions</TabsTrigger>
                    <TabsTrigger value="enrollment">Enrollment</TabsTrigger>
                </TabsList>

                {/* Hosts Tab */}
                <TabsContent value="hosts">
                    <Card>
                        <CardHeader><CardTitle>Hosts</CardTitle></CardHeader>
                        <CardContent>
                            {hostsLoading && <Loader2 className="h-5 w-5 animate-spin text-muted-foreground mx-auto" />}
                            {!hostsLoading && (!hosts || hosts.length === 0) && (
                                <p className="text-sm text-muted-foreground text-center py-6">No hosts enrolled for this site yet.</p>
                            )}
                            {hosts && hosts.length > 0 && (
                                <ScrollArea className="max-h-[500px]">
                                    <table className="w-full text-sm" aria-label="Hosts table">
                                        <thead>
                                            <tr className="border-b text-xs text-muted-foreground">
                                                <th className="text-left py-2 px-2 font-medium">Hostname</th>
                                                <th className="text-left py-2 px-2 font-medium">CPU</th>
                                                <th className="text-left py-2 px-2 font-medium">RAM</th>
                                                <th className="text-left py-2 px-2 font-medium hidden md:table-cell">Disk</th>
                                                <th className="text-left py-2 px-2 font-medium hidden md:table-cell">KVM</th>
                                                <th className="text-left py-2 px-2 font-medium hidden lg:table-cell">NetBird</th>
                                                <th className="text-left py-2 px-2 font-medium">State</th>
                                                <th className="text-left py-2 px-2 font-medium hidden lg:table-cell">Last Seen</th>
                                            </tr>
                                        </thead>
                                        <tbody>
                                            {hosts.map(h => (
                                                <tr key={h.id} className="border-b hover:bg-accent/50">
                                                    <td className="py-2 px-2 font-medium">{h.hostname}</td>
                                                    <td className="py-2 px-2">{h.cpu_cores_total || '—'} cores</td>
                                                    <td className="py-2 px-2">{formatBytes(h.memory_bytes_total)}</td>
                                                    <td className="py-2 px-2 hidden md:table-cell">{formatBytes(h.storage_bytes_total)}</td>
                                                    <td className="py-2 px-2 hidden md:table-cell">
                                                        <Badge variant={h.kvm_available ? 'success' : 'outline'} className="text-[10px]">
                                                            {h.kvm_available ? 'Yes' : 'No'}
                                                        </Badge>
                                                    </td>
                                                    <td className="py-2 px-2 hidden lg:table-cell">
                                                        <Badge variant={h.netbird_ready ? 'success' : 'outline'} className="text-[10px]">
                                                            {h.netbird_ready ? 'Ready' : 'N/A'}
                                                        </Badge>
                                                    </td>
                                                    <td className="py-2 px-2">
                                                        <Badge variant={h.agent_state === 'online' ? 'success' : 'outline'} className="text-[10px]">
                                                            {h.agent_state || 'unknown'}
                                                        </Badge>
                                                    </td>
                                                    <td className="py-2 px-2 hidden lg:table-cell text-xs text-muted-foreground">
                                                        {formatRelativeTime(h.last_facts_at)}
                                                    </td>
                                                </tr>
                                            ))}
                                        </tbody>
                                    </table>
                                </ScrollArea>
                            )}
                        </CardContent>
                    </Card>
                </TabsContent>

                {/* MicroVMs Tab */}
                <TabsContent value="vms">
                    <Card>
                        <CardHeader>
                            <div className="flex items-center justify-between">
                                <CardTitle>MicroVMs</CardTitle>
                                {selectedVM && (
                                    <div className="flex gap-1">
                                        <Button size="sm" variant="outline" onClick={() => handleVMAction(selectedVM, 'START')} disabled={createPlan.isPending}>
                                            <Play className="mr-1 h-3.5 w-3.5" />Start
                                        </Button>
                                        <Button size="sm" variant="outline" onClick={() => handleVMAction(selectedVM, 'STOP')} disabled={createPlan.isPending}>
                                            <Square className="mr-1 h-3.5 w-3.5" />Stop
                                        </Button>
                                        <Button size="sm" variant="destructive" onClick={() => handleVMAction(selectedVM, 'DELETE')} disabled={createPlan.isPending}>
                                            <Trash2 className="mr-1 h-3.5 w-3.5" />Delete
                                        </Button>
                                    </div>
                                )}
                            </div>
                        </CardHeader>
                        <CardContent>
                            {vmsLoading && <Loader2 className="h-5 w-5 animate-spin text-muted-foreground mx-auto" />}
                            {!vmsLoading && (!vms || vms.length === 0) && (
                                <p className="text-sm text-muted-foreground text-center py-6">No MicroVMs.</p>
                            )}
                            {vms && vms.length > 0 && (
                                <ScrollArea className="max-h-[500px]">
                                    <table className="w-full text-sm" aria-label="MicroVMs table">
                                        <thead>
                                            <tr className="border-b text-xs text-muted-foreground">
                                                <th className="text-left py-2 px-2 font-medium">ID</th>
                                                <th className="text-left py-2 px-2 font-medium">Name</th>
                                                <th className="text-left py-2 px-2 font-medium">Status</th>
                                                <th className="text-left py-2 px-2 font-medium hidden md:table-cell">vCPU</th>
                                                <th className="text-left py-2 px-2 font-medium hidden md:table-cell">Mem</th>
                                                <th className="text-left py-2 px-2 font-medium hidden lg:table-cell">Updated</th>
                                            </tr>
                                        </thead>
                                        <tbody>
                                            {vms.map(vm => {
                                                const sel = selectedVM?.id === vm.id
                                                return (
                                                    <tr key={vm.id} className={`border-b hover:bg-accent/50 cursor-pointer ${sel ? 'bg-accent' : ''}`} onClick={() => setSelectedVM(vm)}>
                                                        <td className="py-2 px-2 font-mono text-xs">{truncateId(vm.id)}</td>
                                                        <td className="py-2 px-2">{vm.name || '—'}</td>
                                                        <td className="py-2 px-2"><Badge variant="outline" className="text-[10px]">{vm.state}</Badge></td>
                                                        <td className="py-2 px-2 hidden md:table-cell">{vm.vcpu_count}</td>
                                                        <td className="py-2 px-2 hidden md:table-cell">{vm.memory_mib}M</td>
                                                        <td className="py-2 px-2 hidden lg:table-cell text-muted-foreground text-xs">{formatRelativeTime(vm.updated_at)}</td>
                                                    </tr>
                                                )
                                            })}
                                        </tbody>
                                    </table>
                                </ScrollArea>
                            )}
                        </CardContent>
                    </Card>
                </TabsContent>

                {/* Executions Tab */}
                <TabsContent value="executions">
                    <Card>
                        <CardHeader><CardTitle>Executions</CardTitle></CardHeader>
                        <CardContent>
                            <ExecutionList siteId={sid} />
                        </CardContent>
                    </Card>
                </TabsContent>

                {/* Enrollment Tab */}
                <TabsContent value="enrollment">
                    <Card>
                        <CardHeader><CardTitle>Enrollment</CardTitle></CardHeader>
                        <CardContent>
                            <p className="text-sm text-muted-foreground">
                                Use the{' '}
                                <Link to={`/enroll?siteId=${sid}`} className="text-primary underline">
                                    Enrollment Helper
                                </Link>{' '}
                                page to generate tokens and onboard agents for this site.
                            </p>
                        </CardContent>
                    </Card>
                </TabsContent>
            </Tabs>
        </div>
    )
}
