import { useState, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { Plus, Search, Building2, ExternalLink, Key } from 'lucide-react';
import {
  Card,
  Table,
  Button,
  Badge,
  Input,
  PageLoader,
  EmptyState,
  toast,
} from '@/components/common';
import type { TableColumn } from '@/components/common';
import { useTenants } from '@/api/hooks';
import { Tenant } from '@/api/types';
import { CreateTenantModal } from './CreateTenantModal';

interface TenantTableData {
  id: string;
  name: string;
  slug: string;
  primaryRegion: string;
  createdAt: string;
  sitesCount: number;
}

export function TenantsList() {
  const navigate = useNavigate();
  const [searchQuery, setSearchQuery] = useState('');
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);

  // Fetch tenants
  const { data: tenants, isLoading, error } = useTenants();

  // Transform tenants data for table
  const tableData: TenantTableData[] = useMemo(() => {
    if (!tenants) return [];
    return tenants.map((tenant: Tenant) => ({
      id: tenant.id,
      name: tenant.name,
      slug: tenant.slug,
      primaryRegion: tenant.primary_region || '-',
      createdAt: new Date(tenant.created_at).toLocaleDateString(),
      sitesCount: 0, // Will be fetched separately if needed
    }));
  }, [tenants]);

  // Filter tenants by search query
  const filteredData = useMemo(() => {
    if (!searchQuery.trim()) return tableData;
    const query = searchQuery.toLowerCase();
    return tableData.filter(
      (tenant) =>
        tenant.name.toLowerCase().includes(query) ||
        tenant.slug.toLowerCase().includes(query) ||
        tenant.primaryRegion.toLowerCase().includes(query)
    );
  }, [tableData, searchQuery]);

  // Handle tenant creation success
  const handleTenantCreated = () => {
    toast.success('Tenant created successfully');
  };

  // Handle errors
  if (error) {
    toast.error(`Failed to load tenants: ${error.message}`);
  }

  // Table columns configuration
  const columns: TableColumn<TenantTableData>[] = [
    {
      key: 'name',
      title: 'Name',
      sortable: true,
      render: (_value: unknown, item: TenantTableData) => (
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-100">
            <Building2 className="h-5 w-5 text-blue-600" />
          </div>
          <div>
            <p className="font-medium text-gray-900">{item.name}</p>
            <p className="text-sm text-gray-500">{item.slug}</p>
          </div>
        </div>
      ),
    },
    {
      key: 'slug',
      title: 'Slug',
      sortable: true,
      render: (_value: unknown, item: TenantTableData) => (
        <code className="rounded bg-gray-100 px-2 py-1 text-sm text-gray-700">
          {item.slug}
        </code>
      ),
    },
    {
      key: 'primaryRegion',
      title: 'Primary Region',
      sortable: true,
      render: (_value: unknown, item: TenantTableData) =>
        item.primaryRegion !== '-' ? (
          <Badge variant="info" size="sm">
            {item.primaryRegion}
          </Badge>
        ) : (
          <span className="text-gray-400">-</span>
        ),
    },
    {
      key: 'createdAt',
      title: 'Created',
      sortable: true,
    },
    {
      key: 'id',
      title: 'Actions',
      align: 'right',
      render: (_value: unknown, item: TenantTableData) => (
        <div className="flex items-center justify-end gap-2">
          <Button
            variant="ghost"
            size="sm"
            leftIcon={<Key className="h-4 w-4" />}
            onClick={(e) => {
              e.stopPropagation();
              navigate(`/admin/tenants/${item.id}?tab=apikeys`);
            }}
          >
            API Keys
          </Button>
          <Button
            variant="ghost"
            size="sm"
            leftIcon={<ExternalLink className="h-4 w-4" />}
            onClick={(e) => {
              e.stopPropagation();
              navigate(`/admin/tenants/${item.id}`);
            }}
          >
            View
          </Button>
        </div>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Tenants</h1>
          <p className="mt-1 text-sm text-gray-500">
            Manage organization tenants and their configurations
          </p>
        </div>
        <Button
          variant="primary"
          leftIcon={<Plus className="h-4 w-4" />}
          onClick={() => setIsCreateModalOpen(true)}
          data-testid="create-tenant-btn"
        >
          Create Tenant
        </Button>
      </div>

      {/* Filters */}
      <Card noPadding>
        <div className="border-b border-gray-200 px-6 py-4">
          <div className="flex flex-col gap-4 sm:flex-row sm:items-center">
            <div className="relative flex-1 max-w-md">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
              <Input
                placeholder="Search tenants..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                leftIcon={<Search className="h-4 w-4" />}
                className="pl-10"
              />
            </div>
            <div className="text-sm text-gray-500">
              {filteredData.length} tenant{filteredData.length !== 1 ? 's' : ''}
            </div>
          </div>
        </div>

        {/* Tenants Table */}
        {isLoading ? (
          <div className="p-8">
            <PageLoader message="Loading tenants..." />
          </div>
        ) : filteredData.length === 0 ? (
          <div className="p-8">
            <EmptyState
              title={searchQuery ? 'No tenants found' : 'No tenants yet'}
              description={
                searchQuery
                  ? 'Try adjusting your search terms'
                  : 'Create your first tenant to get started'
              }
              icon={<Building2 className="h-12 w-12 text-gray-400" />}
              action={
                !searchQuery
                  ? {
                      label: 'Create Tenant',
                      onClick: () => setIsCreateModalOpen(true),
                    }
                  : undefined
              }
            />
          </div>
        ) : (
          <Table
            columns={columns}
            data={filteredData}
            keyExtractor={(item) => item.id}
            rowTestId={(item) => `tenant-row-${item.id}`}
            hoverable
            onRowClick={(item) => navigate(`/admin/tenants/${item.id}`)}
          />
        )}
      </Card>

      {/* Create Tenant Modal */}
      <CreateTenantModal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        onSuccess={handleTenantCreated}
      />
    </div>
  );
}
