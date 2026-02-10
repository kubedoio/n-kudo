Name:           nkudo-edge
Version:        {{VERSION}}
Release:        1%{?dist}
Summary:        n-kudo edge agent
License:        Apache-2.0
URL:            https://github.com/kubedoio/n-kudo
Source0:        nkudo-edge-%{version}.tar.gz

Requires:       systemd
Requires(pre):  shadow-utils
Requires(post): systemd
Requires(preun): systemd
Requires(postun): systemd

%description
MicroVM orchestration edge agent for n-kudo control plane.
The edge agent runs on edge nodes and communicates with the
n-kudo control plane to orchestrate MicroVM workloads.

%prep
%setup -q

%install
rm -rf %{buildroot}

# Install binary
install -D -m 0755 nkudo-edge %{buildroot}/usr/local/bin/nkudo-edge

# Install systemd service file
install -D -m 0644 nkudo-edge.service %{buildroot}/etc/systemd/system/nkudo-edge.service

# Install default environment file
install -D -m 0640 nkudo-edge.env %{buildroot}/etc/nkudo-edge/nkudo-edge.env

# Create directories
mkdir -p %{buildroot}/var/lib/nkudo-edge/{state,pki,vms}
mkdir -p %{buildroot}/etc/nkudo-edge

%pre
# Create group if it doesn't exist
getent group nkudo-edge >/dev/null || groupadd --system nkudo-edge

# Create user if it doesn't exist
getent passwd nkudo-edge >/dev/null || \
    useradd --system --gid nkudo-edge --home-dir /var/lib/nkudo-edge \
    --shell /sbin/nologin --comment "n-kudo edge agent" nkudo-edge

%post
# Reload systemd daemon
%systemd_post nkudo-edge.service

# Set permissions
chown -R root:root /usr/local/bin/nkudo-edge
chown -R nkudo-edge:nkudo-edge /var/lib/nkudo-edge
chmod 750 /var/lib/nkudo-edge

# Enable and start service
%systemd_enable nkudo-edge.service
if [ "$1" -eq 1 ]; then
    # First install
    systemctl start nkudo-edge.service || :
fi

%preun
# Stop and disable service before uninstall
%systemd_preun nkudo-edge.service

%postun
# Reload systemd daemon
%systemd_postun_with_restart nkudo-edge.service

# On complete removal, clean up
if [ "$1" -eq 0 ]; then
    # Remove user and group
    userdel nkudo-edge 2>/dev/null || :
    groupdel nkudo-edge 2>/dev/null || :
fi

%files
%license LICENSE
%doc README.md
/usr/local/bin/nkudo-edge
/etc/systemd/system/nkudo-edge.service
%config(noreplace) %attr(640, root, root) /etc/nkudo-edge/nkudo-edge.env
%attr(750, nkudo-edge, nkudo-edge) /var/lib/nkudo-edge
%attr(750, nkudo-edge, nkudo-edge) /var/lib/nkudo-edge/state
%attr(750, nkudo-edge, nkudo-edge) /var/lib/nkudo-edge/pki
%attr(750, nkudo-edge, nkudo-edge) /var/lib/nkudo-edge/vms

%changelog
* Thu Jan 01 2025 n-kudo maintainers <maintainers@nkudo.io> - 0.1.0-1
- Initial package release
