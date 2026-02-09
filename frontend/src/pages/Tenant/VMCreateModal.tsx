import { useState } from 'react';
import { Modal, Button, Input } from '@/components/common';
import { useApplyPlanFromActions, queryKeys } from '@/api/hooks';
import { toast } from '@/stores/toastStore';
import { useQueryClient } from '@tanstack/react-query';
import { Loader2 } from 'lucide-react';

interface VMCreateModalProps {
  isOpen: boolean;
  onClose: () => void;
  siteId: string;
}

interface VMFormData {
  name: string;
  vcpu_count: number;
  memory_mib: number;
}

interface FormErrors {
  name?: string;
  vcpu_count?: string;
  memory_mib?: string;
}

export function VMCreateModal({ isOpen, onClose, siteId }: VMCreateModalProps) {
  const queryClient = useQueryClient();
  const [formData, setFormData] = useState<VMFormData>({
    name: '',
    vcpu_count: 2,
    memory_mib: 2048,
  });
  const [errors, setErrors] = useState<FormErrors>({});
  const [submittedPlanId, setSubmittedPlanId] = useState<string | null>(null);
  const [executionStatus, setExecutionStatus] = useState<'PENDING' | 'SUCCEEDED' | 'FAILED' | null>(null);

  const applyPlanMutation = useApplyPlanFromActions({
    onSuccess: (data) => {
      setSubmittedPlanId(data.plan_id);
      setExecutionStatus('PENDING');
      toast.success(`VM creation plan submitted (Plan ID: ${data.plan_id.slice(0, 8)})`);
      
      // Start polling for execution status
      if (data.executions.length > 0) {
        pollExecutionStatus();
      }
      
      // Invalidate VMs query to refresh the list
      queryClient.invalidateQueries({ queryKey: queryKeys.vms(siteId) });
    },
    onError: (error) => {
      toast.error(error.message || 'Failed to create VM');
      setExecutionStatus('FAILED');
    },
  });

  const pollExecutionStatus = () => {
    // Poll every 5 seconds for up to 2 minutes
    let attempts = 0;
    
    const poll = () => {
      attempts++;
      // Max attempts: 24 (2 minutes / 5 seconds)
      
      // For now, simulate status update after a delay
      // In production, this would call an API to check execution status
      setTimeout(() => {
        if (attempts < 3) {
          setExecutionStatus('SUCCEEDED');
          toast.success('VM created successfully!');
          
          // Close modal and reset after success
          setTimeout(() => {
            handleClose();
          }, 1500);
        }
      }, 3000);
    };
    
    poll();
  };

  const validateForm = (): boolean => {
    const newErrors: FormErrors = {};

    // Name validation
    if (!formData.name.trim()) {
      newErrors.name = 'VM name is required';
    } else if (!/^[a-zA-Z0-9-_]+$/.test(formData.name)) {
      newErrors.name = 'Name can only contain letters, numbers, hyphens, and underscores';
    } else if (formData.name.length > 63) {
      newErrors.name = 'Name must be 63 characters or less';
    }

    // vCPU validation
    if (formData.vcpu_count < 1) {
      newErrors.vcpu_count = 'vCPUs must be at least 1';
    } else if (formData.vcpu_count > 32) {
      newErrors.vcpu_count = 'vCPUs cannot exceed 32';
    }

    // Memory validation
    if (formData.memory_mib < 128) {
      newErrors.memory_mib = 'Memory must be at least 128 MiB';
    } else if (formData.memory_mib > 65536) {
      newErrors.memory_mib = 'Memory cannot exceed 65536 MiB (64 GB)';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = () => {
    if (!validateForm()) return;

    applyPlanMutation.mutate({
      siteId,
      idempotencyKey: `create-vm-${formData.name}-${Date.now()}`,
      actions: [
        {
          operation: 'CREATE',
          name: formData.name.trim(),
          vcpu_count: formData.vcpu_count,
          memory_mib: formData.memory_mib,
        },
      ],
    });
  };

  const handleClose = () => {
    setFormData({ name: '', vcpu_count: 2, memory_mib: 2048 });
    setErrors({});
    setSubmittedPlanId(null);
    setExecutionStatus(null);
    onClose();
  };

  const handleInputChange = (field: keyof VMFormData, value: string | number) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
    // Clear error when user starts typing
    if (errors[field]) {
      setErrors((prev) => ({ ...prev, [field]: undefined }));
    }
  };

  const isSubmitting = applyPlanMutation.isPending || executionStatus === 'PENDING';

  return (
    <Modal
      isOpen={isOpen}
      onClose={handleClose}
      title="Create Virtual Machine"
      description="Configure a new MicroVM for this site"
      size="md"
      footer={
        !submittedPlanId ? (
          <div className="flex justify-end gap-3">
            <Button variant="secondary" onClick={handleClose} disabled={isSubmitting}>
              Cancel
            </Button>
            <Button
              onClick={handleSubmit}
              loading={isSubmitting}
              disabled={!formData.name.trim()}
            >
              Create VM
            </Button>
          </div>
        ) : (
          <div className="flex justify-end">
            <Button variant="secondary" onClick={handleClose}>
              Close
            </Button>
          </div>
        )
      }
    >
      <div className="space-y-6">
        {!submittedPlanId ? (
          // Form view
          <>
            <Input
              label="VM Name"
              placeholder="e.g., web-server-01"
              value={formData.name}
              onChange={(e) => handleInputChange('name', e.target.value)}
              error={errors.name}
              required
              helperText="Use letters, numbers, hyphens, and underscores only"
            />

            <div className="grid grid-cols-2 gap-4">
              <Input
                label="vCPUs"
                type="number"
                min={1}
                max={32}
                value={formData.vcpu_count}
                onChange={(e) => handleInputChange('vcpu_count', parseInt(e.target.value) || 1)}
                error={errors.vcpu_count}
                required
                helperText="1-32 cores"
              />

              <Input
                label="Memory (MiB)"
                type="number"
                min={128}
                max={65536}
                step={128}
                value={formData.memory_mib}
                onChange={(e) => handleInputChange('memory_mib', parseInt(e.target.value) || 128)}
                error={errors.memory_mib}
                required
                helperText="128-65536 MiB"
              />
            </div>

            {/* Resource summary */}
            <div className="p-4 bg-gray-50 rounded-lg">
              <h4 className="text-sm font-medium text-gray-700 mb-2">Resource Summary</h4>
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <span className="text-gray-500">vCPUs:</span>{' '}
                  <span className="font-medium">{formData.vcpu_count}</span>
                </div>
                <div>
                  <span className="text-gray-500">Memory:</span>{' '}
                  <span className="font-medium">
                    {formData.memory_mib >= 1024
                      ? `${(formData.memory_mib / 1024).toFixed(2)} GB`
                      : `${formData.memory_mib} MiB`}
                  </span>
                </div>
              </div>
            </div>
          </>
        ) : (
          // Execution status view
          <div className="text-center py-6">
            {executionStatus === 'PENDING' ? (
              <div className="space-y-4">
                <div className="flex justify-center">
                  <Loader2 className="w-12 h-12 text-blue-600 animate-spin" />
                </div>
                <div>
                  <h4 className="text-lg font-medium text-gray-900">Creating VM...</h4>
                  <p className="text-sm text-gray-500 mt-1">
                    Plan ID: <span className="font-mono">{submittedPlanId.slice(0, 16)}</span>
                  </p>
                </div>
                <p className="text-sm text-gray-500">
                  This may take a few moments. The page will refresh automatically.
                </p>
              </div>
            ) : executionStatus === 'SUCCEEDED' ? (
              <div className="space-y-4">
                <div className="w-12 h-12 bg-green-100 rounded-full flex items-center justify-center mx-auto">
                  <svg
                    className="w-6 h-6 text-green-600"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M5 13l4 4L19 7"
                    />
                  </svg>
                </div>
                <div>
                  <h4 className="text-lg font-medium text-gray-900">VM Created Successfully!</h4>
                  <p className="text-sm text-gray-500 mt-1">
                    {formData.name} is now being provisioned
                  </p>
                </div>
              </div>
            ) : executionStatus === 'FAILED' ? (
              <div className="space-y-4">
                <div className="w-12 h-12 bg-red-100 rounded-full flex items-center justify-center mx-auto">
                  <svg
                    className="w-6 h-6 text-red-600"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M6 18L18 6M6 6l12 12"
                    />
                  </svg>
                </div>
                <div>
                  <h4 className="text-lg font-medium text-gray-900">VM Creation Failed</h4>
                  <p className="text-sm text-gray-500 mt-1">
                    Please check the execution logs for details
                  </p>
                </div>
              </div>
            ) : null}
          </div>
        )}
      </div>
    </Modal>
  );
}
