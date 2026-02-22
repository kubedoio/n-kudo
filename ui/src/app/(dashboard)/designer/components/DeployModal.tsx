'use client';

import { useState, useEffect } from 'react';
import { X, Rocket, AlertCircle, CheckCircle2, Server, Network, HardDrive, Loader2 } from 'lucide-react';
import { cn } from '@/lib/utils';
import { SelectInput } from './forms/SelectInput';
import {
  generatePlanFromDesign,
  validateDesign,
  getActionDescription,
  estimateDeploymentTime,
  type PlanAction,
} from '../plan-generator';
import type { DesignerNode, DesignerEdge } from '../types';

interface DeployModalProps {
  nodes: DesignerNode[];
  edges: DesignerEdge[];
  onClose: () => void;
}

// Mock sites for deployment
const mockSites = [
  { value: 'site-1', label: 'us-east-1 (Virginia)' },
  { value: 'site-2', label: 'eu-west-1 (Ireland)' },
  { value: 'site-3', label: 'ap-southeast-1 (Singapore)' },
];

export function DeployModal({ nodes, edges, onClose }: DeployModalProps) {
  const [selectedSite, setSelectedSite] = useState('');
  const [isDeploying, setIsDeploying] = useState(false);
  const [deploymentStatus, setDeploymentStatus] = useState<'idle' | 'success' | 'error'>('idle');
  const [error, setError] = useState<string | null>(null);
  const [validationResult, setValidationResult] = useState<{
    valid: boolean;
    errors: string[];
  } | null>(null);
  const [planActions, setPlanActions] = useState<PlanAction[]>([]);

  // Generate plan and validate on mount
  useEffect(() => {
    if (nodes.length > 0) {
      const plan = generatePlanFromDesign(nodes, edges);
      setPlanActions(plan);
      const result = validateDesign(nodes, edges);
      setValidationResult(result);
    }
  }, [nodes, edges]);

  const handleDeploy = async () => {
    if (!selectedSite) {
      setError('Please select a deployment site');
      return;
    }

    const result = validateDesign(nodes, edges);
    if (!result.valid) {
      setValidationResult(result);
      setError('Please fix validation errors before deploying');
      return;
    }

    setIsDeploying(true);
    setError(null);

    try {
      // Simulate API call
      await new Promise((resolve) => setTimeout(resolve, 2000));
      setDeploymentStatus('success');
    } catch (e) {
      setDeploymentStatus('error');
      setError('Deployment failed. Please try again.');
    } finally {
      setIsDeploying(false);
    }
  };

  const nodeCounts = {
    vm: nodes.filter((n) => n.type === 'vm').length,
    network: nodes.filter((n) => n.type === 'network').length,
    volume: nodes.filter((n) => n.type === 'volume').length,
  };

  const estimatedTime = estimateDeploymentTime(planActions);
  const estimatedTimeStr = estimatedTime < 60 
    ? `${estimatedTime}s` 
    : `${Math.ceil(estimatedTime / 60)}m`;

  // Success state
  if (deploymentStatus === 'success') {
    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
        <div className="w-full max-w-md rounded-xl bg-white p-6 text-center shadow-xl">
          <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-emerald-100">
            <CheckCircle2 className="h-8 w-8 text-emerald-600" />
          </div>
          <h2 className="mb-2 text-xl font-semibold text-slate-900">Deployment Initiated!</h2>
          <p className="mb-6 text-sm text-slate-500">
            Your infrastructure is being deployed to {mockSites.find(s => s.value === selectedSite)?.label}. 
            You can track the progress in the Executions page.
          </p>
          <button
            onClick={onClose}
            className="rounded-lg bg-indigo-600 px-6 py-2 text-sm font-medium text-white hover:bg-indigo-700"
          >
            Close
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className="w-full max-w-lg rounded-xl bg-white shadow-xl">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-slate-200 px-6 py-4">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-indigo-100">
              <Rocket className="h-5 w-5 text-indigo-600" />
            </div>
            <div>
              <h2 className="text-lg font-semibold text-slate-900">Deploy Infrastructure</h2>
              <p className="text-sm text-slate-500">
                {planActions.length} actions • ~{estimatedTimeStr}
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="flex h-8 w-8 items-center justify-center rounded-lg text-slate-400 hover:bg-slate-100 hover:text-slate-600"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Content */}
        <div className="space-y-4 p-6">
          {error && (
            <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-600">
              <AlertCircle className="h-4 w-4" />
              {error}
            </div>
          )}

          {validationResult && !validationResult.valid && (
            <div className="rounded-lg bg-amber-50 p-4">
              <div className="mb-2 flex items-center gap-2 text-sm font-medium text-amber-800">
                <AlertCircle className="h-4 w-4" />
                Validation Issues
              </div>
              <ul className="space-y-1 text-xs text-amber-700">
                {validationResult.errors.map((err, idx) => (
                  <li key={idx} className="flex items-start gap-1">
                    <span className="mt-1 h-1 w-1 rounded-full bg-amber-500" />
                    {err}
                  </li>
                ))}
              </ul>
            </div>
          )}

          {nodes.length === 0 && (
            <div className="rounded-lg bg-red-50 p-4 text-sm text-red-600">
              <div className="flex items-center gap-2 font-medium">
                <AlertCircle className="h-4 w-4" />
                Empty Template
              </div>
              <p className="mt-1 text-xs">Add at least one node to your design before deploying.</p>
            </div>
          )}

          <div className="space-y-2">
            <label className="text-sm font-medium text-slate-700">
              Deployment Site <span className="text-red-500">*</span>
            </label>
            <SelectInput
              value={selectedSite}
              onChange={setSelectedSite}
              options={[{ value: '', label: 'Select a site...' }, ...mockSites]}
            />
          </div>

          {/* Resources Summary */}
          {nodes.length > 0 && (
            <div className="rounded-lg bg-slate-50 p-4">
              <h3 className="mb-3 text-sm font-medium text-slate-700">Resources to Deploy</h3>
              <div className="grid grid-cols-3 gap-3">
                <div className="rounded-lg bg-white p-3 text-center shadow-sm">
                  <div className="mx-auto mb-1 flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-50">
                    <Server className="h-4 w-4 text-indigo-600" />
                  </div>
                  <p className="text-lg font-semibold text-slate-900">{nodeCounts.vm}</p>
                  <p className="text-xs text-slate-500">VMs</p>
                </div>
                <div className="rounded-lg bg-white p-3 text-center shadow-sm">
                  <div className="mx-auto mb-1 flex h-8 w-8 items-center justify-center rounded-lg bg-emerald-50">
                    <Network className="h-4 w-4 text-emerald-600" />
                  </div>
                  <p className="text-lg font-semibold text-slate-900">{nodeCounts.network}</p>
                  <p className="text-xs text-slate-500">Networks</p>
                </div>
                <div className="rounded-lg bg-white p-3 text-center shadow-sm">
                  <div className="mx-auto mb-1 flex h-8 w-8 items-center justify-center rounded-lg bg-amber-50">
                    <HardDrive className="h-4 w-4 text-amber-600" />
                  </div>
                  <p className="text-lg font-semibold text-slate-900">{nodeCounts.volume}</p>
                  <p className="text-xs text-slate-500">Volumes</p>
                </div>
              </div>
              <div className="mt-3 flex items-center justify-between text-sm">
                <span className="text-slate-500">Total Connections:</span>
                <span className="font-medium text-slate-700">{edges.length}</span>
              </div>
            </div>
          )}

          {/* Execution Plan Preview */}
          {planActions.length > 0 && validationResult?.valid && (
            <div className="rounded-lg border border-slate-200 p-4">
              <h3 className="mb-2 text-sm font-medium text-slate-700">Execution Order</h3>
              <div className="max-h-32 space-y-2 overflow-y-auto">
                {planActions.map((action, idx) => (
                  <div key={action.id} className="flex items-center gap-2 text-sm">
                    <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-indigo-100 text-xs font-medium text-indigo-600">
                      {idx + 1}
                    </span>
                    <span className="text-slate-600">{getActionDescription(action)}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex justify-end gap-3 border-t border-slate-200 px-6 py-4">
          <button
            onClick={onClose}
            disabled={isDeploying}
            className="rounded-lg border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-50"
          >
            Cancel
          </button>
          <button
            onClick={handleDeploy}
            disabled={isDeploying || nodes.length === 0 || !selectedSite}
            className={cn(
              'flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium text-white',
              isDeploying || nodes.length === 0 || !selectedSite
                ? 'cursor-not-allowed bg-slate-300'
                : 'bg-indigo-600 hover:bg-indigo-700'
            )}
          >
            {isDeploying ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" />
                Deploying...
              </>
            ) : (
              <>
                <Rocket className="h-4 w-4" />
                Deploy
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
}
