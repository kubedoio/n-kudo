import { useState } from 'react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Loader2 } from 'lucide-react'
import { useCreatePlan } from '@/api/hooks'
import type { PlanAction } from '@/api/types'

interface CreateMicroVMFormProps {
    siteId: string
    onCreated?: () => void
}

export default function CreateMicroVMForm({ siteId, onCreated }: CreateMicroVMFormProps) {
    const [name, setName] = useState('')
    const [vcpu, setVcpu] = useState('1')
    const [memory, setMemory] = useState('512')
    const [rootfs, setRootfs] = useState('/var/lib/nkudo-edge/images/rootfs.ext4')
    const [bridge, setBridge] = useState('br0')
    const [startAfter, setStartAfter] = useState(true)
    const createPlan = useCreatePlan(siteId)

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        if (!name.trim()) return

        const actions: PlanAction[] = [
            {
                operation: 'CREATE',
                name: name.trim(),
                vcpu_count: parseInt(vcpu) || 1,
                memory_mib: parseInt(memory) || 512,
                rootfs_path: rootfs.trim(),
                bridge: bridge.trim() || 'br0',
            },
        ]
        if (startAfter) {
            actions.push({ operation: 'START', name: name.trim() })
        }

        try {
            await createPlan.mutateAsync({
                idempotency_key: `create-${name.trim()}-${Date.now()}`,
                actions,
            })
            setName('')
            onCreated?.()
        } catch {
            // error handled by interceptor
        }
    }

    return (
        <form onSubmit={handleSubmit} className="space-y-3">
            <div className="space-y-1.5">
                <Label htmlFor="vm-name">Name *</Label>
                <Input id="vm-name" value={name} onChange={(e) => setName(e.target.value)} placeholder="my-vm-01" required />
            </div>
            <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                    <Label htmlFor="vm-vcpu">vCPU</Label>
                    <Input id="vm-vcpu" type="number" min="1" max="16" value={vcpu} onChange={(e) => setVcpu(e.target.value)} />
                </div>
                <div className="space-y-1.5">
                    <Label htmlFor="vm-memory">Memory (MiB)</Label>
                    <Input id="vm-memory" type="number" min="128" value={memory} onChange={(e) => setMemory(e.target.value)} />
                </div>
            </div>
            <div className="space-y-1.5">
                <Label htmlFor="vm-rootfs">RootFS Path</Label>
                <Input id="vm-rootfs" value={rootfs} onChange={(e) => setRootfs(e.target.value)} className="font-mono text-xs" />
            </div>
            <div className="space-y-1.5">
                <Label htmlFor="vm-bridge">Bridge</Label>
                <Input id="vm-bridge" value={bridge} onChange={(e) => setBridge(e.target.value)} />
            </div>
            <div className="flex items-center gap-2">
                <Checkbox id="start-after" checked={startAfter} onCheckedChange={(v) => setStartAfter(v === true)} />
                <Label htmlFor="start-after" className="text-xs">Start after create</Label>
            </div>
            <Button type="submit" className="w-full" disabled={!name.trim() || !siteId || createPlan.isPending}>
                {createPlan.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                Submit Plan
            </Button>
            {createPlan.isError && <p className="text-sm text-destructive">Plan submission failed.</p>}
            {createPlan.isSuccess && <p className="text-sm text-emerald-500">Plan submitted ✓</p>}
        </form>
    )
}
