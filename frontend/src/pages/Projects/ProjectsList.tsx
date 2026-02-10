import { useState, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { Plus, Search, FolderOpen, ExternalLink, Settings, Mail, AlertCircle } from 'lucide-react';
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
import { useProjects } from '@/api/hooks';
import { Project } from '@/api/types';
import { CreateProjectModal } from './CreateProjectModal';
import { getCurrentUser, resendVerification } from '@/api/auth';

interface ProjectTableData {
  id: string;
  name: string;
  slug: string;
  primaryRegion: string;
  role: string;
  createdAt: string;
}

export function ProjectsList() {
  const navigate = useNavigate();
  const [searchQuery, setSearchQuery] = useState('');
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [isResending, setIsResending] = useState(false);

  // Fetch projects
  const { data: projects, isLoading, error } = useProjects();
  
  // Get current user to check email verification
  const currentUser = getCurrentUser();
  const emailNotVerified = currentUser && !currentUser.email_verified;

  // Transform projects data for table
  const tableData: ProjectTableData[] = useMemo(() => {
    if (!projects) return [];
    return projects.map((project: Project) => ({
      id: project.id,
      name: project.name,
      slug: project.slug,
      primaryRegion: project.primary_region || '-',
      role: project.role,
      createdAt: new Date(project.created_at).toLocaleDateString(),
    }));
  }, [projects]);

  // Filter projects by search query
  const filteredData = useMemo(() => {
    if (!searchQuery.trim()) return tableData;
    const query = searchQuery.toLowerCase();
    return tableData.filter(
      (project) =>
        project.name.toLowerCase().includes(query) ||
        project.slug.toLowerCase().includes(query) ||
        project.primaryRegion.toLowerCase().includes(query)
    );
  }, [tableData, searchQuery]);

  // Handle project creation success
  const handleProjectCreated = () => {
    toast.success('Project created successfully');
  };
  
  // Handle resend verification email
  const handleResendVerification = async () => {
    setIsResending(true);
    try {
      const result = await resendVerification();
      toast.success(result.message);
    } catch (error: any) {
      const message = error.response?.data?.error || 'Failed to send verification email';
      toast.error(message);
    } finally {
      setIsResending(false);
    }
  };

  // Handle errors
  if (error) {
    toast.error(`Failed to load projects: ${error.message}`);
  }

  // Table columns configuration
  const columns: TableColumn<ProjectTableData>[] = [
    {
      key: 'name',
      title: 'Name',
      sortable: true,
      render: (_value: unknown, item: ProjectTableData) => (
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-100">
            <FolderOpen className="h-5 w-5 text-blue-600" />
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
      render: (_value: unknown, item: ProjectTableData) => (
        <code className="rounded bg-gray-100 px-2 py-1 text-sm text-gray-700">
          {item.slug}
        </code>
      ),
    },
    {
      key: 'primaryRegion',
      title: 'Region',
      sortable: true,
      render: (_value: unknown, item: ProjectTableData) =>
        item.primaryRegion !== '-' ? (
          <Badge variant="info" size="sm">
            {item.primaryRegion}
          </Badge>
        ) : (
          <span className="text-gray-400">-</span>
        ),
    },
    {
      key: 'role',
      title: 'Your Role',
      sortable: true,
      render: (_value: unknown, item: ProjectTableData) => (
        <Badge variant={item.role === 'OWNER' ? 'success' : 'default'} size="sm">
          {item.role}
        </Badge>
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
      render: (_value: unknown, item: ProjectTableData) => (
        <div className="flex items-center justify-end gap-2">
          <Button
            variant="ghost"
            size="sm"
            leftIcon={<Settings className="h-4 w-4" />}
            onClick={(e) => {
              e.stopPropagation();
              navigate(`/projects/${item.id}/settings`);
            }}
          >
            Settings
          </Button>
          <Button
            variant="ghost"
            size="sm"
            leftIcon={<ExternalLink className="h-4 w-4" />}
            onClick={(e) => {
              e.stopPropagation();
              navigate(`/projects/${item.id}`);
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
          <h1 className="text-2xl font-bold text-gray-900">Projects</h1>
          <p className="mt-1 text-sm text-gray-500">
            Manage your projects and their configurations
          </p>
        </div>
        <Button
          variant="primary"
          leftIcon={<Plus className="h-4 w-4" />}
          onClick={() => setIsCreateModalOpen(true)}
          data-testid="create-project-btn"
        >
          Create Project
        </Button>
      </div>

      {/* Email Verification Banner */}
      {emailNotVerified && (
        <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
          <div className="flex items-start gap-3">
            <AlertCircle className="h-5 w-5 text-yellow-600 mt-0.5" />
            <div className="flex-1">
              <h3 className="text-sm font-medium text-yellow-800">
                Please verify your email address
              </h3>
              <p className="text-sm text-yellow-700 mt-1">
                Check your inbox for a verification email, or click the button below to resend it.
              </p>
            </div>
            <Button
              variant="secondary"
              size="sm"
              leftIcon={<Mail className="h-4 w-4" />}
              onClick={handleResendVerification}
              loading={isResending}
            >
              Resend Email
            </Button>
          </div>
        </div>
      )}

      {/* Filters */}
      <Card noPadding>
        <div className="border-b border-gray-200 px-6 py-4">
          <div className="flex flex-col gap-4 sm:flex-row sm:items-center">
            <div className="relative flex-1 max-w-md">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
              <Input
                placeholder="Search projects..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                leftIcon={<Search className="h-4 w-4" />}
                className="pl-10"
              />
            </div>
            <div className="text-sm text-gray-500">
              {filteredData.length} project{filteredData.length !== 1 ? 's' : ''}
            </div>
          </div>
        </div>

        {/* Projects Table */}
        {isLoading ? (
          <div className="p-8">
            <PageLoader message="Loading projects..." />
          </div>
        ) : filteredData.length === 0 ? (
          <div className="p-8">
            <EmptyState
              title={searchQuery ? 'No projects found' : 'No projects yet'}
              description={
                searchQuery
                  ? 'Try adjusting your search terms'
                  : 'Create your first project to get started'
              }
              icon={<FolderOpen className="h-12 w-12 text-gray-400" />}
              action={
                !searchQuery
                  ? {
                      label: 'Create Project',
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
            rowTestId={(item) => `project-row-${item.id}`}
            hoverable
            onRowClick={(item) => navigate(`/projects/${item.id}`)}
          />
        )}
      </Card>

      {/* Create Project Modal */}
      <CreateProjectModal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        onSuccess={handleProjectCreated}
      />
    </div>
  );
}
