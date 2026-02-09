import { useState } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import {
  Plus,
  Search,
  MapPin,
  Server,
  ArrowLeft,
  MoreVertical,
  Globe,
  Activity,
} from 'lucide-react';
import {
  Card,
  Button,
  Input,
  Badge,
  Modal,
  EmptyState,
  SkeletonCard,
} from '@/components/common';
import { useSites, useCreateSite, queryKeys } from '@/api/hooks';
import { toast } from '@/stores/toastStore';
import { useQueryClient } from '@tanstack/react-query';

export function SitesList() {
  const { tenantId } = useParams<{ tenantId: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [searchQuery, setSearchQuery] = useState('');
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);

  // Form state for create site
  const [newSiteName, setNewSiteName] = useState('');
  const [newSiteKey, setNewSiteKey] = useState('');
  const [newSiteLocation, setNewSiteLocation] = useState('');

  const { data: sites, isLoading, error } = useSites(tenantId || '');

  const createSiteMutation = useCreateSite({
    onSuccess: () => {
      toast.success('Site created successfully');
      setIsCreateModalOpen(false);
      resetForm();
      // Invalidate sites query to refresh the list
      if (tenantId) {
        queryClient.invalidateQueries({ queryKey: queryKeys.sites(tenantId) });
      }
    },
    onError: (error) => {
      toast.error(error.message || 'Failed to create site');
    },
  });

  const resetForm = () => {
    setNewSiteName('');
    setNewSiteKey('');
    setNewSiteLocation('');
  };

  const filteredSites = sites?.filter((site) =>
    site.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    site.external_key.toLowerCase().includes(searchQuery.toLowerCase()) ||
    site.location_country_code.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const handleCreateSite = () => {
    if (!tenantId || !newSiteName.trim()) return;

    createSiteMutation.mutate({
      tenantId,
      data: {
        name: newSiteName.trim(),
        external_key: newSiteKey.trim() || undefined,
        location_country_code: newSiteLocation.trim() || undefined,
      },
    });
  };

  const getStatusBadge = (state: string) => {
    const isOnline = state === 'ONLINE' || state === 'connected';
    return (
      <Badge
        variant={isOnline ? 'success' : 'error'}
        withDot
        size="sm"
      >
        {isOnline ? 'ONLINE' : 'OFFLINE'}
      </Badge>
    );
  };

  const formatLastHeartbeat = (date: string) => {
    if (!date) return 'Never';
    const diff = Date.now() - new Date(date).getTime();
    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(diff / 3600000);
    const days = Math.floor(diff / 86400000);

    if (minutes < 1) return 'Just now';
    if (minutes < 60) return `${minutes} min ago`;
    if (hours < 24) return `${hours} hour${hours > 1 ? 's' : ''} ago`;
    return `${days} day${days > 1 ? 's' : ''} ago`;
  };

  if (error) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-2 text-sm text-gray-500">
          <Link to="/admin/tenants" className="hover:text-gray-700 flex items-center gap-1">
            <ArrowLeft className="w-4 h-4" />
            Back to Tenants
          </Link>
          <span>/</span>
          <span className="text-gray-900 font-medium">Sites</span>
        </div>
        <EmptyState
          icon="error"
          title="Failed to load sites"
          description={error.message}
          action={{
            label: 'Retry',
            onClick: () => {
              if (tenantId) {
                queryClient.invalidateQueries({ queryKey: queryKeys.sites(tenantId) });
              }
            },
          }}
        />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Breadcrumb */}
      <div className="flex items-center gap-2 text-sm text-gray-500">
        <Link to="/admin/tenants" className="hover:text-gray-700 flex items-center gap-1">
          <ArrowLeft className="w-4 h-4" />
          Back to Tenants
        </Link>
        <span>/</span>
        <span className="text-gray-900 font-medium">Sites</span>
      </div>

      {/* Page Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Sites</h1>
          <p className="mt-1 text-sm text-gray-500">
            Manage edge locations and infrastructure for this tenant
          </p>
        </div>
        <Button
          onClick={() => setIsCreateModalOpen(true)}
          leftIcon={<Plus className="w-4 h-4" />}
        >
          Create Site
        </Button>
      </div>

      {/* Search */}
      <div className="flex items-center gap-4">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <Input
            type="text"
            placeholder="Search sites by name, key, or location..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>
      </div>

      {/* Sites Grid */}
      {isLoading ? (
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          <SkeletonCard hasHeader hasFooter={false} contentLines={4} />
          <SkeletonCard hasHeader hasFooter={false} contentLines={4} />
          <SkeletonCard hasHeader hasFooter={false} contentLines={4} />
        </div>
      ) : filteredSites?.length === 0 ? (
        <EmptyState
          icon={searchQuery ? 'search' : 'folder'}
          title={searchQuery ? 'No matching sites' : 'No sites yet'}
          description={
            searchQuery
              ? 'Try adjusting your search criteria'
              : 'Create your first site to start deploying edge infrastructure'
          }
          action={
            !searchQuery
              ? {
                  label: 'Create Site',
                  onClick: () => setIsCreateModalOpen(true),
                }
              : undefined
          }
        />
      ) : (
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {filteredSites?.map((site) => (
            <Card
              key={site.id}
              className="cursor-pointer hover:shadow-md transition-shadow"
              onClick={() => navigate(`/tenant/${tenantId}/sites/${site.id}`)}
            >
              <div className="p-6">
                {/* Header */}
                <div className="flex items-start justify-between mb-4">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 bg-blue-100 rounded-lg flex items-center justify-center">
                      <Globe className="w-5 h-5 text-blue-600" />
                    </div>
                    <div>
                      <h3 className="font-semibold text-gray-900">{site.name}</h3>
                      <p className="text-sm text-gray-500">{site.external_key || site.id.slice(0, 8)}</p>
                    </div>
                  </div>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      // Handle menu actions
                    }}
                    className="p-1 text-gray-400 hover:text-gray-600"
                  >
                    <MoreVertical className="w-4 h-4" />
                  </button>
                </div>

                {/* Location */}
                <div className="flex items-center gap-2 text-sm text-gray-600 mb-4">
                  <MapPin className="w-4 h-4" />
                  <span>{site.location_country_code || 'Unknown location'}</span>
                </div>

                {/* Stats */}
                <div className="grid grid-cols-3 gap-4 mb-4">
                  <div className="text-center p-2 bg-gray-50 rounded-lg">
                    <div className="flex items-center justify-center gap-1 text-gray-500 mb-1">
                      <Server className="w-3 h-3" />
                      <span className="text-xs">Hosts</span>
                    </div>
                    <p className="font-semibold text-gray-900">-</p>
                  </div>
                  <div className="text-center p-2 bg-gray-50 rounded-lg">
                    <div className="flex items-center justify-center gap-1 text-gray-500 mb-1">
                      <Activity className="w-3 h-3" />
                      <span className="text-xs">VMs</span>
                    </div>
                    <p className="font-semibold text-gray-900">-</p>
                  </div>
                  <div className="text-center p-2 bg-gray-50 rounded-lg">
                    <div className="flex items-center justify-center gap-1 text-gray-500 mb-1">
                      <Activity className="w-3 h-3" />
                      <span className="text-xs">Heartbeat</span>
                    </div>
                    <p className="font-semibold text-gray-900">
                      {formatLastHeartbeat(site.last_heartbeat_at)}
                    </p>
                  </div>
                </div>

                {/* Status */}
                <div className="flex items-center justify-between pt-4 border-t border-gray-100">
                  {getStatusBadge(site.connectivity_state)}
                  <span className="text-xs text-gray-400">
                    Created {new Date(site.created_at).toLocaleDateString()}
                  </span>
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}

      {/* Create Site Modal */}
      <Modal
        isOpen={isCreateModalOpen}
        onClose={() => {
          setIsCreateModalOpen(false);
          resetForm();
        }}
        title="Create New Site"
        description="Add a new edge location to your tenant"
        footer={
          <div className="flex justify-end gap-3">
            <Button
              variant="secondary"
              onClick={() => {
                setIsCreateModalOpen(false);
                resetForm();
              }}
            >
              Cancel
            </Button>
            <Button
              onClick={handleCreateSite}
              loading={createSiteMutation.isPending}
              disabled={!newSiteName.trim()}
            >
              Create Site
            </Button>
          </div>
        }
      >
        <div className="space-y-4">
          <Input
            label="Site Name"
            placeholder="e.g., Main Data Center"
            value={newSiteName}
            onChange={(e) => setNewSiteName(e.target.value)}
            required
          />
          <Input
            label="External Key"
            placeholder="e.g., datacenter-nyc-01 (optional)"
            value={newSiteKey}
            onChange={(e) => setNewSiteKey(e.target.value)}
            helperText="A unique identifier for external integrations"
          />
          <Input
            label="Location Country Code"
            placeholder="e.g., US, DE, SG (optional)"
            value={newSiteLocation}
            onChange={(e) => setNewSiteLocation(e.target.value)}
            helperText="ISO country code for the site location"
          />
        </div>
      </Modal>
    </div>
  );
}
