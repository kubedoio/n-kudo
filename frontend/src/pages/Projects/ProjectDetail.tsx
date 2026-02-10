import { useParams, useNavigate } from 'react-router-dom';
import {
  FolderOpen,
  Server,
  Cpu,
  Settings,
  ExternalLink,
  Globe,
  Calendar,
  ArrowRight,
  Plus,
  MapPin,
} from 'lucide-react';
import {
  Card,
  Button,
  Badge,
  PageLoader,
  EmptyState,
} from '@/components/common';
import { useProject, useSites } from '@/api/hooks';
import { toast } from '@/stores/toastStore';

export function ProjectDetail() {
  const { projectId } = useParams<{ projectId: string }>();
  const navigate = useNavigate();

  // Fetch project data
  const { data: project, isLoading: isLoadingProject, error: projectError } = useProject(projectId || '');

  // Fetch sites for this project (project ID = tenant ID)
  const { data: sites, isLoading: isLoadingSites } = useSites(projectId || '');

  // Handle errors
  if (projectError) {
    toast.error(`Failed to load project: ${projectError.message}`);
  }

  if (isLoadingProject) {
    return (
      <div className="flex min-h-[400px] items-center justify-center">
        <PageLoader message="Loading project..." />
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

  const isLoading = isLoadingSites;
  const sitesCount = sites?.length || 0;
  const hostsCount = 0; // Would need to fetch from all sites
  const vmsCount = 0; // Would need to fetch from all sites

  return (
    <div className="space-y-6">
      {/* Project Info Card */}
      <Card>
        <div className="flex flex-col gap-6 lg:flex-row lg:items-start lg:justify-between">
          <div className="flex items-start gap-4">
            <div className="flex h-16 w-16 items-center justify-center rounded-xl bg-blue-100">
              <FolderOpen className="h-8 w-8 text-blue-600" />
            </div>
            <div>
              <h1 className="text-2xl font-bold text-gray-900">{project.name}</h1>
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
            </div>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button
              variant="secondary"
              leftIcon={<Settings className="h-4 w-4" />}
              onClick={() => navigate(`/projects/${projectId}/settings`)}
            >
              Settings
            </Button>
            <Button
              variant="primary"
              leftIcon={<Plus className="h-4 w-4" />}
              onClick={() => navigate(`/projects/${projectId}/sites`)}
            >
              Manage Sites
            </Button>
          </div>
        </div>

        {/* Stats */}
        <div className="mt-6 grid gap-4 border-t border-gray-200 pt-6 sm:grid-cols-3">
          <div className="flex items-center gap-4">
            <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-green-100">
              <Server className="h-6 w-6 text-green-600" />
            </div>
            <div>
              <p className="text-sm text-gray-500">Sites</p>
              <p className="text-2xl font-semibold text-gray-900">
                {isLoading ? '-' : sitesCount}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-4">
            <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-purple-100">
              <Cpu className="h-6 w-6 text-purple-600" />
            </div>
            <div>
              <p className="text-sm text-gray-500">Hosts</p>
              <p className="text-2xl font-semibold text-gray-900">
                {isLoading ? '-' : hostsCount}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-4">
            <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-orange-100">
              <Globe className="h-6 w-6 text-orange-600" />
            </div>
            <div>
              <p className="text-sm text-gray-500">VMs</p>
              <p className="text-2xl font-semibold text-gray-900">
                {isLoading ? '-' : vmsCount}
              </p>
            </div>
          </div>
        </div>
      </Card>

      {/* Quick Actions */}
      <div className="grid gap-6 md:grid-cols-2">
        {/* Sites Card */}
        <Card>
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-green-100">
                <Server className="h-5 w-5 text-green-600" />
              </div>
              <div>
                <h3 className="font-semibold text-gray-900">Sites</h3>
                <p className="text-sm text-gray-500">
                  {sitesCount} {sitesCount === 1 ? 'site' : 'sites'} in this project
                </p>
              </div>
            </div>
            <Button
              variant="ghost"
              size="sm"
              rightIcon={<ArrowRight className="h-4 w-4" />}
              onClick={() => navigate(`/projects/${projectId}/sites`)}
            >
              View All
            </Button>
          </div>
          
          {isLoadingSites ? (
            <div className="animate-pulse space-y-3">
              <div className="h-12 bg-gray-100 rounded"></div>
              <div className="h-12 bg-gray-100 rounded"></div>
            </div>
          ) : sites && sites.length > 0 ? (
            <div className="space-y-2">
              {sites.slice(0, 3).map((site) => (
                <div
                  key={site.id}
                  className="flex items-center justify-between p-3 rounded-lg hover:bg-gray-50 cursor-pointer transition-colors"
                  onClick={() => navigate(`/projects/${projectId}/sites/${site.id}`)}
                >
                  <div className="flex items-center gap-3">
                    <MapPin className="h-4 w-4 text-gray-400" />
                    <div>
                      <p className="font-medium text-gray-900">{site.name}</p>
                      <p className="text-xs text-gray-500">
                        {site.external_key || site.id.slice(0, 8)}
                      </p>
                    </div>
                  </div>
                  <Badge
                    variant={site.connectivity_state === 'ONLINE' ? 'success' : 'default'}
                    size="sm"
                  >
                    {site.connectivity_state}
                  </Badge>
                </div>
              ))}
              {sites.length > 3 && (
                <p className="text-sm text-gray-500 text-center pt-2">
                  +{sites.length - 3} more sites
                </p>
              )}
            </div>
          ) : (
            <EmptyState
              title="No sites yet"
              description="Sites are created when edge agents enroll using an enrollment token."
              icon={<Server className="h-8 w-8 text-gray-400" />}
              action={{
                label: 'Go to Sites',
                onClick: () => navigate(`/projects/${projectId}/sites`),
              }}
            />
          )}
        </Card>

        {/* Settings Card */}
        <Card>
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-purple-100">
                <Settings className="h-5 w-5 text-purple-600" />
              </div>
              <div>
                <h3 className="font-semibold text-gray-900">Project Settings</h3>
                <p className="text-sm text-gray-500">Manage API keys and enrollment tokens</p>
              </div>
            </div>
            <Button
              variant="ghost"
              size="sm"
              rightIcon={<ArrowRight className="h-4 w-4" />}
              onClick={() => navigate(`/projects/${projectId}/settings`)}
            >
              Open
            </Button>
          </div>
          
          <div className="space-y-3">
            <div className="flex items-center gap-3 p-3 rounded-lg bg-gray-50">
              <ExternalLink className="h-4 w-4 text-gray-400" />
              <div>
                <p className="font-medium text-gray-900">API Keys</p>
                <p className="text-xs text-gray-500">
                  Manage API keys for programmatic access
                </p>
              </div>
            </div>
            <div className="flex items-center gap-3 p-3 rounded-lg bg-gray-50">
              <Server className="h-4 w-4 text-gray-400" />
              <div>
                <p className="font-medium text-gray-900">Enrollment Tokens</p>
                <p className="text-xs text-gray-500">
                  Issue tokens to enroll edge agents
                </p>
              </div>
            </div>
            <div className="flex items-center gap-3 p-3 rounded-lg bg-gray-50">
              <Globe className="h-4 w-4 text-gray-400" />
              <div>
                <p className="font-medium text-gray-900">Danger Zone</p>
                <p className="text-xs text-gray-500">
                  Delete project (irreversible)
                </p>
              </div>
            </div>
          </div>
        </Card>
      </div>

      {/* Documentation Link */}
      <div className="rounded-lg border border-blue-200 bg-blue-50 p-4">
        <div className="flex items-start gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-100">
            <ExternalLink className="h-5 w-5 text-blue-600" />
          </div>
          <div className="flex-1">
            <h3 className="font-medium text-blue-900">Getting Started</h3>
            <p className="mt-1 text-sm text-blue-700">
              Learn how to enroll edge agents, create VMs, and manage your infrastructure.
            </p>
            <div className="mt-3">
              <Button
                variant="secondary"
                size="sm"
                onClick={() => navigate(`/projects/${projectId}/sites`)}
              >
                View Sites
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
