import { useState } from 'react'
import { Box, ChevronDown, Play, Square, Trash2 } from 'lucide-react'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Popover, PopoverTrigger, PopoverContent } from '@/components/ui/popover'
import { Separator } from '@/components/ui/separator'
import CreateMicroVMForm from './CreateMicroVMForm'
import { useCreatePlan } from '@/api/hooks'
import type { MicroVM } from '@/api/types'

interface ActionsPanelProps {
    siteId: string | null
    selectedVM: MicroVM | null
}

export default function ActionsPanel({ siteId, selectedVM }: ActionsPanelProps) {
    const [popoverOpen, setPopoverOpen] = useState(false)
    const createPlan = useCreatePlan(siteId || '')

    const handleVMAction = async (operation: 'START' | 'STOP' | 'DELETE') => {
        if (!siteId || !selectedVM) return
        try {
            await createPlan.mutateAsync({
                idempotency_key: `${operation.toLowerCase()}-${selectedVM.id}-${Date.now()}`,
                actions: [{ operation, vm_id: selectedVM.id }],
            })
        } catch {
            // error handled by interceptor
        }
    }

    return (
        <Card className="h-full flex flex-col">
            <CardHeader className="pb-2">
                <CardTitle>Actions</CardTitle>
            </CardHeader>
            <CardContent className="flex-1 space-y-4">
                {/* MicroVM Create */}
                <div>
                    <h4 className="text-xs font-semibold text-muted-foreground uppercase mb-2">MicroVM</h4>
                    <div className="flex gap-2">
                        <Popover open={popoverOpen} onOpenChange={setPopoverOpen}>
                            <PopoverTrigger asChild>
                                <Button size="sm" disabled={!siteId} className="flex-1">
                                    <Box className="mr-1.5 h-3.5 w-3.5" />
                                    Create MicroVM
                                    <ChevronDown className="ml-1.5 h-3 w-3" />
                                </Button>
                            </PopoverTrigger>
                            <PopoverContent className="w-80" align="start">
                                <CreateMicroVMForm siteId={siteId || ''} onCreated={() => setPopoverOpen(false)} />
                            </PopoverContent>
                        </Popover>
                    </div>
                </div>

                <Separator />

                {/* VM Operations */}
                <div>
                    <h4 className="text-xs font-semibold text-muted-foreground uppercase mb-2">
                        MicroVM Operations
                    </h4>
                    {selectedVM ? (
                        <div className="space-y-2">
                            <p className="text-xs text-muted-foreground truncate">
                                Selected: <span className="font-mono text-foreground">{selectedVM.name || selectedVM.id}</span>
                            </p>
                            <div className="flex gap-2">
                                <Button size="sm" variant="outline" onClick={() => handleVMAction('START')} disabled={createPlan.isPending}>
                                    <Play className="mr-1 h-3.5 w-3.5" />Start
                                </Button>
                                <Button size="sm" variant="outline" onClick={() => handleVMAction('STOP')} disabled={createPlan.isPending}>
                                    <Square className="mr-1 h-3.5 w-3.5" />Stop
                                </Button>
                                <Button size="sm" variant="destructive" onClick={() => handleVMAction('DELETE')} disabled={createPlan.isPending}>
                                    <Trash2 className="mr-1 h-3.5 w-3.5" />Delete
                                </Button>
                            </div>
                        </div>
                    ) : (
                        <p className="text-xs text-muted-foreground">
                            {siteId ? 'Select a VM in the Live panel to see operations.' : 'Select a site first.'}
                        </p>
                    )}
                </div>
            </CardContent>
        </Card>
    )
}
