# VPS Tools - Security Best Practices

This document outlines security recommendations for VPS Tools deployments.

## SSH Security

### Recommended Configuration

```bash
# /etc/ssh/sshd_config
Port 22222              # Non-standard port
PermitRootLogin prohibit-password
PasswordAuthentication no
PubkeyAuthentication yes
MaxAuthTries 3
MaxSessions 5
```

### Key Types (Strongest to Weakest)

| Type | Bits | Recommendation |
|------|------|----------------|
| Ed25519 | 256 | ✅ Preferred |
| ECDSA | 256+ | ✅ Good |
| RSA | 4096 | ⚠️ Acceptable |
| RSA | 2048 | ⚠️ Minimum |
| DSS/DSA | * | ❌ Deprecated |

### Regular Audits

```bash
# Weekly SSH key audit
vps-tools ssh-audit

# Check for weak keys
sudo bash security/vps-ssh-audit.sh --audit
```

## Firewall Configuration

### Essential Rules

```bash
# Allow SSH (your port)
sudo ufw allow 22222/tcp

# Allow HTTP/HTTPS
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Deny everything else
sudo ufw default deny incoming
sudo ufw enable
```

### Port Monitoring

```bash
# Weekly port audit
vps-tools ports --expected-ports=22222,80,443

# Alert on unexpected ports
vps-tools ports --expected-ports=22222,80,443 --alert-email=admin@example.com
```

## Failed Login Protection

### Monitoring

```bash
# Daily login analysis
vps-tools logins --days=1

# Block suspicious IPs
vps-tools logins --block-ips --threshold=5
```

### Integration with fail2ban

VPS Tools complements fail2ban. For comprehensive protection:

```bash
sudo apt install fail2ban
sudo systemctl enable fail2ban
```

## SSL/TLS Certificates

### Certificate Monitoring

```bash
# Weekly SSL check
vps-tools ssl --warn-days=30

# Alert before expiration
vps-tools ssl --warn-days=60 --alert-email=admin@example.com
```

### Auto-Renewal

For Let's Encrypt:

```bash
# Test renewal
sudo certbot renew --dry-run

# Automatic via cron (usually auto-configured)
0 12 * * * /usr/bin/certbot renew --quiet
```

## System Hardening

VPS Tools applies these sysctl settings:

```bash
# Disable IP forwarding
net.ipv4.ip_forward = 0
net.ipv6.conf.all.forwarding = 0

# Disable ICMP redirects
net.ipv4.conf.all.send_redirects = 0
net.ipv4.conf.all.accept_redirects = 0

# Enable SYN flood protection
net.ipv4.tcp_syncookies = 1

# Log martian packets
net.ipv4.conf.all.log_martians = 1
```

## Backup Security

### Encryption

Consider encrypting sensitive backups:

```bash
# Encrypt backup
tar -czf - /opt/backups | gpg -c > backup.tar.gz.gpg

# Decrypt
gpg -d backup.tar.gz.gpg | tar -xzf -
```

### Off-site Storage

Store backups on separate systems:

```bash
# Sync to remote server
rsync -avz /opt/backups/ user@backup-server:/backups/

# Or use cloud storage
aws s3 sync /opt/backups s3://my-bucket/backups/
```

## Docker Security

### Running Rootless

Consider rootless Docker for enhanced security:

```bash
# Install rootless Docker
dockerd-rootless-setuptool.sh install
```

### Image Scanning

Scan images for vulnerabilities:

```bash
# Using Trivy
docker run aquasec/trivy image your-image:tag
```

## Monitoring and Alerting

### Email Alerts

Configure email for all critical scripts:

```bash
# /etc/cron.d/vps-tools
MAILTO=admin@example.com
```

### Log Monitoring

```bash
# Watch logs in real-time
sudo tail -f /var/log/vps-tools/*.log

# Check for errors
grep -r "ERROR\|CRITICAL" /var/log/vps-tools/
```

## Incident Response

### If Compromised

1. **Isolate**: Block all traffic except your IP
   ```bash
   sudo ufw default deny incoming
   sudo ufw allow from YOUR_IP
   ```

2. **Review**: Check logs for unauthorized access
   ```bash
   sudo bash monitoring/vps-log-analyzer.sh --days=30 --type=auth
   ```

3. **Rotate**: Change all SSH keys
   ```bash
   sudo bash security/vps-ssh-audit.sh --rotate-user=ubuntu
   ```

4. **Restore**: From known good backup
   ```bash
   sudo bash docker/vps-docker-backup-restore.sh --mode=restore --backup-file=/path/to/backup.tar.gz
   ```

## Checklist

- [ ] SSH uses key-based authentication only
- [ ] Non-standard SSH port configured
- [ ] UFW enabled with minimal rules
- [ ] SSL certificates monitored
- [ ] Failed logins tracked and blocked
- [ ] Regular backups with off-site copies
- [ ] Email alerts configured
- [ ] System updates automated
- [ ] Cron jobs installed and running
