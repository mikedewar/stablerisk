# Nginx Configuration for StableRisk

This directory contains Nginx configurations for both development and production environments with TLS support.

## Files

- **nginx.conf** - Development configuration (HTTP only)
- **nginx-production.conf** - Production configuration with TLS 1.3
- **docker-compose.nginx.yml** - Docker Compose setup with Certbot for automatic TLS certificate management

## Development Setup

For local development, use the basic `nginx.conf`:

```bash
docker run -d \
  --name nginx \
  --network stablerisk \
  -p 80:80 \
  -v $(pwd)/nginx.conf:/etc/nginx/nginx.conf:ro \
  nginx:1.25-alpine
```

## Production Setup

### Prerequisites

1. A registered domain name pointing to your server
2. Ports 80 and 443 open in your firewall
3. Docker and Docker Compose installed

### Step 1: Generate Diffie-Hellman Parameters

This is required for strong TLS security (takes 5-10 minutes):

```bash
cd deployments/nginx
openssl dhparam -out dhparam.pem 4096
```

### Step 2: Update Domain Configuration

Edit `nginx-production.conf` and replace `stablerisk.yourdomain.com` with your actual domain:

```bash
sed -i 's/stablerisk.yourdomain.com/your-actual-domain.com/g' nginx-production.conf
```

### Step 3: Obtain Let's Encrypt Certificate

First, create the certbot network and obtain the initial certificate:

```bash
# Create network
docker network create stablerisk

# Obtain certificate (replace with your domain and email)
docker run -it --rm \
  --name certbot \
  -v "$(pwd)/certbot-conf:/etc/letsencrypt" \
  -v "$(pwd)/certbot-www:/var/www/certbot" \
  certbot/certbot certonly \
  --webroot \
  --webroot-path=/var/www/certbot \
  --email admin@yourdomain.com \
  --agree-tos \
  --no-eff-email \
  -d your-actual-domain.com
```

**Note**: For initial certificate generation, you'll need to temporarily run Nginx without TLS:

```bash
# Temporarily comment out the HTTPS server block in nginx-production.conf
# Then start Nginx
docker-compose -f docker-compose.nginx.yml up -d nginx

# Obtain certificate
docker run -it --rm \
  --name certbot \
  --network stablerisk \
  -v "$(pwd)/certbot-conf:/etc/letsencrypt" \
  -v "$(pwd)/certbot-www:/var/www/certbot" \
  certbot/certbot certonly \
  --webroot \
  --webroot-path=/var/www/certbot \
  --email admin@yourdomain.com \
  --agree-tos \
  --no-eff-email \
  -d your-actual-domain.com

# Stop nginx, uncomment HTTPS block, restart
docker-compose -f docker-compose.nginx.yml down
# Uncomment HTTPS server block
docker-compose -f docker-compose.nginx.yml up -d
```

### Step 4: Start Nginx with Certbot

```bash
docker-compose -f docker-compose.nginx.yml up -d
```

This will:
- Start Nginx with production TLS configuration
- Start Certbot for automatic certificate renewal (every 12 hours)
- Expose ports 80 (HTTP redirect) and 443 (HTTPS)

### Step 5: Verify TLS Configuration

Test your TLS setup:

```bash
# Check certificate
openssl s_client -connect your-domain.com:443 -servername your-domain.com < /dev/null

# Test SSL Labs rating (aim for A+)
# Visit: https://www.ssllabs.com/ssltest/analyze.html?d=your-domain.com

# Verify HSTS header
curl -I https://your-domain.com
```

## Certificate Renewal

Certbot automatically renews certificates every 12 hours. Manual renewal:

```bash
docker exec stablerisk-certbot certbot renew --webroot -w /var/www/certbot

# Reload Nginx to use new certificate
docker exec stablerisk-nginx nginx -s reload
```

## Security Features

The production configuration includes:

### TLS Configuration (PCI-DSS Compliant)
- TLS 1.3 and TLS 1.2 only
- Strong cipher suites (ECDHE, AES-GCM, ChaCha20-Poly1305)
- Perfect Forward Secrecy (PFS)
- OCSP stapling
- 4096-bit DH parameters

### Security Headers (ISO27001 Compliant)
- **HSTS**: Forces HTTPS for 1 year
- **X-Frame-Options**: Prevents clickjacking
- **X-Content-Type-Options**: Prevents MIME sniffing
- **CSP**: Content Security Policy
- **Permissions-Policy**: Restricts browser features

### Rate Limiting
- API endpoints: 100 requests/second with burst of 20
- Login endpoint: 5 requests/minute with burst of 2 (brute-force protection)
- Connection limit: 50 concurrent connections per IP

### Access Controls
- Metrics endpoint restricted to internal networks only
- Hidden files (.git, .env) blocked
- Sensitive file extensions (.sql, .log, .bak) blocked

## Monitoring

View Nginx logs:

```bash
# Access logs
docker logs stablerisk-nginx -f

# Error logs
docker exec stablerisk-nginx tail -f /var/log/nginx/error.log

# Check Nginx status
docker exec stablerisk-nginx nginx -t
```

## Kubernetes Deployment

For Kubernetes, TLS is handled by the Ingress controller with cert-manager. See `../kubernetes/ingress.yaml` for configuration.

## Troubleshooting

### Certificate Issues

```bash
# Check certificate validity
docker exec stablerisk-certbot certbot certificates

# Force renewal
docker exec stablerisk-certbot certbot renew --force-renewal

# Check certificate files
docker exec stablerisk-certbot ls -la /etc/letsencrypt/live/
```

### Nginx Configuration Test

```bash
# Test configuration
docker exec stablerisk-nginx nginx -t

# Reload configuration
docker exec stablerisk-nginx nginx -s reload
```

### Permission Issues

If Nginx can't read certificates:

```bash
# Check certificate permissions
docker exec stablerisk-certbot ls -la /etc/letsencrypt/live/your-domain.com/

# Certbot should create certificates with correct permissions automatically
# If issues persist, check SELinux or AppArmor policies
```

## Compliance Notes

### ISO27001
- A.9 Access Control: Metrics endpoint restricted to internal networks
- A.10 Cryptography: TLS 1.3 with strong ciphers
- A.12 Operations Security: Security headers, rate limiting

### PCI-DSS
- Requirement 2.3: Encryption of non-console admin access (TLS 1.3)
- Requirement 4.1: Strong cryptography for transmission over open networks
- Requirement 6.5.10: Broken authentication protection (rate limiting)
- Requirement 8.2.1: Strong authentication (supports JWT via API backend)

## References

- [Mozilla SSL Configuration Generator](https://ssl-config.mozilla.org/)
- [OWASP Secure Headers Project](https://owasp.org/www-project-secure-headers/)
- [Let's Encrypt Documentation](https://letsencrypt.org/docs/)
