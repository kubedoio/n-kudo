BEGIN;

-- VXLAN networks
CREATE TABLE vxlan_networks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    site_id UUID REFERENCES sites(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    vni INTEGER NOT NULL UNIQUE,
    cidr CIDR NOT NULL,
    gateway INET,
    mtu INTEGER DEFAULT 1450,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, site_id, name)
);

-- VXLAN tunnels per host
CREATE TABLE vxlan_tunnels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    network_id UUID NOT NULL REFERENCES vxlan_networks(id) ON DELETE CASCADE,
    host_id UUID NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
    local_ip INET NOT NULL,
    vtep_name TEXT NOT NULL,
    status TEXT DEFAULT 'pending',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(network_id, host_id)
);

-- VM network attachments
CREATE TABLE vm_network_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vm_id UUID NOT NULL REFERENCES microvms(id) ON DELETE CASCADE,
    network_id UUID NOT NULL REFERENCES vxlan_networks(id) ON DELETE CASCADE,
    ip_address INET,
    mac_address MACADDR,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(vm_id, network_id)
);

-- Indexes for efficient lookups
CREATE INDEX idx_vxlan_networks_tenant_site 
    ON vxlan_networks(tenant_id, site_id);

CREATE INDEX idx_vxlan_networks_vni 
    ON vxlan_networks(vni);

CREATE INDEX idx_vxlan_tunnels_network 
    ON vxlan_tunnels(network_id);

CREATE INDEX idx_vxlan_tunnels_host 
    ON vxlan_tunnels(host_id);

CREATE INDEX idx_vxlan_tunnels_status 
    ON vxlan_tunnels(status);

CREATE INDEX idx_vm_network_attachments_vm 
    ON vm_network_attachments(vm_id);

CREATE INDEX idx_vm_network_attachments_network 
    ON vm_network_attachments(network_id);

COMMIT;
