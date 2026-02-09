import { useState, useCallback } from 'react';
import { Copy, Check, AlertCircle, Building2, Key } from 'lucide-react';
import {
  Modal,
  Button,
  Input,
  toast,
} from '@/components/common';
import { useCreateTenant, useCreateAPIKey } from '@/api/hooks';
import { Tenant, CreateAPIKeyResponse } from '@/api/types';

interface CreateTenantModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess?: (tenant: Tenant) => void;
}

interface FormData {
  name: string;
  slug: string;
  primaryRegion: string;
  dataRetentionDays: string;
}

interface FormErrors {
  name?: string;
  slug?: string;
  primaryRegion?: string;
  dataRetentionDays?: string;
}

export function CreateTenantModal({ isOpen, onClose, onSuccess }: CreateTenantModalProps) {
  const [formData, setFormData] = useState<FormData>({
    name: '',
    slug: '',
    primaryRegion: 'us-east-1',
    dataRetentionDays: '30',
  });
  const [errors, setErrors] = useState<FormErrors>({});
  const [createdTenant, setCreatedTenant] = useState<Tenant | null>(null);
  const [apiKeyResponse, setApiKeyResponse] = useState<CreateAPIKeyResponse | null>(null);
  const [copied, setCopied] = useState(false);

  const createTenant = useCreateTenant();
  const createAPIKey = useCreateAPIKey();

  // Reset form when modal opens/closes
  const handleClose = useCallback(() => {
    setFormData({
      name: '',
      slug: '',
      primaryRegion: 'us-east-1',
      dataRetentionDays: '30',
    });
    setErrors({});
    setCreatedTenant(null);
    setApiKeyResponse(null);
    setCopied(false);
    onClose();
  }, [onClose]);

  // Validate form
  const validateForm = (): boolean => {
    const newErrors: FormErrors = {};

    if (!formData.name.trim()) {
      newErrors.name = 'Name is required';
    }

    if (!formData.slug.trim()) {
      newErrors.slug = 'Slug is required';
    } else if (!/^[a-z0-9-]+$/.test(formData.slug)) {
      newErrors.slug = 'Slug must contain only lowercase letters, numbers, and hyphens';
    }

    if (!formData.primaryRegion.trim()) {
      newErrors.primaryRegion = 'Primary region is required';
    }

    const retentionDays = parseInt(formData.dataRetentionDays, 10);
    if (isNaN(retentionDays) || retentionDays < 1) {
      newErrors.dataRetentionDays = 'Must be a positive number';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  // Generate slug from name
  const generateSlug = (name: string): string => {
    return name
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/^-+|-+$/g, '');
  };

  // Handle name change with auto-generated slug
  const handleNameChange = (value: string) => {
    setFormData((prev) => ({
      ...prev,
      name: value,
      slug: prev.slug || generateSlug(value),
    }));
    if (errors.name) {
      setErrors((prev) => ({ ...prev, name: undefined }));
    }
  };

  // Handle form submission
  const handleSubmit = async () => {
    if (!validateForm()) return;

    try {
      // Create tenant
      const tenant = await createTenant.mutateAsync({
        name: formData.name,
        slug: formData.slug,
        primary_region: formData.primaryRegion,
        data_retention_days: parseInt(formData.dataRetentionDays, 10),
      });

      setCreatedTenant(tenant);

      // Create initial API key for the tenant
      const apiKey = await createAPIKey.mutateAsync({
        tenantId: tenant.id,
        data: {
          name: 'Default Admin Key',
          expires_in_seconds: 365 * 24 * 60 * 60, // 1 year
        },
      });

      setApiKeyResponse(apiKey);
      onSuccess?.(tenant);
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to create tenant';
      toast.error(message);
    }
  };

  // Copy API key to clipboard
  const handleCopyKey = async () => {
    if (!apiKeyResponse?.api_key) return;

    try {
      await navigator.clipboard.writeText(apiKeyResponse.api_key);
      setCopied(true);
      toast.success('API key copied to clipboard');
      setTimeout(() => setCopied(false), 2000);
    } catch {
      toast.error('Failed to copy API key');
    }
  };

  // Success state - show API key
  if (createdTenant && apiKeyResponse) {
    return (
      <Modal
        isOpen={isOpen}
        onClose={handleClose}
        title="Tenant Created Successfully"
        description={`${createdTenant.name} has been created with the following API key.`}
        size="md"
        footer={
          <div className="flex justify-end gap-3">
            <Button variant="secondary" onClick={handleClose}>
              Close
            </Button>
            <Button
              variant="primary"
              leftIcon={copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
              onClick={handleCopyKey}
            >
              {copied ? 'Copied!' : 'Copy API Key'}
            </Button>
          </div>
        }
      >
        <div className="space-y-4">
          {/* Warning banner */}
          <div className="rounded-lg bg-yellow-50 p-4 dark:bg-yellow-900/20">
            <div className="flex gap-3">
              <AlertCircle className="h-5 w-5 text-yellow-600 dark:text-yellow-400" />
              <div>
                <h4 className="text-sm font-medium text-yellow-800 dark:text-yellow-300">
                  Save this API key now
                </h4>
                <p className="mt-1 text-sm text-yellow-700 dark:text-yellow-400">
                  This key will only be shown once. You won&apos;t be able to see it again.
                </p>
              </div>
            </div>
          </div>

          {/* Tenant info */}
          <div className="rounded-lg border border-gray-200 p-4 dark:border-gray-700">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-100">
                <Building2 className="h-5 w-5 text-blue-600" />
              </div>
              <div>
                <p className="font-medium text-gray-900">{createdTenant.name}</p>
                <p className="text-sm text-gray-500">{createdTenant.slug}</p>
              </div>
            </div>
          </div>

          {/* API Key display */}
          <div>
            <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
              API Key
            </label>
            <div className="relative">
              <div className="flex items-center gap-2 rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-gray-700 dark:bg-gray-900">
                <Key className="h-4 w-4 text-gray-400" />
                <code className="flex-1 break-all text-sm font-mono text-gray-900 dark:text-gray-100">
                  {apiKeyResponse.api_key}
                </code>
              </div>
            </div>
            <p className="mt-1.5 text-xs text-gray-500">
              Expires: {new Date(apiKeyResponse.expires_at).toLocaleDateString()}
            </p>
          </div>
        </div>
      </Modal>
    );
  }

  // Form state
  return (
    <Modal
      isOpen={isOpen}
      onClose={handleClose}
      title="Create New Tenant"
      description="Create a new tenant organization. An initial API key will be generated."
      size="md"
      footer={
        <div className="flex justify-end gap-3">
          <Button variant="secondary" onClick={handleClose}>
            Cancel
          </Button>
          <Button
            variant="primary"
            onClick={handleSubmit}
            loading={createTenant.isPending || createAPIKey.isPending}
          >
            Create Tenant
          </Button>
        </div>
      }
    >
      <div className="space-y-4">
        <Input
          label="Organization Name"
          placeholder="e.g., Acme Corporation"
          value={formData.name}
          onChange={(e) => handleNameChange(e.target.value)}
          error={errors.name}
          data-testid="tenant-name-input"
          required
        />

        <Input
          label="Slug"
          placeholder="e.g., acme-corp"
          value={formData.slug}
          onChange={(e) => {
            setFormData((prev) => ({ ...prev, slug: e.target.value }));
            if (errors.slug) {
              setErrors((prev) => ({ ...prev, slug: undefined }));
            }
          }}
          error={errors.slug}
          helperText="Used in URLs and API calls. Lowercase letters, numbers, and hyphens only."
          data-testid="tenant-slug-input"
          required
        />

        <Input
          label="Primary Region"
          placeholder="e.g., us-east-1"
          value={formData.primaryRegion}
          onChange={(e) => {
            setFormData((prev) => ({ ...prev, primaryRegion: e.target.value }));
            if (errors.primaryRegion) {
              setErrors((prev) => ({ ...prev, primaryRegion: undefined }));
            }
          }}
          error={errors.primaryRegion}
          required
        />

        <Input
          label="Data Retention (days)"
          type="number"
          min={1}
          value={formData.dataRetentionDays}
          onChange={(e) => {
            setFormData((prev) => ({ ...prev, dataRetentionDays: e.target.value }));
            if (errors.dataRetentionDays) {
              setErrors((prev) => ({ ...prev, dataRetentionDays: undefined }));
            }
          }}
          error={errors.dataRetentionDays}
          helperText="Number of days to retain audit logs and metrics"
          required
        />
      </div>
    </Modal>
  );
}
