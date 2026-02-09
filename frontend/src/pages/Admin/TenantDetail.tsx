import { useState } from 'react';
import { useParams, useNavigate, useSearchParams } from 'react-router-dom';
import {
  Building2,
  ArrowLeft,
  MapPin,
  Key,
  Plus,
  Copy,
  Check,
  Globe,
  Calendar,
  Clock,
  Trash2,
  ExternalLink,
  Shield,
  AlertTriangle,
} from 'lucide-react';
import {
  Card,
  Table,
  Button,
  Badge,
  PageLoader,
  EmptyState,
  toast,
} from '@/components/common';
import type { TableColumn } from '@/components/common';
import { useTenant, useAPIKeys, useRevokeAPIKey, useEnrollmentTokens } from '@/api/hooks';
import { useSites } from '@/api/hooks';
import { Site, APIKey, EnrollmentToken } from '@/api/types';
import { IssueTokenModal } from './IssueTokenModal';
import { CreateAPIKeyModal } from './CreateAPIKeyModal';



type TabType = 'sites' | 'apikeys' | 'tokens';

export function TenantDetail() {
  const { tenantId } = useParams<{ tenantId: string }>();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const [activeTab, setActiveTab] = useState<TabType>(
    (searchParams.get('tab') as TabType) || 'sites'
  );
  const [isTokenModalOpen, setIsTokenModalOpen] = useState(false);
  const [isAPIKeyModalOpen, setIsAPIKeyModalOpen] = useState(false);
  const [copiedTokenId, setCopiedTokenId] = useState<string | null>(null);

  // Fetch tenant data
  const { data: tenant, isLoading: isLoadingTenant, error: tenantError } = useTenant(tenantId || '');

  // Fetch sites for this tenant
  const { data: sites, isLoading: isLoadingSites } = useSites(tenantId || '');

  // Fetch API keys for this tenant
  const { data: apiKeys, isLoading: isLoadingAPIKeys } = useAPIKeys(tenantId || '');

  // Fetch enrollment tokens for this tenant
  const { data: enrollmentTokens, isLoading: isLoadingTokens } = useEnrollmentTokens(tenantId || '');

  // Handle tab change
  const handleTabChange = (tab: TabType) => {
    setActiveTab(tab);
    setSearchParams({ tab });
  };

  // Copy token to clipboard
  const handleCopyToken = async (tokenId: string, token: string) => {
    try {
      await navigator.clipboard.writeText(token);
      setCopiedTokenId(tokenId);
      toast.success('Token copied to clipboard');
      setTimeout(() => setCopiedTokenId(null), 2000);
    } catch {
      toast.error('Failed to copy token');
    }
  };

  // Handle errors
  if (tenantError) {
    toast.error(`Failed to load tenant: ${tenantError.message}`);
  }

  if (isLoadingTenant) {
    return (
      <div className="flex min-h-[400px] items-center justify-center">
        <PageLoader message="Loading tenant..." />
      </div>
    );
  }

  if (!tenant) {
    return (
      <div className="flex min-h-[400px] flex-col items-center justify-center">
        <EmptyState
          title="Tenant not found"
          description="The tenant you are looking for does not exist or you don't have access to it."
          icon={<Building2 className="h-12 w-12 text-gray-400" />}
          action={{
            label: 'Back to Tenants',
            onClick: () => navigate('/admin/tenants'),
          }}
        />
      </div>
    );
  }

  const tabs = [
    { id: 'sites' as const, label: 'Sites', count: sites?.length || 0 },
    { id: 'apikeys' as const, label: 'API Keys', count: apiKeys?.length || 0 },
    { id: 'tokens' as const, label: 'Token History', count: enrollmentTokens?.length || 0 },
  ];

  return (
    <div className="space-y-6">
      {/* Back button */}
      <Button
        variant="ghost"
        size="sm"
        leftIcon={<ArrowLeft className="h-4 w-4" />}
        onClick={() => navigate('/admin/tenants')}
        className="-ml-2"
      >
        Back to Tenants
      </Button>

      {/* Tenant Info Card */}
      <Card>
        <div className="flex flex-col gap-6 lg:flex-row lg:items-start lg:justify-between">
          <div className="flex items-start gap-4">
            <div className="flex h-16 w-16 items-center justify-center rounded-xl bg-blue-100">
              <Building2 className="h-8 w-8 text-blue-600" />
            </div>
            <div>
              <h1 className="text-2xl font-bold text-gray-900">{tenant.name}</h1>
              <div className="mt-1 flex flex-wrap items-center gap-3 text-sm text-gray-500">
                <code className="rounded bg-gray-100 px-2 py-1">{tenant.slug}</code>
                <span>•</span>
                <span className="flex items-center gap-1">
                  <Globe className="h-4 w-4" />
                  {tenant.primary_region || 'No region set'}
                </span>
                <span>•</span>
                <span className="flex items-center gap-1">
                  <Calendar className="h-4 w-4" />
                  Created {new Date(tenant.created_at).toLocaleDateString()}
                </span>
              </div>
            </div>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button
              variant="secondary"
              leftIcon={<Key className="h-4 w-4" />}
              onClick={() => handleTabChange('apikeys')}
            >
              Manage API Keys
            </Button>
            <Button
              variant="primary"
              leftIcon={<Plus className="h-4 w-4" />}
              onClick={() => setIsTokenModalOpen(true)}
            >
              Issue Enrollment Token
            </Button>
          </div>
        </div>

        {/* Stats */}
        <div className="mt-6 grid gap-4 border-t border-gray-200 pt-6 sm:grid-cols-3">
          <div>
            <p className="text-sm text-gray-500">Data Retention</p>
            <p className="mt-1 text-lg font-semibold">{tenant.data_retention_days} days</p>
          </div>
          <div>
            <p className="text-sm text-gray-500">Sites</p>
            <p className="mt-1 text-lg font-semibold">{sites?.length || 0}</p>
          </div>
          <div>
            <p className="text-sm text-gray-500">Status</p>
            <div className="mt-1">
              <Badge variant="success" size="sm">
                Active
              </Badge>
            </div>
          </div>
        </div>
      </Card>

      {/* Tabs */}
      <div className="border-b border-gray-200">
        <nav className="-mb-px flex space-x-8">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => handleTabChange(tab.id)}
              data-testid={`tab-${tab.id}`}
              className={`
                whitespace-nowrap border-b-2 py-4 px-1 text-sm font-medium
                ${
                  activeTab === tab.id
                    ? 'border-blue-500 text-blue-600'
                    : 'border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700'
                }
              `}
            >
              {tab.label}
              <span
                className={`
                ml-2 rounded-full px-2.5 py-0.5 text-xs
                ${
                  activeTab === tab.id
                    ? 'bg-blue-100 text-blue-600'
                    : 'bg-gray-100 text-gray-600'
                }
              `}
              >
                {tab.count}
              </span>
            </button>
          ))}
        </nav>
      </div>

      {/* Tab Content */}
      <div className="min-h-[300px]">
        {activeTab === 'sites' && (
          <SitesTab tenantId={tenantId || ''} sites={sites || []} isLoading={isLoadingSites} />
        )}
        {activeTab === 'apikeys' && (
          <APIKeysTab
            apiKeys={apiKeys || []}
            tenantId={tenantId || ''}
            isLoading={isLoadingAPIKeys}
            onCreateClick={() => setIsAPIKeyModalOpen(true)}
          />
        )}
        {activeTab === 'tokens' && (
          <TokenHistoryTab
            tokenHistory={enrollmentTokens || []}
            isLoading={isLoadingTokens}
            copiedTokenId={copiedTokenId}
            onCopyToken={handleCopyToken}
          />
        )}
      </div>

      {/* Issue Token Modal */}
      <IssueTokenModal
        isOpen={isTokenModalOpen}
        onClose={() => setIsTokenModalOpen(false)}
        tenantId={tenantId || ''}
        sites={sites || []}
      />

      {/* Create API Key Modal */}
      <CreateAPIKeyModal
        isOpen={isAPIKeyModalOpen}
        onClose={() => setIsAPIKeyModalOpen(false)}
        tenantId={tenantId || ''}
      />
    </div>
  );
}

