import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import VMTable from './VMTable'
import ExecutionList from './ExecutionList'
import type { MicroVM } from '@/api/types'

interface LivePanelProps {
    siteId: string | null
    selectedVM: MicroVM | null
    onSelectVM: (vm: MicroVM) => void
    onVMAction: (vm: MicroVM, op: 'START' | 'STOP' | 'DELETE') => void
}

export default function LivePanel({ siteId, selectedVM, onSelectVM, onVMAction }: LivePanelProps) {
    if (!siteId) {
        return (
            <Card className="h-full flex items-center justify-center">
                <p className="text-sm text-muted-foreground">Select a site to view live data.</p>
            </Card>
        )
    }

    return (
        <Card className="h-full flex flex-col">
            <CardHeader className="pb-2">
                <CardTitle>Live</CardTitle>
            </CardHeader>
            <CardContent className="flex-1 min-h-0">
                <Tabs defaultValue="vms">
                    <TabsList>
                        <TabsTrigger value="vms">MicroVMs</TabsTrigger>
                        <TabsTrigger value="executions">Recent Executions</TabsTrigger>
                    </TabsList>
                    <TabsContent value="vms">
                        <VMTable siteId={siteId} selectedVM={selectedVM} onSelectVM={onSelectVM} onVMAction={onVMAction} />
                    </TabsContent>
                    <TabsContent value="executions">
                        <ExecutionList siteId={siteId} />
                    </TabsContent>
                </Tabs>
            </CardContent>
        </Card>
    )
}
