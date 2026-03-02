# Health Check & Status Endpoints - Documentation

## Overview

Health check endpoints provide real-time monitoring of your API's health and system status. Essential for load balancers, monitoring tools, and Kubernetes deployments.

---

## Endpoints

### 1. **GET `/health`** - Quick Health Check

**Purpose:** Simple health check for load balancers  
**Authentication:** None (public)  
**Response Time:** <50ms

**Usage:**
```bash
curl http://api.yourdomain.com/health
```

**Response (Healthy):**
```json
{
    "status": true,
    "message": "Service is healthy",
    "data": {
        "status": "UP",
        "timestamp": "2026-01-24 12:00:00"
    }
}
```
**HTTP Code:** 200

**Response (Unhealthy):**
```json
{
    "status": false,
    "message": "Service is unhealthy",
    "data": {
        "status": "DOWN",
        "timestamp": "2026-01-24 12:00:00"
    }
}
```
**HTTP Code:** 503

---

### 2. **GET `/health/detailed`** - Detailed Health Check

**Purpose:** Comprehensive health information  
**Authentication:** Optional  
**Checks:** Database, cache, disk space, uploads directory

**Usage:**
```bash
curl http://api.yourdomain.com/health/detailed
```

**Response:**
```json
{
    "status": true,
    "message": "All systems operational",
    "data": {
        "healthy": true,
        "timestamp": "2026-01-24 12:00:00",
        "checks": {
            "database": {
                "healthy": true,
                "message": "Database connection successful",
                "response_time_ms": 2.5
            },
            "cache": {
                "healthy": true,
                "message": "Cache is working",
                "type": "file-based"
            },
            "disk_space": {
                "healthy": true,
                "message": "Sufficient disk space",
                "free_mb": 45000,
                "total_mb": 50000,
                "used_percent": 10
            },
            "uploads_directory": {
                "healthy": true,
                "message": "Uploads directory accessible",
                "writable": true
            }
        }
    }
}
```

---

### 3. **GET `/health/status`** - System Status

**Purpose:** Detailed system information for debugging  
**Authentication:** Optional (recommended for admin only)

**Usage:**
```bash
curl http://api.yourdomain.com/health/status
```

**Response:**
```json
{
    "status": true,
    "message": "System status retrieved",
    "data": {
        "api": {
            "name": "Divine API",
            "version": "1.0.0",
            "environment": "production"
        },
        "server": {
            "php_version": "8.1.0",
            "server_software": "Apache/2.4.54",
            "server_name": "api.yourdomain.com",
            "server_time": "2026-01-24 12:00:00",
            "timezone": "UTC"
        },
        "resources": {
            "memory": {
                "current_mb": 25.5,
                "peak_mb": 32.1,
                "limit": "256M"
            },
            "cpu": {
                "load_average": {
                    "1min": 0.5,
                    "5min": 0.7,
                    "15min": 0.6
                }
            }
        },
        "uptime": "15d 8h 32m",
        "health": { ... }
    }
}
```

---

### 4. **GET `/health/ping`** - Simple Ping

**Purpose:** Lightweight endpoint to confirm server responds  
**Response Time:** <10ms

**Usage:**
```bash
curl http://api.yourdomain.com/health/ping
```

**Response:**
```json
{
    "status": true,
    "message": "pong",
    "data": {
        "timestamp": 1706150400.123
    }
}
```

---

### 5. **GET `/health/ready`** - Readiness Check

**Purpose:** Kubernetes readiness probe  
**Checks:** All critical services

**Usage:**
```bash
curl http://api.yourdomain.com/health/ready
```

**Response (Ready):**
```json
{
    "status": true,
    "message": "Service is ready",
    "data": {
        "ready": true
    }
}
```
**HTTP Code:** 200

---

### 6. **GET `/health/live`** - Liveness Check

**Purpose:** Kubernetes liveness probe  
**Checks:** Process is alive

**Usage:**
```bash
curl http://api.yourdomain.com/health/live
```

**Response:**
```json
{
    "status": true,
    "message": "Service is alive",
    "data": {
        "alive": true,
        "timestamp": "2026-01-24 12:00:00"
    }
}
```