// Sites Tab Component
interface SitesTabProps {
  tenantId: string;
  sites: Site[];
  isLoading: boolean;
}

function SitesTab({ tenantId, sites, isLoading }: SitesTabProps) {
  const navigate = useNavigate();

  const siteColumns: TableColumn<Site>[] = [
    {
      key: 'name',
      title: 'Name',
      sortable: true,
      render: (_value: unknown, item: Site) => (
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-green-100">
            <MapPin className="h-5 w-5 text-green-600" />
          </div>
          <div>
            <p className="font-medium text-gray-900">{item.name}</p>
            <p className="text-sm text-gray-500">{item.external_key || item.id.slice(0, 8)}</p>
          </div>
        </div>
      ),
    },
    {
      key: 'location_country_code',
      title: 'Location',
      sortable: true,
      render: (_value: unknown, item: Site) =>
        item.location_country_code ? (
          <Badge variant="default" size="sm">
            {item.location_country_code.toUpperCase()}
          </Badge>
        ) : (
          <span className="text-gray-400">-</span>
        ),
    },
    {
      key: 'connectivity_state',
      title: 'Status',
      sortable: true,
      render: (_value: unknown, item: Site) => {
        const statusMap: Record<string, { variant: 'success' | 'warning' | 'error'; label: string }> = {
          CONNECTED: { variant: 'success', label: 'Connected' },
          DISCONNECTED: { variant: 'warning', label: 'Disconnected' },
          PENDING: { variant: 'warning', label: 'Pending' },
          ERROR: { variant: 'error', label: 'Error' },
        };
        const status = statusMap[item.connectivity_state] || { variant: 'default', label: item.connectivity_state };
        return (
          <Badge variant={status.variant} size="sm">
            {status.label}
          </Badge>
        );
      },
    },
    {
      key: 'last_heartbeat_at',
      title: 'Last Heartbeat',
      sortable: true,
      render: (_value: unknown, item: Site) =>
        item.last_heartbeat_at ? (
          <span className="flex items-center gap-1 text-sm text-gray-500">
            <Clock className="h-3.5 w-3.5" />
            {new Date(item.last_heartbeat_at).toLocaleString()}
          </span>
        ) : (
          <span className="text-gray-400">Never</span>
        ),
    },
    {
      key: 'id',
      title: 'Actions',
      align: 'right',
      render: (_value: unknown, item: Site) => (
        <Button
          variant="ghost"
          size="sm"
          leftIcon={<ExternalLink className="h-4 w-4" />}
          onClick={(e) => {
            e.stopPropagation();
            navigate(`/tenant/${tenantId}/sites/${item.id}`);
          }}
        >
          View
        </Button>
      ),
    },
  ];

  if (isLoading) {
    return <PageLoader message="Loading sites..." />;
  }

  if (sites.length === 0) {
    return (
      <EmptyState
        title="No sites yet"
        description="Sites are created when edge agents enroll using an enrollment token."
        icon={<MapPin className="h-12 w-12 text-gray-400" />}
      />
    );
  }

  return (
    <Table
      columns={siteColumns}
      data={sites}
      keyExtractor={(item) => item.id}
      hoverable
      onRowClick={(item) => navigate(`/tenant/${tenantId}/sites/${item.id}`)}
    />
  );
}

