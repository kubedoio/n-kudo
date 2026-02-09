import { useState, useCallback } from 'react';
import { Copy, Check, AlertCircle, Key, Shield } from 'lucide-react';
import {
  Modal,
  Button,
  Input,
  toast,
} from '@/components/common';
import { useCreateAPIKey } from '@/api/hooks';
import { CreateAPIKeyResponse } from '@/api/types';

interface CreateAPIKeyModalProps {
  isOpen: boolean;
  onClose: () => void;
  tenantId: string;
}

interface FormData {
  name: string;
  expiresInDays: string;
}

interface FormErrors {
  name?: string;
  expiresInDays?: string;
}

export function CreateAPIKeyModal({ isOpen, onClose, tenantId }: CreateAPIKeyModalProps) {
  const [formData, setFormData] = useState<FormData>({
    name: '',
    expiresInDays: '30',
  });
  const [errors, setErrors] = useState<FormErrors>({});
  const [apiKeyResponse, setApiKeyResponse] = useState<CreateAPIKeyResponse | null>(null);
  const [copied, setCopied] = useState(false);

  const createAPIKey = useCreateAPIKey();

  // Reset form when modal closes
  const handleClose = useCallback(() => {
    setFormData({
      name: '',
      expiresInDays: '30',
    });
    setErrors({});
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

    const expiresInDays = parseInt(formData.expiresInDays, 10);
    if (isNaN(expiresInDays) || expiresInDays < 1) {
      newErrors.expiresInDays = 'Must be a positive number';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  // Handle form submission
  const handleSubmit = async () => {
    if (!validateForm()) return;

    try {
      const expiresInSeconds = parseInt(formData.expiresInDays, 10) * 24 * 60 * 60;
      const response = await createAPIKey.mutateAsync({
        tenantId,
        data: {
          name: formData.name,
          expires_in_seconds: expiresInSeconds,
        },
      });

      setApiKeyResponse(response);
      toast.success('API key created successfully');
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to create API key';
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
  if (apiKeyResponse) {
    return (
      <Modal
        isOpen={isOpen}
        onClose={handleClose}
        title="API Key Created"
        description="Your new API key has been created. Copy it now - it won't be shown again."
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
              <AlertCircle className="h-5 w-5 flex-shrink-0 text-yellow-600 dark:text-yellow-400" />
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

          {/* Key info */}
          <div className="rounded-lg border border-gray-200 p-4 dark:border-gray-700">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-purple-100">
                <Shield className="h-5 w-5 text-purple-600" />
              </div>
              <div>
                <p className="font-medium text-gray-900">{apiKeyResponse.name}</p>
                <p className="text-sm text-gray-500">
                  Expires: {new Date(apiKeyResponse.expires_at).toLocaleDateString()}
                </p>
              </div>
            </div>
          </div>

          {/* API Key display */}
          <div>
            <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
              API Key
            </label>
            <div className="relative">
              <div className="flex items-center gap-2 rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-gray-700 dark:bg-gray-900" data-testid="api-key-value">
                <Key className="h-4 w-4 flex-shrink-0 text-gray-400" />
                <code className="flex-1 break-all text-sm font-mono text-gray-900 dark:text-gray-100">
                  {apiKeyResponse.api_key}
                </code>
              </div>
            </div>
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
      title="Create API Key"
      description="Create a new API key for accessing the n-kudo API."
      size="md"
      footer={
        <div className="flex justify-end gap-3">
          <Button variant="secondary" onClick={handleClose}>
            Cancel
          </Button>
          <Button
            variant="primary"
            onClick={handleSubmit}
            loading={createAPIKey.isPending}
          >
            Create API Key
          </Button>
        </div>
      }
    >
      <div className="space-y-4">
        <Input
          label="Key Name"
          placeholder="e.g., Production API Key"
          value={formData.name}
          onChange={(e) => {
            setFormData((prev) => ({ ...prev, name: e.target.value }));
            if (errors.name) {
              setErrors((prev) => ({ ...prev, name: undefined }));
            }
          }}
          error={errors.name}
          data-testid="api-key-name-input"
          required
        />

        <Input
          label="Expires In (days)"
          type="number"
          min={1}
          value={formData.expiresInDays}
          onChange={(e) => {
            setFormData((prev) => ({ ...prev, expiresInDays: e.target.value }));
            if (errors.expiresInDays) {
              setErrors((prev) => ({ ...prev, expiresInDays: undefined }));
            }
          }}
          error={errors.expiresInDays}
          helperText="Number of days until the key expires"
          required
        />

        <div className="rounded-lg border border-blue-200 bg-blue-50 p-4 dark:border-blue-800 dark:bg-blue-900/20">
          <div className="flex gap-3">
            <AlertCircle className="h-5 w-5 flex-shrink-0 text-blue-600 dark:text-blue-400" />
            <div>
              <p className="text-sm font-medium text-blue-800 dark:text-blue-300">
                Important
              </p>
              <p className="mt-1 text-sm text-blue-700 dark:text-blue-400">
                The API key will only be shown once after creation. Make sure to copy and store it securely.
              </p>
            </div>
          </div>
        </div>
      </div>
    </Modal>
  );
}
