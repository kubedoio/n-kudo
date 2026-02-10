'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { authStore } from '@/store/auth';
import { ApiClient } from '@/api/client';
import { Shield, Server, Key, Loader2, AlertCircle } from 'lucide-react';
import { cn } from '@/lib/utils';

export default function LoginPage() {
  const [baseUrl, setBaseUrl] = useState('https://localhost:8443');
  const [tenantId, setTenantId] = useState('');
  const [apiKey, setApiKey] = useState('');
  const [remember, setRemember] = useState(true);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const router = useRouter();

  useEffect(() => {
    // Check if already connected
    const state = authStore.getState();
    if (state.client) {
      router.push('/sites');
    }
  }, [router]);

  const handleConnect = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setError(null);

    try {
      const config = { baseUrl, tenantId, apiKey };
      const client = new ApiClient(config);

      // Verify connection
      await client.healthz();
      await client.listSites();

      authStore.setConfig(config);
      router.push('/sites');
    } catch (err: any) {
      setError(err.message || 'Failed to connect. Please check your credentials and URL.');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <div className="w-full max-w-md space-y-8 rounded-2xl bg-white p-8 shadow-xl">
        <div className="text-center">
          <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-indigo-100">
            <Shield className="h-8 w-8 text-indigo-600" />
          </div>
          <h2 className="mt-6 text-3xl font-bold tracking-tight text-slate-900">nkudo</h2>
          <p className="mt-2 text-sm text-slate-600">Connect to your control-plane</p>
        </div>

        <form className="mt-8 space-y-6" onSubmit={handleConnect}>
          <div className="space-y-4 rounded-md">
            <div>
              <label className="block text-sm font-medium text-slate-700">API Gateway URL</label>
              <div className="relative mt-1">
                <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
                  <Server className="h-5 w-5 text-slate-400" />
                </div>
                <input
                  type="url"
                  required
                  value={baseUrl}
                  onChange={(e) => setBaseUrl(e.target.value)}
                  className="block w-full rounded-lg border border-slate-300 py-2 pl-10 pr-3 shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 sm:text-sm"
                  placeholder="https://api.n-kudo.io:8443"
                />
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-slate-700">Tenant ID</label>
              <div className="relative mt-1">
                <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
                  <Shield className="h-4 w-5 text-slate-400" />
                </div>
                <input
                  type="text"
                  required
                  value={tenantId}
                  onChange={(e) => setTenantId(e.target.value)}
                  className="block w-full rounded-lg border border-slate-300 py-2 pl-10 pr-3 shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 sm:text-sm"
                  placeholder="uuid-of-tenant"
                />
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-slate-700">X-API-Key</label>
              <div className="relative mt-1">
                <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
                  <Key className="h-5 w-5 text-slate-400" />
                </div>
                <input
                  type="password"
                  required
                  value={apiKey}
                  onChange={(e) => setApiKey(e.target.value)}
                  className="block w-full rounded-lg border border-slate-300 py-2 pl-10 pr-3 shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 sm:text-sm"
                  placeholder="nk_..."
                />
              </div>
            </div>
          </div>

          <div className="flex items-center justify-between">
            <div className="flex items-center">
              <input
                id="remember"
                type="checkbox"
                checked={remember}
                onChange={(e) => setRemember(e.target.checked)}
                className="h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
              />
              <label htmlFor="remember" className="ml-2 block text-sm text-slate-900">
                Remember for this session
              </label>
            </div>
          </div>

          {error && (
            <div className="flex items-start gap-3 rounded-lg bg-red-50 p-3 text-sm text-red-700">
              <AlertCircle className="mt-0.5 h-5 w-5 shrink-0" />
              <p>{error}</p>
            </div>
          )}

          <button
            type="submit"
            disabled={isLoading}
            className="flex w-full justify-center rounded-lg bg-indigo-600 px-4 py-3 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 disabled:opacity-50"
          >
            {isLoading ? (
              <Loader2 className="h-5 w-5 animate-spin" />
            ) : (
              'Connect Dashboard'
            )}
          </button>
        </form>
      </div>
    </div>
  );
}