// API Keys Tab Component
interface APIKeysTabProps {
  apiKeys: APIKey[];
  tenantId: string;
  isLoading: boolean;
  onCreateClick: () => void;
}

function APIKeysTab({ apiKeys, tenantId, isLoading, onCreateClick }: APIKeysTabProps) {
  const revokeAPIKey = useRevokeAPIKey();
  const [confirmingKeyId, setConfirmingKeyId] = useState<string | null>(null);

  const handleRevoke = async (keyId: string) => {
    if (confirmingKeyId !== keyId) {
      setConfirmingKeyId(keyId);
      return;
    }

    try {
      await revokeAPIKey.mutateAsync({ tenantId, keyId });
      toast.success('API key revoked successfully');
      setConfirmingKeyId(null);
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to revoke API key';
      toast.error(message);
    }
  };

  const handleCancelRevoke = () => {
    setConfirmingKeyId(null);
  };

  const keyColumns: TableColumn<APIKey>[] = [
    {
      key: 'name',
      title: 'Name',
      sortable: true,
      render: (_value: unknown, item: APIKey) => (
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-purple-100">
            <Shield className="h-5 w-5 text-purple-600" />
          </div>
          <span className="font-medium text-gray-900">{item.name}</span>
        </div>
      ),
    },
    {
      key: 'created_at',
      title: 'Created',
      sortable: true,
      render: (_value: unknown, item: APIKey) => new Date(item.created_at).toLocaleDateString(),
    },
    {
      key: 'expires_at',
      title: 'Expires',
      sortable: true,
      render: (_value: unknown, item: APIKey) =>
        item.expires_at ? (
          new Date(item.expires_at) < new Date() ? (
            <Badge variant="error" size="sm">Expired</Badge>
          ) : (
            new Date(item.expires_at).toLocaleDateString()
          )
        ) : (
          <span className="text-gray-400">Never</span>
        ),
    },
    {
      key: 'last_used_at',
      title: 'Last Used',
      sortable: true,
      render: (_value: unknown, item: APIKey) =>
        item.last_used_at ? (
          <span className="text-sm text-gray-500">{new Date(item.last_used_at).toLocaleString()}</span>
        ) : (
          <span className="text-gray-400">Never</span>
        ),
    },
    {
      key: 'id',
      title: 'Actions',
      align: 'right',
      render: (_value: unknown, item: APIKey) => {
        const isConfirming = confirmingKeyId === item.id;
        return isConfirming ? (
          <div className="flex items-center gap-2">
            <span className="text-sm text-red-600 flex items-center gap-1">
              <AlertTriangle className="h-4 w-4" />
              Are you sure?
            </span>
            <Button
              variant="ghost"
              size="sm"
              onClick={handleCancelRevoke}
              disabled={revokeAPIKey.isPending}
            >
              Cancel
            </Button>
            <Button
              variant="ghost"
              size="sm"
              className="text-red-600 hover:text-red-700"
              leftIcon={<Trash2 className="h-4 w-4" />}
              onClick={() => handleRevoke(item.id)}
              loading={revokeAPIKey.isPending}
            >
              Revoke
            </Button>
          </div>
        ) : (
          <Button
            variant="ghost"
            size="sm"
            leftIcon={<Trash2 className="h-4 w-4 text-red-500" />}
            onClick={() => handleRevoke(item.id)}
            data-testid={`revoke-key-${item.name}`}
          >
            Revoke
          </Button>
        );
      },
    },
  ];

  if (isLoading) {
    return <PageLoader message="Loading API keys..." />;
  }

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        <Button variant="primary" leftIcon={<Plus className="h-4 w-4" />} onClick={onCreateClick} data-testid="create-api-key-btn">
          Create API Key
        </Button>
      </div>
      {apiKeys.length === 0 ? (
        <EmptyState
          title="No API keys"
          description="Create an API key to authenticate with the n-kudo API."
          icon={<Key className="h-12 w-12 text-gray-400" />}
          action={{
            label: 'Create API Key',
            onClick: onCreateClick,
          }}
        />
      ) : (
        <Table
          columns={keyColumns}
          data={apiKeys}
          keyExtractor={(item) => item.id}
          hoverable
        />
      )}
    </div>
  );
}

