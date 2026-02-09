import { useState, useCallback } from 'react';
import { Copy, Check, AlertCircle, Clock, MapPin, Key, Terminal } from 'lucide-react';
import {
  Modal,
  Button,
  Select,
  Input,
  Badge,
  toast,
} from '@/components/common';
import { useIssueToken } from '@/api/hooks';
import { Site, IssueEnrollmentTokenResponse } from '@/api/types';

interface IssueTokenModalProps {
  isOpen: boolean;
  onClose: () => void;
  tenantId: string;
  sites: Site[];
}

export function IssueTokenModal({ isOpen, onClose, tenantId, sites }: IssueTokenModalProps) {
  const [selectedSiteId, setSelectedSiteId] = useState('');
  const [ttlMinutes, setTtlMinutes] = useState('15');
  const [error, setError] = useState<string | null>(null);
  const [tokenResponse, setTokenResponse] = useState<IssueEnrollmentTokenResponse | null>(null);
  const [copied, setCopied] = useState(false);

  const issueToken = useIssueToken();

  // Reset state when modal closes
  const handleClose = useCallback(() => {
    setSelectedSiteId('');
    setTtlMinutes('15');
    setError(null);
    setTokenResponse(null);
    setCopied(false);
    onClose();
  }, [onClose]);

  // Validate form
  const validateForm = (): boolean => {
    if (!selectedSiteId) {
      setError('Please select a site');
      return false;
    }
    const ttl = parseInt(ttlMinutes, 10);
    if (isNaN(ttl) || ttl < 1 || ttl > 1440) {
      setError('TTL must be between 1 and 1440 minutes (24 hours)');
      return false;
    }
    setError(null);
    return true;
  };

  // Handle form submission
  const handleSubmit = async () => {
    if (!validateForm()) return;

    try {
      const response = await issueToken.mutateAsync({
        tenantId,
        siteId: selectedSiteId,
        ttl: parseInt(ttlMinutes, 10) * 60, // Convert to seconds
      });
      setTokenResponse(response);
      toast.success('Enrollment token issued successfully');
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to issue token';
      toast.error(message);
    }
  };

  // Copy token to clipboard
  const handleCopyToken = async () => {
    if (!tokenResponse?.token) return;

    try {
      await navigator.clipboard.writeText(tokenResponse.token);
      setCopied(true);
      toast.success('Token copied to clipboard');
      setTimeout(() => setCopied(false), 2000);
    } catch {
      toast.error('Failed to copy token');
    }
  };

  // Copy install command to clipboard
  const handleCopyCommand = async () => {
    if (!tokenResponse?.token) return;

    const command = `curl -fsSL https://n-kudo.io/install.sh | sudo bash -s -- --token ${tokenResponse.token}`;
    try {
      await navigator.clipboard.writeText(command);
      toast.success('Install command copied to clipboard');
    } catch {
      toast.error('Failed to copy command');
    }
  };

  // Get site options for select
  const siteOptions = sites.map((site) => ({
    value: site.id,
    label: site.name,
  }));

  // Get selected site name
  const selectedSite = sites.find((s) => s.id === selectedSiteId);

  // Success state - show token
  if (tokenResponse) {
    const expiresAt = new Date(tokenResponse.expires_at);
    const timeLeft = Math.max(0, Math.floor((expiresAt.getTime() - Date.now()) / 60000));

    return (
      <Modal
        isOpen={isOpen}
        onClose={handleClose}
        title="Enrollment Token Issued"
        description={`Use this token to enroll an edge agent for ${selectedSite?.name}.`}
        size="lg"
        footer={
          <div className="flex justify-end gap-3">
            <Button variant="secondary" onClick={handleClose}>
              Close
            </Button>
            <Button
              variant="primary"
              leftIcon={copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
              onClick={handleCopyToken}
            >
              {copied ? 'Copied!' : 'Copy Token'}
            </Button>
          </div>
        }
      >
        <div className="space-y-6">
          {/* Warning banner */}
          <div className="rounded-lg bg-yellow-50 p-4 dark:bg-yellow-900/20">
            <div className="flex gap-3">
              <AlertCircle className="h-5 w-5 flex-shrink-0 text-yellow-600 dark:text-yellow-400" />
              <div>
                <h4 className="text-sm font-medium text-yellow-800 dark:text-yellow-300">
                  One-time use token
                </h4>
                <p className="mt-1 text-sm text-yellow-700 dark:text-yellow-400">
                  This token can only be used once and expires in {timeLeft} minutes. 
                  Copy it now - it won&apos;t be shown again.
                </p>
              </div>
            </div>
          </div>

          {/* Site info */}
          <div className="rounded-lg border border-gray-200 p-4 dark:border-gray-700">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-green-100">
                <MapPin className="h-5 w-5 text-green-600" />
              </div>
              <div>
                <p className="font-medium text-gray-900">{selectedSite?.name}</p>
                <p className="text-sm text-gray-500">
                  {selectedSite?.external_key || selectedSite?.id.slice(0, 8)}
                </p>
              </div>
            </div>
          </div>

          {/* Token display */}
          <div>
            <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
              Enrollment Token
            </label>
            <div className="relative">
              <div className="flex items-center gap-2 rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-gray-700 dark:bg-gray-900">
                <Key className="h-4 w-4 flex-shrink-0 text-gray-400" />
                <code className="flex-1 break-all text-sm font-mono text-gray-900 dark:text-gray-100">
                  {tokenResponse.token}
                </code>
              </div>
            </div>
            <div className="mt-2 flex items-center gap-2 text-sm text-gray-500">
              <Clock className="h-4 w-4" />
              <span>Expires at {expiresAt.toLocaleString()}</span>
            </div>
          </div>

          {/* Install command */}
          <div>
            <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
              Quick Install Command
            </label>
            <div className="relative">
              <div className="flex items-start gap-2 rounded-lg border border-gray-200 bg-gray-900 p-3">
                <Terminal className="mt-0.5 h-4 w-4 flex-shrink-0 text-gray-400" />
                <code className="flex-1 break-all text-sm font-mono text-gray-100">
                  curl -fsSL https://n-kudo.io/install.sh | sudo bash -s -- --token {tokenResponse.token}
                </code>
              </div>
              <Button
                variant="ghost"
                size="sm"
                className="absolute right-2 top-2"
                leftIcon={<Copy className="h-4 w-4" />}
                onClick={handleCopyCommand}
              >
                Copy
              </Button>
            </div>
            <p className="mt-1.5 text-xs text-gray-500">
              Run this command on the edge host to install and enroll the agent.
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
      title="Issue Enrollment Token"
      description="Create a one-time token for enrolling an edge agent at a site."
      size="md"
      footer={
        <div className="flex justify-end gap-3">
          <Button variant="secondary" onClick={handleClose}>
            Cancel
          </Button>
          <Button
            variant="primary"
            onClick={handleSubmit}
            loading={issueToken.isPending}
            disabled={sites.length === 0}
          >
            Issue Token
          </Button>
        </div>
      }
    >
      <div className="space-y-4">
        {sites.length === 0 ? (
          <div className="rounded-lg border border-yellow-200 bg-yellow-50 p-4 dark:border-yellow-800 dark:bg-yellow-900/20">
            <div className="flex gap-3">
              <AlertCircle className="h-5 w-5 text-yellow-600 dark:text-yellow-400" />
              <div>
                <p className="text-sm font-medium text-yellow-800 dark:text-yellow-300">
                  No sites available
                </p>
                <p className="mt-1 text-sm text-yellow-700 dark:text-yellow-400">
                  Sites are automatically created when agents enroll. You can issue a token
                  for manual site creation or wait for an agent to connect.
                </p>
              </div>
            </div>
          </div>
        ) : (
          <>
            <Select
              label="Select Site"
              placeholder="Choose a site..."
              options={siteOptions}
              value={selectedSiteId}
              onChange={(e) => {
                setSelectedSiteId(e.target.value);
                setError(null);
              }}
              required
            />

            <Input
              label="Token Expiration (minutes)"
              type="number"
              min={1}
              max={1440}
              value={ttlMinutes}
              onChange={(e) => {
                setTtlMinutes(e.target.value);
                setError(null);
              }}
              helperText="How long the token will be valid (1-1440 minutes, default: 15)"
              required
            />

            {error && (
              <div className="flex items-center gap-2 text-sm text-red-600">
                <AlertCircle className="h-4 w-4" />
                <span>{error}</span>
              </div>
            )}

            <div className="rounded-lg border border-blue-200 bg-blue-50 p-4 dark:border-blue-800 dark:bg-blue-900/20">
              <div className="flex gap-3">
                <Badge variant="info" size="sm" className="mt-0.5">
                  Info
                </Badge>
                <div>
                  <p className="text-sm font-medium text-blue-800 dark:text-blue-300">
                    One-time use
                  </p>
                  <p className="mt-1 text-sm text-blue-700 dark:text-blue-400">
                    The token can only be used once to enroll a single agent. After use,
                    it will be marked as consumed and cannot be reused.
                  </p>
                </div>
              </div>
            </div>
          </>
        )}
      </div>
    </Modal>
  );
}
