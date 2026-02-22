import { Play, Square, Trash2, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { useVMs } from '@/api/hooks'
import { formatRelativeTime, truncateId } from '@/lib/utils'
import type { MicroVM } from '@/api/types'

interface VMTableProps {
    siteId: string
    selectedVM: MicroVM | null
    onSelectVM: (vm: MicroVM) => void
    onVMAction: (vm: MicroVM, op: 'START' | 'STOP' | 'DELETE') => void
}

function stateVariant(state: string) {
    switch (state?.toUpperCase()) {
        case 'RUNNING': return 'success' as const
        case 'STOPPED': return 'secondary' as const
        case 'CREATING': case 'STARTING': return 'warning' as const
        case 'ERROR': case 'FAILED': return 'destructive' as const
        default: return 'outline' as const
    }
}

export default function VMTable({ siteId, selectedVM, onSelectVM, onVMAction }: VMTableProps) {
    const { data: vms, isLoading } = useVMs(siteId)

    if (isLoading) {
        return (
            <div className="flex items-center justify-center py-8">
                <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
            </div>
        )
    }
    if (!vms || vms.length === 0) {
        return <p className="text-sm text-muted-foreground text-center py-8">No MicroVMs found for this site.</p>
    }

    return (
        <ScrollArea className="max-h-[calc(100vh-340px)]">
            <table className="w-full text-sm" aria-label="MicroVMs table">
                <thead>
                    <tr className="border-b text-xs text-muted-foreground">
                        <th className="text-left py-2 px-2 font-medium">ID</th>
                        <th className="text-left py-2 px-2 font-medium">Name</th>
                        <th className="text-left py-2 px-2 font-medium">Status</th>
                        <th className="text-left py-2 px-2 font-medium hidden md:table-cell">vCPU</th>
                        <th className="text-left py-2 px-2 font-medium hidden md:table-cell">Mem</th>
                        <th className="text-left py-2 px-2 font-medium hidden lg:table-cell">Updated</th>
                        <th className="text-right py-2 px-2 font-medium">Actions</th>
                    </tr>
                </thead>
                <tbody>
                    {vms.map((vm) => {
                        const isSelected = selectedVM?.id === vm.id
                        return (
                            <tr
                                key={vm.id}
                                className={`border-b hover:bg-accent/50 cursor-pointer transition-colors ${isSelected ? 'bg-accent' : ''}`}
                                onClick={() => onSelectVM(vm)}
                                aria-selected={isSelected}
                            >
                                <td className="py-2 px-2 font-mono text-xs">{truncateId(vm.id)}</td>
                                <td className="py-2 px-2">{vm.name || '—'}</td>
                                <td className="py-2 px-2">
                                    <Badge variant={stateVariant(vm.state)} className="text-[10px]">{vm.state}</Badge>
                                </td>
                                <td className="py-2 px-2 hidden md:table-cell">{vm.vcpu_count || '—'}</td>
                                <td className="py-2 px-2 hidden md:table-cell">{vm.memory_mib ? `${vm.memory_mib}M` : '—'}</td>
                                <td className="py-2 px-2 hidden lg:table-cell text-muted-foreground text-xs">{formatRelativeTime(vm.updated_at)}</td>
                                <td className="py-2 px-2 text-right">
                                    <div className="flex justify-end gap-1" onClick={(e) => e.stopPropagation()}>
                                        <Button size="icon" variant="ghost" className="h-7 w-7" onClick={() => onVMAction(vm, 'START')} aria-label="Start VM">
                                            <Play className="h-3.5 w-3.5" />
                                        </Button>
                                        <Button size="icon" variant="ghost" className="h-7 w-7" onClick={() => onVMAction(vm, 'STOP')} aria-label="Stop VM">
                                            <Square className="h-3.5 w-3.5" />
                                        </Button>
                                        <Button size="icon" variant="ghost" className="h-7 w-7 text-destructive" onClick={() => onVMAction(vm, 'DELETE')} aria-label="Delete VM">
                                            <Trash2 className="h-3.5 w-3.5" />
                                        </Button>
                                    </div>
                                </td>
                            </tr>
                        )
                    })}
                </tbody>
            </table>
        </ScrollArea>
    )
}