// Token History Tab Component
interface TokenHistoryTabProps {
  tokenHistory: EnrollmentToken[];
  isLoading: boolean;
  copiedTokenId: string | null;
  onCopyToken: (tokenId: string, token: string) => void;
}

function TokenHistoryTab({ tokenHistory, isLoading, copiedTokenId, onCopyToken }: TokenHistoryTabProps) {
  const getStatus = (token: EnrollmentToken) => {
    if (token.consumed) return { label: 'Used', variant: 'success' as const };
    if (new Date(token.expires_at) < new Date()) return { label: 'Expired', variant: 'error' as const };
    return { label: 'Pending', variant: 'warning' as const };
  };

  const tokenColumns: TableColumn<EnrollmentToken>[] = [
    {
      key: 'site_name',
      title: 'Site',
      sortable: true,
      render: (_value: unknown, item: EnrollmentToken) => (
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-orange-100">
            <Key className="h-5 w-5 text-orange-600" />
          </div>
          <span className="font-medium text-gray-900">{item.site_name}</span>
        </div>
      ),
    },
    {
      key: 'consumed',
      title: 'Status',
      sortable: true,
      render: (_value: unknown, item: EnrollmentToken) => {
        const status = getStatus(item);
        return (
          <div className="flex flex-col gap-1">
            <Badge variant={status.variant} size="sm">
              {status.label}
            </Badge>
            {item.consumed_at && (
              <span className="text-xs text-gray-500">
                {new Date(item.consumed_at).toLocaleString()}
              </span>
            )}
          </div>
        );
      },
    },
    {
      key: 'created_at',
      title: 'Created',
      sortable: true,
      render: (_value: unknown, item: EnrollmentToken) => new Date(item.created_at).toLocaleString(),
    },
    {
      key: 'expires_at',
      title: 'Expires',
      sortable: true,
      render: (_value: unknown, item: EnrollmentToken) => new Date(item.expires_at).toLocaleString(),
    },
    {
      key: 'id',
      title: 'Token',
      align: 'right',
      render: (_value: unknown, item: EnrollmentToken) => {
        const status = getStatus(item);
        const isPending = status.label === 'Pending';
        
        return (
          <Button
            variant="ghost"
            size="sm"
            disabled={!isPending}
            leftIcon={
              copiedTokenId === item.id ? (
                <Check className="h-4 w-4 text-green-500" />
              ) : (
                <Copy className="h-4 w-4" />
              )
            }
            onClick={(e) => {
              e.stopPropagation();
              // Note: The actual token is only available at creation time.
              // For pending tokens, we show the token ID for reference.
              // In a real app, you might want to fetch the actual token if still valid.
              onCopyToken(item.id, item.id);
            }}
          >
            {copiedTokenId === item.id ? 'Copied' : isPending ? 'Copy' : 'Used'}
          </Button>
        );
      },
    },
  ];

  if (isLoading) {
    return <PageLoader message="Loading tokens..." />;
  }

  if (tokenHistory.length === 0) {
    return (
      <EmptyState
        title="No tokens issued"
        description="Enrollment tokens will appear here after they are issued."
        icon={<Key className="h-12 w-12 text-gray-400" />}
      />
    );
  }

  return <Table columns={tokenColumns} data={tokenHistory} keyExtractor={(item) => item.id} />;
}
