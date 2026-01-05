# Self Deployer - Go Deployment Service

A Linux service written in Go that handles deployment requests from PHP applications with zero downtime support.

## Features

- ✅ HTTP API endpoint for PHP integration
- ✅ Path validation and permission checking
- ✅ Asynchronous deployment processing
- ✅ Real-time logging with timestamps
- ✅ Non-blocking log files (readable by PHP with polling)
- ✅ Zero downtime deployment support
- ✅ Systemd service integration

## Installation

### 1. Build the Application

```bash
go mod download
go build -o self-deployer
```

### 2. Install as System Service

```bash
# Copy binary to /opt/deployer
sudo mkdir -p /opt/deployer
sudo cp self-deployer /opt/deployer/
sudo chmod +x /opt/deployer/self-deployer

# Copy service file
sudo cp deployer.service /etc/systemd/system/

# Create necessary directories
sudo mkdir -p /tmp/deployer/queue
sudo chown www-data:www-data /tmp/deployer/queue

# Reload systemd and start service
sudo systemctl daemon-reload
sudo systemctl enable deployer.service
sudo systemctl start deployer.service

# Check status
sudo systemctl status deployer.service
```

### 3. Configure User/Group and Port

Edit `/etc/systemd/system/deployer.service` and:
- Change `User` and `Group` to match your web server user (commonly `www-data`, `nginx`, or `apache`)
- Change `PORT` environment variable if you want to use a custom port (default: 8080)

## Usage

### From PHP

```php
<?php
$data = [
    'projectPath' => '/var/www/myproject',
    'deploymentScriptPath' => '/var/www/myproject/deploy.sh',
    'logPath' => '/var/www/myproject/deployment.log'
];

$ch = curl_init('http://localhost:8080/deploy');
curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
curl_setopt($ch, CURLOPT_POST, true);
curl_setopt($ch, CURLOPT_POSTFIELDS, json_encode($data));
curl_setopt($ch, CURLOPT_HTTPHEADER, ['Content-Type: application/json']);

$response = curl_exec($ch);
$httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
curl_close($ch);

if ($httpCode === 200) {
    $result = json_decode($response, true);
    echo "Deployment queued: " . $result['taskId'];
} else {
    echo "Error: " . $response;
}
?>
```

### Example Deployment Script

Create a deployment script (e.g., `deploy.sh`) in your project:

```bash
#!/bin/bash
set -e

echo "Starting deployment..."

# Pull latest code
git pull origin main

# Install dependencies
composer install --no-dev --optimize-autoloader
npm install --production

# Build assets
npm run build

# Run migrations
php artisan migrate --force

# Clear caches
php artisan config:cache
php artisan route:cache
php artisan view:cache

# Reload PHP-FPM for zero downtime
sudo systemctl reload php8.1-fpm

echo "Deployment completed successfully!"
```

Make sure the script is executable:
```bash
chmod +x deploy.sh
```

### Reading Logs from PHP (Livewire with Polling)

```php
<?php
// In your Livewire component
public function getLogs()
{
    $logPath = '/var/www/myproject/deployment.log';
    
    if (file_exists($logPath)) {
        // Read last 100 lines
        $lines = file($logPath);
        $recentLines = array_slice($lines, -100);
        return implode('', $recentLines);
    }
    
    return 'No logs available';
}
```

In your Livewire view:
```blade
<div wire:poll.5s="getLogs">
    <pre>{{ $this->getLogs() }}</pre>
</div>
```

## API Endpoints

### POST /deploy

Deploy a project.

**Request Body:**
```json
{
    "projectPath": "/var/www/myproject",
    "deploymentScriptPath": "/var/www/myproject/deploy.sh",
    "logPath": "/var/www/myproject/deployment.log"
}
```

**Response:**
```json
{
    "status": "queued",
    "taskId": "1234567890123456789",
    "message": "Deployment queued successfully"
}
```

### GET /health

Health check endpoint.

**Response:**
```
OK
```

## Zero Downtime Deployment

The service supports zero downtime deployments through:

1. **Symlink Switching**: Create a new deployment directory and switch symlinks
2. **Service Reload**: Reload services (PHP-FPM, Nginx) without full restart
3. **Health Checks**: Verify new deployment before switching

### Example Zero Downtime Script

```bash
#!/bin/bash
set -e

PROJECT_PATH="/var/www/myproject"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
NEW_DEPLOY="$PROJECT_PATH/.deployments/$TIMESTAMP"

# Create new deployment directory
mkdir -p "$NEW_DEPLOY"

# Clone/copy project to new directory
cp -r "$PROJECT_PATH/current"/* "$NEW_DEPLOY/"

# Install dependencies in new directory
cd "$NEW_DEPLOY"
composer install --no-dev --optimize-autoloader

# Run migrations
php artisan migrate --force

# Health check (customize as needed)
curl -f http://localhost/health || exit 1

# Switch symlink atomically
ln -sfn "$NEW_DEPLOY" "$PROJECT_PATH/current"

# Reload services
sudo systemctl reload php8.1-fpm
sudo systemctl reload nginx

echo "Zero downtime deployment completed!"
```

## Configuration

### Setting Custom Port

The service runs on port **8080** by default, but you can configure a custom port using the `PORT` environment variable.

#### Method 1: Systemd Service File (Recommended)

Edit `/etc/systemd/system/deployer.service` and change the `PORT` environment variable:

```ini
# Environment variables
Environment="PORT=9000"
```

Then reload and restart:
```bash
sudo systemctl daemon-reload
sudo systemctl restart deployer.service
```

#### Method 2: Command Line

Run directly with custom port:
```bash
PORT=9000 ./self-deployer
```

#### Method 3: Export Environment Variable

```bash
export PORT=9000
./self-deployer
```

#### Method 4: Using .env file (if using systemd)

Create `/opt/deployer/.env` and update the service file to load it:
```ini
EnvironmentFile=/opt/deployer/.env
```

### Environment Variables

- `PORT`: HTTP server port (default: 8080)

### Directory Structure

```
/tmp/deployer/
  └── queue/          # Temporary task files
```

## Logging

Logs are written with timestamps in the format:
```
[2024-01-15 10:30:45] [STDOUT] Your deployment output here
[2024-01-15 10:30:46] [STDERR] Any errors here
```

Log files are opened in append mode and flushed after each write, ensuring they're readable by PHP polling without file locking issues.

## Troubleshooting

### Service won't start

```bash
# Check logs
sudo journalctl -u deployer.service -f

# Check permissions
ls -la /opt/deployer/
ls -la /tmp/deployer/queue/
```

### Deployment not processing

```bash
# Check if file watcher is working
ls -la /tmp/deployer/queue/

# Check service logs
sudo journalctl -u deployer.service -n 50
```

### Permission issues

```bash
# Ensure web server user can write to log directory
sudo chown -R www-data:www-data /var/www/myproject
sudo chmod 755 /var/www/myproject
```

## Security Considerations

- The service runs as a system user (www-data) with limited privileges
- Path validation ensures only absolute paths are accepted
- Log directory must be writable before deployment is queued
- Consider adding authentication/authorization for production use

## License

MIT

