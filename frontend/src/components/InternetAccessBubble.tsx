import { Globe } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'

export default function InternetAccessBubble() {
    return (
        <Card className="border-dashed">
            <CardContent className="p-4 text-center">
                <Globe className="h-8 w-8 mx-auto text-muted-foreground mb-2" />
                <p className="text-xs font-semibold uppercase text-muted-foreground">Internet Public Access</p>
                <p className="text-xs text-muted-foreground mt-1">N/A (metrics in MVP-2)</p>
            </CardContent>
        </Card>
    )
}
