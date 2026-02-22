'use client';

import { useState, useEffect } from 'react';
import { NetworkNodeData } from '../types';
import { FormField, TextInput, SelectInput } from './forms';

interface NetworkPropertyFormProps {
  data: NetworkNodeData;
  onChange: (data: Partial<NetworkNodeData>) => void;
}

const NETWORK_TYPE_OPTIONS = [
  { value: 'bridge', label: 'Bridge' },
  { value: 'vxlan', label: 'VXLAN' },
];

const CIDR_REGEX = /^(\d{1,3}\.){3}\d{1,3}\/\d{1,2}$/;

function isValidCIDR(cidr: string): boolean {
  if (!CIDR_REGEX.test(cidr)) return false;
  const [ip, prefix] = cidr.split('/');
  const prefixNum = parseInt(prefix, 10);
  if (prefixNum < 0 || prefixNum > 32) return false;
  const octets = ip.split('.').map(Number);
  return octets.every((octet) => octet >= 0 && octet <= 255);
}

export function NetworkPropertyForm({ data, onChange }: NetworkPropertyFormProps) {
  const [cidrError, setCidrError] = useState<string>('');

  useEffect(() => {
    if (data.cidr && !isValidCIDR(data.cidr)) {
      setCidrError('Invalid CIDR format (e.g., 10.0.0.0/24)');
    } else {
      setCidrError('');
    }
  }, [data.cidr]);

  const handleCIDRChange = (value: string) => {
    onChange({ cidr: value });
  };

  return (
    <div className="space-y-4">
      <div className="border-b pb-3">
        <h3 className="text-sm font-semibold text-slate-900">Network Configuration</h3>
        <p className="text-xs text-slate-500 mt-0.5">Configure network settings</p>
      </div>

      <FormField label="Name">
        <TextInput
          value={data.name}
          onChange={(value) => onChange({ name: value })}
          placeholder="Enter network name"
        />
      </FormField>

      <FormField label="CIDR">
        <TextInput
          value={data.cidr || ''}
          onChange={handleCIDRChange}
          placeholder="e.g., 10.0.0.0/24"
          className={cidrError ? 'border-red-300 focus:border-red-500 focus:ring-red-500' : ''}
        />
        {cidrError && (
          <p className="text-xs text-red-500 mt-1">{cidrError}</p>
        )}
      </FormField>

      <FormField label="Bridge Name (Optional)">
        <TextInput
          value={(data.bridgeName as string | undefined) || ''}
          onChange={(value) => onChange({ bridgeName: value })}
          placeholder="e.g., br0"
        />
      </FormField>

      <FormField label="Type">
        <SelectInput
          value={data.networkType}
          onChange={(value) => onChange({ networkType: value as NetworkNodeData['networkType'] })}
          options={NETWORK_TYPE_OPTIONS}
        />
      </FormField>
    </div>
  );
}
