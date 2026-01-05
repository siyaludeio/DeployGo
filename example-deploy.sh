#!/bin/bash
# Example deployment script for zero downtime deployment
# This script will be executed from the projectPath directory

set -e

echo "Starting deployment process..."

# Example: Git pull
# git pull origin main

# Example: Install dependencies
# composer install --no-dev --optimize-autoloader
# npm install --production

# Example: Build assets
# npm run build

# Example: Run migrations
# php artisan migrate --force

# Example: Clear caches
# php artisan config:cache
# php artisan route:cache
# php artisan view:cache

# Example: Zero downtime - Reload PHP-FPM
# sudo systemctl reload php8.1-fpm

# Example: Zero downtime - Reload Nginx
# sudo systemctl reload nginx

echo "Deployment completed successfully!"

