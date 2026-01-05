#!/bin/bash
set -e

echo "Starting deployment..."

# Pull changes
git pull origin main

# PHP/Composer
composer install --no-dev --optimize-autoloader

# Laravel Specifics
php artisan migrate --force
php artisan config:cache
php artisan route:cache
php artisan view:cache

# Optional: Restart Worker
# php artisan queue:restart

echo "Deployment finished successfully!"
