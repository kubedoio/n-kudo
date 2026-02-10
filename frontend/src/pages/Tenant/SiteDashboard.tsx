import { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import {
  ArrowLeft,
  Server,
  Cpu,
  MemoryStick,
  Play,
  Square,
  Plus,
  MapPin,
  Globe,
  ChevronRight,
  FolderOpen,
} from 'lucide-react';
import {
  Card,
  Button,
  Badge,
  Table,
  Modal,
  EmptyState,
  SkeletonCard,
  DataTableSkeleton,
} from '@/components/common';
import type { TableColumn } from '@/components/common';
import { VMCreateModal } from './VMCreateModal';
import { VMActionsMenu } from './VMActionsMenu';
import { ExecutionLogViewer } from './ExecutionLogViewer';
import {
  useSite,
  useVMs,
  useHosts,
  useApplyPlanFromActions,
  useExecutions,
  queryKeys,
} from '@/api/hooks';
import { toast } from '@/stores/toastStore';
import { useQueryClient } from '@tanstack/react-query';
import type { MicroVM, Host, Execution } from '@/api/types';

type Tab = 'vms' | 'hosts' | 'plans';

const statusMap: Record<string, { variant: 'success' | 'error' | 'warning' | 'info' | 'default'; label: string }> = {
  PENDING: { variant: 'warning', label: 'Pending' },
  IN_PROGRESS: { variant: 'info', label: 'Running' },
  SUCCEEDED: { variant: 'success', label: 'Succeeded' },
  FAILED: { variant: 'error', label: 'Failed' },
};

export function SiteDashboard() {
  const { projectId, siteId } = useParams<{ projectId: string; siteId: string }>();
  const queryClient = useQueryClient();
  const [activeTab, setActiveTab] = useState<Tab>('vms');
  const [isCreateVMModalOpen, setIsCreateVMModalOpen] = useState(false);
  const [selectedExecutionId, setSelectedExecutionId] = useState<string | null>(null);
  const [selectedVM, setSelectedVM] = useState<MicroVM | null>(null);
  const [isVMDetailsOpen, setIsVMDetailsOpen] = useState(false);

  const { data: site, isLoading: isSiteLoading } = useSite(projectId || '', siteId || '');
  const { data: vms, isLoading: isVMsLoading } = useVMs(siteId || '');
  const { data: hosts, isLoading: isHostsLoading } = useHosts(siteId || '');
  
  // Fetch executions for the plans tab
  const { data: executions, isLoading: isExecutionsLoading, refetch: refetchExecutions } = useExecutions(
    siteId || '',
    { limit: 50 }
  );

  // Poll for pending/in-progress executions (5-second interval)
  useEffect(() => {
    if (!siteId) return;

    const interval = setInterval(() => {
      // Only poll if there are pending or in-progress executions
      const hasActiveExecutions = executions?.some(
        (e) => e.state === 'PENDING' || e.state === 'IN_PROGRESS'
      );
      if (hasActiveExecutions) {
        refetchExecutions();
      }
    }, 5000);

    return () => clearInterval(interval);
  }, [siteId, executions, refetchExecutions]);

  const applyPlanMutation = useApplyPlanFromActions({
    onSuccess: (data) => {
      toast.success(`Plan applied successfully (version ${data.plan_version})`);
      // Invalidate VMs query to refresh the list
      if (siteId) {
        queryClient.invalidateQueries({ queryKey: queryKeys.vms(siteId) });
      }
    },
    onError: (error) => {
      toast.error(error.message || 'Failed to apply plan');
    },
  });

  const getVMStatusBadge = (state: string) => {
    const statusMap: Record<string, { variant: 'success' | 'error' | 'warning' | 'info' | 'default'; label: string }> = {
      RUNNING: { variant: 'success', label: 'RUNNING' },
      STOPPED: { variant: 'error', label: 'STOPPED' },
      CREATING: { variant: 'warning', label: 'CREATING' },
      STARTING: { variant: 'info', label: 'STARTING' },
      STOPPING: { variant: 'info', label: 'STOPPING' },
      DELETING: { variant: 'warning', label: 'DELETING' },
    };
    const status = statusMap[state] || { variant: 'default', label: state };
    return <Badge variant={status.variant}>{status.label}</Badge>;
  };

  const handleVMClick = (vm: MicroVM) => {
    setSelectedVM(vm);
    setIsVMDetailsOpen(true);
  };

  const handleStartVM = (vm: MicroVM) => {
    if (!siteId) return;
    applyPlanMutation.mutate({
      siteId,
      idempotencyKey: `start-${vm.id}-${Date.now()}`,
      actions: [{ operation: 'START', vm_id: vm.id }],
    });
  };

  const handleStopVM = (vm: MicroVM) => {
    if (!siteId) return;
    applyPlanMutation.mutate({
      siteId,
      idempotencyKey: `stop-${vm.id}-${Date.now()}`,
      actions: [{ operation: 'STOP', vm_id: vm.id }],
    });
  };

  const handleDeleteVM = (vm: MicroVM) => {
    if (!siteId) return;
    applyPlanMutation.mutate({
      siteId,
      idempotencyKey: `delete-${vm.id}-${Date.now()}`,
      actions: [{ operation: 'DELETE', vm_id: vm.id }],
    });
  };

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const formatDate = (date: string) => {
    return new Date(date).toLocaleString();
  };

  const vmColumns: TableColumn<MicroVM>[] = [
    {
      key: 'name',
      title: 'Name',
      sortable: true,
      render: (_value: unknown, vm: MicroVM) => (
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 bg-gray-100 rounded-lg flex items-center justify-center">
            <Server className="w-4 h-4 text-gray-600" />
          </div>
          <div>
            <div className="font-medium text-gray-900">{vm.name}</div>
            <div className="text-xs text-gray-500">{String(vm.id).slice(0, 8)}</div>
          </div>
        </div>
      ),
    },
    {
      key: 'state',
      title: 'State',
      sortable: true,
      render: (_value: unknown, vm: MicroVM) => getVMStatusBadge(vm.state),
    },
    {
      key: 'vcpu_count',
      title: 'vCPUs',
      sortable: true,
      render: (_value: unknown, vm: MicroVM) => (
        <div className="flex items-center gap-2">
          <Cpu className="w-4 h-4 text-gray-400" />
          <span>{vm.vcpu_count}</span>
        </div>
      ),
    },
    {
      key: 'memory_mib',
      title: 'Memory',
      sortable: true,
      render: (_value: unknown, vm: MicroVM) => (
        <div className="flex items-center gap-2">
          <MemoryStick className="w-4 h-4 text-gray-400" />
          <span>{vm.memory_mib} MiB</span>
        </div>
      ),
    },
    {
      key: 'host_id',
      title: 'Host',
      sortable: true,
      render: (_value: unknown, vm: MicroVM) => (
        <span className="text-sm text-gray-500">{String(vm.host_id || '-').slice(0, 8)}</span>
      ),
    },
    {
      key: 'id',
      title: 'Actions',
      align: 'right',
      render: (_value: unknown, vm: MicroVM) => (
        <div className="flex items-center justify-end gap-1">
          {vm.state === 'RUNNING' ? (
            <Button
              variant="ghost"
              size="sm"
              onClick={(e) => {
                e.stopPropagation();
                handleStopVM(vm);
              }}
              disabled={applyPlanMutation.isPending}
              leftIcon={<Square className="w-4 h-4" />}
            >
              Stop
            </Button>
          ) : vm.state === 'STOPPED' ? (
            <Button
              variant="ghost"
              size="sm"
              onClick={(e) => {
                e.stopPropagation();
                handleStartVM(vm);
              }}
              disabled={applyPlanMutation.isPending}
              leftIcon={<Play className="w-4 h-4" />}
            >
              Start
            </Button>
          ) : null}
          <VMActionsMenu
            vm={vm}
            onStart={() => handleStartVM(vm)}
            onStop={() => handleStopVM(vm)}
            onDelete={() => handleDeleteVM(vm)}
            isLoading={applyPlanMutation.isPending}
          />
        </div>
      ),
    },
  ];

  const hostColumns: TableColumn<Host>[] = [
    {
      key: 'hostname',
      title: 'Hostname',
      sortable: true,
      render: (_value: unknown, host: Host) => (
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 bg-green-100 rounded-lg flex items-center justify-center">
            <Server className="w-4 h-4 text-green-600" />
          </div>
          <div>
            <div className="font-medium text-gray-900">{host.hostname}</div>
            <div className="text-xs text-gray-500">{String(host.id).slice(0, 8)}</div>
          </div>
        </div>
      ),
    },
    {
      key: 'agent_state',
      title: 'Status',
      sortable: true,
      render: (_value: unknown, host: Host) => (
        <Badge variant={host.agent_state === 'connected' ? 'success' : 'warning'}>
          {String(host.agent_state || 'UNKNOWN').toUpperCase()}
        </Badge>
      ),
    },
    {
      key: 'cpu_cores_total',
      title: 'CPU Cores',
      sortable: true,
      render: (_value: unknown, host: Host) => (
        <div className="flex items-center gap-2">
          <Cpu className="w-4 h-4 text-gray-400" />
          <span>{host.cpu_cores_total} cores</span>
        </div>
      ),
    },
    {
      key: 'memory_bytes_total',
      title: 'Memory',
      sortable: true,
      render: (_value: unknown, host: Host) => formatBytes(host.memory_bytes_total),
    },
    {
      key: 'kvm_available',
      title: 'Capabilities',
      render: (_value: unknown, host: Host) => (
        <div className="flex items-center gap-2">
          {host.kvm_available && (
            <Badge variant="info" size="sm">KVM</Badge>
          )}
          {host.cloud_hypervisor_available && (
            <Badge variant="success" size="sm">Cloud Hypervisor</Badge>
          )}
        </div>
      ),
    },
    {
      key: 'last_facts_at',
      title: 'Last Heartbeat',
      sortable: true,
      render: (_value: unknown, host: Host) => {
        if (!host.last_facts_at) return '-';
        const diff = Date.now() - new Date(host.last_facts_at).getTime();
        const minutes = Math.floor(diff / 60000);
        if (minutes < 1) return 'Just now';
        if (minutes < 60) return `${minutes} min ago`;
        const hours = Math.floor(minutes / 60);
        if (hours < 24) return `${hours} hour${hours > 1 ? 's' : ''} ago`;
        return new Date(host.last_facts_at).toLocaleDateString();
      },
    },
  ];

  const planColumns: TableColumn<Execution>[] = [
    {
      key: 'id',
      title: 'Execution ID',
      sortable: true,
      render: (_value: unknown, execution: Execution) => <span className="font-mono text-sm">{execution.id.slice(0, 12)}</span>,
    },
    {
      key: 'plan_id',
      title: 'Plan ID',
      sortable: true,
      render: (_value: unknown, execution: Execution) => <span className="font-mono text-sm">{execution.plan_id.slice(0, 12)}</span>,
    },
    {
      key: 'operation_type',
      title: 'Operation',
      sortable: true,
      render: (_value: unknown, execution: Execution) => (
        <Badge variant="info" size="sm">
          {execution.operation_type}
        </Badge>
      ),
    },
    {
      key: 'state',
      title: 'Status',
      sortable: true,
      render: (_value: unknown, execution: Execution) => {
        const status = statusMap[execution.state] || { variant: 'default', label: execution.state };
        return (
          <Badge variant={status.variant}>
            {status.label}
          </Badge>
        );
      },
    },
    {
      key: 'created_at',
      title: 'Created',
      sortable: true,
      render: (_value: unknown, execution: Execution) => formatDate(execution.created_at),
    },
    {
      key: 'id',
      title: 'Actions',
      align: 'right',
      render: (_value: unknown, execution: Execution) => (
        <Button
          variant="ghost"
          size="sm"
          onClick={() => setSelectedExecutionId(String(execution.id))}
          rightIcon={<ChevronRight className="w-4 h-4" />}
        >
          View Logs
        </Button>
      ),
    },
  ];

  if (isSiteLoading) {
    return (
      <div className="space-y-6">
        <SkeletonCard hasHeader={false} contentLines={2} />
        <div className="grid gap-4 md:grid-cols-4">
          <SkeletonCard hasHeader={false} contentLines={2} />
          <SkeletonCard hasHeader={false} contentLines={2} />
          <SkeletonCard hasHeader={false} contentLines={2} />
          <SkeletonCard hasHeader={false} contentLines={2} />
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Breadcrumb */}
      <div className="flex items-center gap-2 text-sm text-gray-500">
        <Link to="/projects" className="hover:text-gray-700 flex items-center gap-1">
          <FolderOpen className="w-4 h-4" />
          Projects
        </Link>
        <span>/</span>
        <Link to={`/projects/${projectId}`} className="hover:text-gray-700">
          Project
        </Link>
        <span>/</span>
        <Link to={`/projects/${projectId}/sites`} className="hover:text-gray-700 flex items-center gap-1">
          <ArrowLeft className="w-3 h-3" />
          Sites
        </Link>
        <span>/</span>
        <span className="text-gray-900 font-medium">{site?.name || siteId}</span>
      </div>

      {/* Page Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">{site?.name}</h1>
          <div className="mt-1 flex items-center gap-4 text-sm text-gray-500">
            {site?.location_country_code && (
              <span className="flex items-center gap-1">
                <MapPin className="w-3 h-3" />
                {site.location_country_code}
              </span>
            )}
            <span className="flex items-center gap-1">
              <Globe className="w-3 h-3" />
              ID: {siteId?.slice(0, 8)}
            </span>
            <Badge
              variant={site?.connectivity_state === 'ONLINE' ? 'success' : 'error'}
              size="sm"
              withDot
            >
              {site?.connectivity_state || 'UNKNOWN'}
            </Badge>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <Button variant="secondary">Configure</Button>
          <Button
            onClick={() => setIsCreateVMModalOpen(true)}
            leftIcon={<Plus className="w-4 h-4" />}
          >
            Create VM
          </Button>
        </div>
      </div>

      {/* Stats Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <div className="flex items-center gap-4 p-4">
            <div className="w-12 h-12 bg-blue-100 rounded-lg flex items-center justify-center">
              <Server className="w-6 h-6 text-blue-600" />
            </div>
            <div>
              <p className="text-sm text-gray-500">Total VMs</p>
              <p className="text-2xl font-bold text-gray-900">{vms?.length || 0}</p>
            </div>
          </div>
        </Card>
        <Card>
          <div className="flex items-center gap-4 p-4">
            <div className="w-12 h-12 bg-green-100 rounded-lg flex items-center justify-center">
              <Play className="w-6 h-6 text-green-600" />
            </div>
            <div>
              <p className="text-sm text-gray-500">Running</p>
              <p className="text-2xl font-bold text-gray-900">
                {vms?.filter((vm) => vm.state === 'RUNNING').length || 0}
              </p>
            </div>
          </div>
        </Card>
        <Card>
          <div className="flex items-center gap-4 p-4">
            <div className="w-12 h-12 bg-purple-100 rounded-lg flex items-center justify-center">
              <Cpu className="w-6 h-6 text-purple-600" />
            </div>
            <div>
              <p className="text-sm text-gray-500">Total vCPUs</p>
              <p className="text-2xl font-bold text-gray-900">
                {vms?.reduce((sum, vm) => sum + vm.vcpu_count, 0) || 0}
              </p>
            </div>
          </div>
        </Card>
        <Card>
          <div className="flex items-center gap-4 p-4">
            <div className="w-12 h-12 bg-orange-100 rounded-lg flex items-center justify-center">
              <MemoryStick className="w-6 h-6 text-orange-600" />
            </div>
            <div>
              <p className="text-sm text-gray-500">Total Memory</p>
              <p className="text-2xl font-bold text-gray-900">
                {Math.round((vms?.reduce((sum, vm) => sum + vm.memory_mib, 0) || 0) / 1024)} GB
              </p>
            </div>
          </div>
        </Card>
      </div>

      {/* Tabs */}
      <div className="border-b border-gray-200">
        <nav className="-mb-px flex space-x-8">
          {[
            { id: 'vms', label: 'VMs', count: vms?.length },
            { id: 'hosts', label: 'Hosts', count: hosts?.length },
            { id: 'plans', label: 'Executions', count: executions?.length },
          ].map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id as Tab)}
              className={`
                py-4 px-1 border-b-2 font-medium text-sm whitespace-nowrap
                ${activeTab === tab.id
                  ? 'border-blue-500 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                }
              `}
            >
              {tab.label}
              {tab.count !== undefined && (
                <span className="ml-2 bg-gray-100 text-gray-600 py-0.5 px-2 rounded-full text-xs">
                  {tab.count}
                </span>
              )}
            </button>
          ))}
        </nav>
      </div>

      {/* Tab Content */}
      <div className="space-y-4">
        {activeTab === 'vms' && (
          <>
            {isVMsLoading ? (
              <DataTableSkeleton rows={5} columns={6} />
            ) : vms?.length === 0 ? (
              <Card>
                <EmptyState
                  icon="folder"
                  title="No VMs yet"
                  description="Create your first VM to get started"
                  action={{
                    label: 'Create VM',
                    onClick: () => setIsCreateVMModalOpen(true),
                  }}
                />
              </Card>
            ) : (
              <Card noPadding>
                <Table
                  columns={vmColumns}
                  data={vms || []}
                  keyExtractor={(vm) => String(vm.id)}
                  onRowClick={handleVMClick}
                  hoverable
                />
              </Card>
            )}
          </>
        )}

        {activeTab === 'hosts' && (
          <>
            {isHostsLoading ? (
              <DataTableSkeleton rows={3} columns={6} />
            ) : hosts?.length === 0 ? (
              <Card>
                <EmptyState
                  icon="folder"
                  title="No hosts connected"
                  description="Install the edge agent on your hosts to connect them"
                />
              </Card>
            ) : (
              <Card noPadding>
                <Table
                  columns={hostColumns}
                  data={hosts || []}
                  keyExtractor={(host) => String(host.id)}
                  hoverable
                />
              </Card>
            )}
          </>
        )}

        {activeTab === 'plans' && (
          <>
            {isExecutionsLoading ? (
              <DataTableSkeleton rows={5} columns={6} />
            ) : executions?.length === 0 ? (
              <Card>
                <EmptyState
                  icon="folder"
                  title="No executions yet"
                  description="Plan executions will appear here when you create, start, stop, or delete VMs"
                />
              </Card>
            ) : (
              <Card noPadding>
                <Table
                  columns={planColumns}
                  data={executions || []}
                  keyExtractor={(execution) => String(execution.id)}
                  hoverable
                />
              </Card>
            )}
          </>
        )}
      </div>

      {/* Create VM Modal */}
      <VMCreateModal
        isOpen={isCreateVMModalOpen}
        onClose={() => setIsCreateVMModalOpen(false)}
        siteId={siteId || ''}
      />

      {/* Execution Log Viewer Modal */}
      <Modal
        isOpen={!!selectedExecutionId}
        onClose={() => setSelectedExecutionId(null)}
        title="Execution Logs"
        description={`Execution ID: ${selectedExecutionId}`}
        size="lg"
      >
        {selectedExecutionId && (
          <ExecutionLogViewer executionId={selectedExecutionId} />
        )}
      </Modal>

      {/* VM Details Drawer */}
      <Modal
        isOpen={isVMDetailsOpen}
        onClose={() => {
          setIsVMDetailsOpen(false);
          setSelectedVM(null);
        }}
        title={selectedVM?.name}
        description={`ID: ${selectedVM?.id}`}
        size="md"
      >
        {selectedVM && (
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="p-3 bg-gray-50 rounded-lg">
                <p className="text-sm text-gray-500">State</p>
                {getVMStatusBadge(selectedVM.state)}
              </div>
              <div className="p-3 bg-gray-50 rounded-lg">
                <p className="text-sm text-gray-500">vCPUs</p>
                <p className="font-medium">{selectedVM.vcpu_count}</p>
              </div>
              <div className="p-3 bg-gray-50 rounded-lg">
                <p className="text-sm text-gray-500">Memory</p>
                <p className="font-medium">{selectedVM.memory_mib} MiB</p>
              </div>
              <div className="p-3 bg-gray-50 rounded-lg">
                <p className="text-sm text-gray-500">Host ID</p>
                <p className="font-medium text-xs">{selectedVM.host_id}</p>
              </div>
            </div>
            <div className="flex gap-2 pt-4 border-t">
              {selectedVM.state === 'STOPPED' ? (
                <Button
                  onClick={() => {
                    handleStartVM(selectedVM);
                    setIsVMDetailsOpen(false);
                  }}
                  leftIcon={<Play className="w-4 h-4" />}
                  loading={applyPlanMutation.isPending}
                >
                  Start VM
                </Button>
              ) : (
                <Button
                  onClick={() => {
                    handleStopVM(selectedVM);
                    setIsVMDetailsOpen(false);
                  }}
                  variant="secondary"
                  leftIcon={<Square className="w-4 h-4" />}
                  loading={applyPlanMutation.isPending}
                >
                  Stop VM
                </Button>
              )}
              <VMActionsMenu
                vm={selectedVM}
                onStart={() => {
                  handleStartVM(selectedVM);
                  setIsVMDetailsOpen(false);
                }}
                onStop={() => {
                  handleStopVM(selectedVM);
                  setIsVMDetailsOpen(false);
                }}
                onDelete={() => {
                  handleDeleteVM(selectedVM);
                  setIsVMDetailsOpen(false);
                }}
                isLoading={applyPlanMutation.isPending}
              />
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}