---

## Load Balancer Integration

### NGINX Configuration

```nginx
upstream divine_api {
    server backend1.example.com:80;
    server backend2.example.com:80;
    
    # Health check configuration
    check interval=3000 rise=2 fall=3 timeout=1000 type=http;
    check_http_send "GET /health HTTP/1.0\r\n\r\n";
    check_http_expect_alive http_2xx;
}
```

### HAProxy Configuration

```haproxy
backend divine_api
    option httpchk GET /health
    http-check expect status 200
    server backend1 192.168.1.10:80 check
    server backend2 192.168.1.11:80 check
```

---

## Kubernetes/Docker Configuration

### Kubernetes Deployment

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: divine-api
spec:
  containers:
  - name: api
    image: divine-api:latest
    ports:
    - containerPort: 80
    
    # Liveness probe
    livenessProbe:
      httpGet:
        path: /health/live
        port: 80
      initialDelaySeconds: 30
      periodSeconds: 10
    
    # Readiness probe
    readinessProbe:
      httpGet:
        path: /health/ready
        port: 80
      initialDelaySeconds: 5
      periodSeconds: 5
```

---

## Monitoring Integration

### Prometheus/Grafana

Use `/health/detailed` to expose metrics:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'divine-api'
    metrics_path: '/health/detailed'
    static_configs:
      - targets: ['api.yourdomain.com:80']
```

### Uptime Monitoring (UptimeRobot, Pingdom)

**Endpoint:** `/health`  
**Expected Response:** 200  
**Check Interval:** 1 minute

---

## Use Cases

### 1. Basic Health Monitoring

```bash
#!/bin/bash
# Simple monitoring script

HEALTH_URL="http://api.yourdomain.com/health"

response=$(curl -s -o /dev/null -w "%{http_code}" $HEALTH_URL)

if [ $response -eq 200 ]; then
    echo "✅ API is healthy"
else
    echo "❌ API is down - HTTP $response"
    # Send alert
fi
```

### 2. Detailed Diagnostics

```bash
# Get full system status
curl http://api.yourdomain.com/health/status | jq '.'

# Check specific component
curl http://api.yourdomain.com/health/detailed | jq '.data.checks.database'
```

### 3. Pre-Deployment Check

```bash
# Before deploying new version, check if current is healthy
if curl -f http://staging.yourdomain.com/health; then
    echo "Staging is healthy, proceeding with deployment"
    # Deploy to production
else
    echo "Staging is unhealthy, aborting deployment"
    exit 1
fi
```

---

## Troubleshooting

### Service Returns 503

**Check:**
1. Database connection
2. Disk space (needs >1GB free)
3. Cache directory permissions
4. Uploads directory exists and writable

**Fix:**
```bash
# Check disk space
df -h

# Fix uploads directory
mkdir -p public/assets/uploads
chmod 755 public/assets/uploads

# Check database
mysql -u root -p -e "SELECT 1"
```

### Slow Response Times

If `/health` takes >100ms:
1. Check database performance
2. Review disk I/O
3. Check server load

---

## Security Considerations

### Public vs Private Endpoints

**Public (No Auth):**
- `/health` - Basic health check
- `/health/ping` - Ping endpoint
- `/health/live` - Liveness probe
- `/health/ready` - Readiness probe

**Private (Auth Recommended):**
- `/health/status` - Exposes system details
- `/health/detailed` - Shows internal state

### Rate Limiting

Health endpoints typically skip rate limiting for monitoring tools, but you can add it:

```php
// In Health controller
public function status()
{
    $this->applyRateLimit('health_status', 60, 3600);
    // ...
}
```

---

## Summary

**Health Check Endpoints:**
- ✅ `/health` - Quick check (load balancers)
- ✅ `/health/detailed` - Comprehensive checks
- ✅ `/health/status` - System information
- ✅ `/health/ping` - Lightweight ping
- ✅ `/health/ready` - Kubernetes ready
- ✅ `/health/live` - Kubernetes alive

**Use For:**
- Load balancer health checks
- Uptime monitoring
- Kubernetes probes
- System diagnostics
- Pre-deployment validation

**Essential for production deployments!** 🚀
