'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { authStore } from '@/store/auth';
import { Sidebar } from '@/components/Sidebar';
import { Loader2 } from 'lucide-react';

export default function AuthLayout({
    children,
}: {
    children: React.ReactNode;
}) {
    const [isReady, setIsReady] = useState(false);
    const router = useRouter();

    useEffect(() => {
        // Try to load from session
        authStore.initFromSession();

        const state = authStore.getState();
        if (!state.client) {
            router.push('/');
        } else {
            setIsReady(true);
        }
    }, [router]);

    if (!isReady) {
        return (
            <div className="flex h-screen w-screen items-center justify-center bg-slate-50">
                <Loader2 className="h-8 w-8 animate-spin text-indigo-600" />
            </div>
        );
    }

    return (
        <div className="flex bg-slate-50">
            <Sidebar />
            <main className="h-screen flex-1 overflow-y-auto p-8">
                {children}
            </main>
        </div>
    );
}
