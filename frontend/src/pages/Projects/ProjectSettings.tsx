import { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  FolderOpen,
  ArrowLeft,
  Globe,
  Calendar,
  Key,
  Plus,
  Copy,
  Check,
  Trash2,
  Shield,
  Clock,
  AlertTriangle,
  AlertCircle,
} from 'lucide-react';
import {
  Card,
  Table,
  Button,
  Badge,
  PageLoader,
  EmptyState,
  Modal,
} from '@/components/common';
import type { TableColumn } from '@/components/common';
import { useProject, useAPIKeys, useRevokeAPIKey, useEnrollmentTokens, useSites } from '@/api/hooks';
import { APIKey, EnrollmentToken } from '@/api/types';
import { toast } from '@/stores/toastStore';
import { IssueTokenModal } from '../Admin/IssueTokenModal';
import { CreateAPIKeyModal } from '../Admin/CreateAPIKeyModal';

export function ProjectSettings() {
  const { projectId } = useParams<{ projectId: string }>();
  const navigate = useNavigate();
  const [isTokenModalOpen, setIsTokenModalOpen] = useState(false);
  const [isAPIKeyModalOpen, setIsAPIKeyModalOpen] = useState(false);
  const [copiedTokenId, setCopiedTokenId] = useState<string | null>(null);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

  // Fetch project data
  const { data: project, isLoading: isLoadingProject, error: projectError } = useProject(projectId || '');

  // Fetch sites for this project (needed for token modal)
  const { data: sites } = useSites(projectId || '');

  // Fetch API keys for this project
  const { data: apiKeys, isLoading: isLoadingAPIKeys } = useAPIKeys(projectId || '');

  // Fetch enrollment tokens for this project
  const { data: enrollmentTokens, isLoading: isLoadingTokens } = useEnrollmentTokens(projectId || '');

  // Handle errors
  if (projectError) {
    toast.error(`Failed to load project: ${projectError.message}`);
  }

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

  if (isLoadingProject) {
    return (
      <div className="flex min-h-[400px] items-center justify-center">
        <PageLoader message="Loading project settings..." />
      </div>
    );
  }

  if (!project) {
    return (
      <div className="flex min-h-[400px] flex-col items-center justify-center">
        <EmptyState
          title="Project not found"
          description="The project you are looking for does not exist or you don't have access to it."
          icon={<FolderOpen className="h-12 w-12 text-gray-400" />}
          action={{
            label: 'Back to Projects',
            onClick: () => navigate('/projects'),
          }}
        />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Back button */}
      <Button
        variant="ghost"
        size="sm"
        leftIcon={<ArrowLeft className="h-4 w-4" />}
        onClick={() => navigate(`/projects/${projectId}`)}
        className="-ml-2"
      >
        Back to Project
      </Button>

      {/* Page Header */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Project Settings</h1>
        <p className="mt-1 text-sm text-gray-500">
          Manage {project.name} settings and configuration
        </p>
      </div>

      {/* Project Info Card */}
      <Card>
        <div className="flex items-start gap-4">
          <div className="flex h-16 w-16 items-center justify-center rounded-xl bg-blue-100">
            <FolderOpen className="h-8 w-8 text-blue-600" />
          </div>
          <div className="flex-1">
            <h2 className="text-xl font-bold text-gray-900">{project.name}</h2>
            <div className="mt-1 flex flex-wrap items-center gap-3 text-sm text-gray-500">
              <code className="rounded bg-gray-100 px-2 py-1">{project.slug}</code>
              <span>•</span>
              <span className="flex items-center gap-1">
                <Globe className="h-4 w-4" />
                {project.primary_region || 'No region set'}
              </span>
              <span>•</span>
              <span className="flex items-center gap-1">
                <Calendar className="h-4 w-4" />
                Created {new Date(project.created_at).toLocaleDateString()}
              </span>
            </div>
            <div className="mt-4 flex items-center gap-2">
              <Badge variant="info" size="sm">
                ID: {project.id.slice(0, 8)}...
              </Badge>
              <Badge variant="success" size="sm">
                {project.role}
              </Badge>
            </div>
          </div>
        </div>
      </Card>

      {/* API Keys Section */}
      <Card>
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-purple-100">
              <Shield className="h-5 w-5 text-purple-600" />
            </div>
            <div>
              <h3 className="font-semibold text-gray-900">API Keys</h3>
              <p className="text-sm text-gray-500">
                Manage API keys for programmatic access to the n-kudo API
              </p>
            </div>
          </div>
          <Button
            variant="primary"
            size="sm"
            leftIcon={<Plus className="h-4 w-4" />}
            onClick={() => setIsAPIKeyModalOpen(true)}
          >
            Create API Key
          </Button>
        </div>

        <APIKeysTable
          apiKeys={apiKeys || []}
          projectId={projectId || ''}
          isLoading={isLoadingAPIKeys}
        />
      </Card>

      {/* Enrollment Tokens Section */}
      <Card>
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-orange-100">
              <Key className="h-5 w-5 text-orange-600" />
            </div>
            <div>
              <h3 className="font-semibold text-gray-900">Enrollment Tokens</h3>
              <p className="text-sm text-gray-500">
                Issue one-time tokens to enroll edge agents at your sites
              </p>
            </div>
          </div>
          <Button
            variant="primary"
            size="sm"
            leftIcon={<Plus className="h-4 w-4" />}
            onClick={() => setIsTokenModalOpen(true)}
            disabled={!sites || sites.length === 0}
          >
            Issue Token
          </Button>
        </div>

        {!sites || sites.length === 0 ? (
          <div className="rounded-lg border border-yellow-200 bg-yellow-50 p-4">
            <div className="flex gap-3">
              <AlertCircle className="h-5 w-5 text-yellow-600" />
              <div>
                <p className="text-sm font-medium text-yellow-800">
                  No sites available
                </p>
                <p className="mt-1 text-sm text-yellow-700">
                  Create a site first before issuing enrollment tokens.
                </p>
                <Button
                  variant="secondary"
                  size="sm"
                  className="mt-2"
                  onClick={() => navigate(`/projects/${projectId}/sites`)}
                >
                  Go to Sites
                </Button>
              </div>
            </div>
          </div>
        ) : (
          <TokensTable
            tokens={enrollmentTokens || []}
            isLoading={isLoadingTokens}
            copiedTokenId={copiedTokenId}
            onCopyToken={handleCopyToken}
          />
        )}
      </Card>

      {/* Danger Zone */}
      <Card className="border-red-200">
        <div className="flex items-start gap-4">
          <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-red-100">
            <AlertTriangle className="h-6 w-6 text-red-600" />
          </div>
          <div className="flex-1">
            <h3 className="font-semibold text-gray-900">Danger Zone</h3>
            <p className="mt-1 text-sm text-gray-500">
              Irreversible and destructive actions. Proceed with caution.
            </p>
            <div className="mt-4">
              <Button
                variant="danger"
                leftIcon={<Trash2 className="h-4 w-4" />}
                onClick={() => setShowDeleteConfirm(true)}
              >
                Delete Project
              </Button>
            </div>
          </div>
        </div>
      </Card>

      {/* Issue Token Modal */}
      <IssueTokenModal
        isOpen={isTokenModalOpen}
        onClose={() => setIsTokenModalOpen(false)}
        tenantId={projectId || ''}
        sites={sites || []}
      />

      {/* Create API Key Modal */}
      <CreateAPIKeyModal
        isOpen={isAPIKeyModalOpen}
        onClose={() => setIsAPIKeyModalOpen(false)}
        tenantId={projectId || ''}
      />

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={showDeleteConfirm}
        onClose={() => setShowDeleteConfirm(false)}
        title="Delete Project"
        description="This action cannot be undone. This will permanently delete the project and all associated data."
        size="md"
        footer={
          <div className="flex justify-end gap-3">
            <Button variant="secondary" onClick={() => setShowDeleteConfirm(false)}>
              Cancel
            </Button>
            <Button
              variant="danger"
              leftIcon={<Trash2 className="h-4 w-4" />}
              onClick={() => {
                toast.error('Project deletion is not yet implemented');
                setShowDeleteConfirm(false);
              }}
            >
              Delete Project
            </Button>
          </div>
        }
      >
        <div className="rounded-lg border border-red-200 bg-red-50 p-4">
          <div className="flex gap-3">
            <AlertTriangle className="h-5 w-5 text-red-600" />
            <div>
              <p className="text-sm font-medium text-red-800">
                Warning
              </p>
              <p className="mt-1 text-sm text-red-700">
                Deleting this project will remove all sites, hosts, VMs, and associated data.
                This action is irreversible.
              </p>
            </div>
          </div>
        </div>
      </Modal>
    </div>
  );
}

// API Keys Table Component
interface APIKeysTableProps {
  apiKeys: APIKey[];
  projectId: string;
  isLoading: boolean;
}

function APIKeysTable({ apiKeys, projectId, isLoading }: APIKeysTableProps) {
  const revokeAPIKey = useRevokeAPIKey();
  const [confirmingKeyId, setConfirmingKeyId] = useState<string | null>(null);

  const handleRevoke = async (keyId: string) => {
    if (confirmingKeyId !== keyId) {
      setConfirmingKeyId(keyId);
      return;
    }

    try {
      await revokeAPIKey.mutateAsync({ tenantId: projectId, keyId });
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

  if (apiKeys.length === 0) {
    return (
      <EmptyState
        title="No API keys"
        description="Create an API key to authenticate with the n-kudo API."
        icon={<Key className="h-12 w-12 text-gray-400" />}
      />
    );
  }

  return (
    <Table
      columns={keyColumns}
      data={apiKeys}
      keyExtractor={(item) => item.id}
      hoverable
    />
  );
}

// Tokens Table Component
interface TokensTableProps {
  tokens: EnrollmentToken[];
  isLoading: boolean;
  copiedTokenId: string | null;
  onCopyToken: (tokenId: string, token: string) => void;
}

function TokensTable({ tokens, isLoading, copiedTokenId, onCopyToken }: TokensTableProps) {
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
      render: (_value: unknown, item: EnrollmentToken) => (
        <span className="flex items-center gap-1 text-sm text-gray-500">
          <Clock className="h-3.5 w-3.5" />
          {new Date(item.created_at).toLocaleString()}
        </span>
      ),
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

  if (tokens.length === 0) {
    return (
      <EmptyState
        title="No tokens issued"
        description="Enrollment tokens will appear here after they are issued."
        icon={<Key className="h-12 w-12 text-gray-400" />}
      />
    );
  }

  return (
    <Table
      columns={tokenColumns}
      data={tokens}
      keyExtractor={(item) => item.id}
      hoverable
    />
  );
}
