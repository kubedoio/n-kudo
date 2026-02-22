import { useState } from 'react'
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useCreateSite } from '@/api/hooks'
import { Loader2 } from 'lucide-react'

interface AddSiteModalProps {
    open: boolean
    onClose: () => void
    tenantId: string
}

export default function AddSiteModal({ open, onClose, tenantId }: AddSiteModalProps) {
    const [name, setName] = useState('')
    const [externalKey, setExternalKey] = useState('')
    const [countryCode, setCountryCode] = useState('')
    const createSite = useCreateSite()

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        if (!name.trim() || !tenantId) return
        try {
            await createSite.mutateAsync({
                tenantId,
                data: {
                    name: name.trim(),
                    external_key: externalKey.trim() || undefined,
                    location_country_code: countryCode.trim() || undefined,
                },
            })
            setName('')
            setExternalKey('')
            setCountryCode('')
            onClose()
        } catch {
            // error displayed by interceptor
        }
    }

    return (
        <Dialog open={open} onOpenChange={(v: boolean) => !v && onClose()}>
            <DialogContent>
                <DialogHeader>
                    <DialogTitle>Create Site</DialogTitle>
                    <DialogDescription>Add a new edge site under your tenant.</DialogDescription>
                </DialogHeader>
                <form onSubmit={handleSubmit} className="space-y-4">
                    <div className="space-y-2">
                        <Label htmlFor="site-name">Site Name *</Label>
                        <Input id="site-name" value={name} onChange={(e) => setName(e.target.value)} placeholder="e.g. edge-berlin-01" required />
                    </div>
                    <div className="space-y-2">
                        <Label htmlFor="external-key">External Key</Label>
                        <Input id="external-key" value={externalKey} onChange={(e) => setExternalKey(e.target.value)} placeholder="Optional identifier" />
                    </div>
                    <div className="space-y-2">
                        <Label htmlFor="country-code">Country Code</Label>
                        <Input id="country-code" value={countryCode} onChange={(e) => setCountryCode(e.target.value)} placeholder="DE" maxLength={2} />
                    </div>
                    <div className="flex justify-end gap-2">
                        <Button type="button" variant="outline" onClick={onClose}>Cancel</Button>
                        <Button type="submit" disabled={!name.trim() || !tenantId || createSite.isPending}>
                            {createSite.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                            Create Site
                        </Button>
                    </div>
                    {createSite.isError && (
                        <p className="text-sm text-destructive">Failed to create site. Check console for details.</p>
                    )}
                </form>
            </DialogContent>
        </Dialog>
    )
}
