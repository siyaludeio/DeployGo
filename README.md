# DeployGo 🚀

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.21-cyan.svg)
![Platform](https://img.shields.io/badge/platform-linux%20%7C%20macos-lightgrey.svg)

**DeployGo** is a lightweight, high-performance command-line tool written in Go designed to bridge the gap between web applications (like Laravel/PHP) and system-level deployment tasks.

It solves the common "timeout" problem when triggering long-running deployment scripts from a web request or queue worker by spawning a completely detached background process.



## ✨ Key Features

- **🔥 Fire & Forget**: Triggers deployment and immediately releases the calling process.
- **🛡️ Secure Execution**: Strict absolute path validation to prevent traversal attacks.
- **📝 Robust Logging**: Real-time logging with timestamps and auto-rotation (`deployment_TIMESTAMP.log`).
- **⚡ Performance**: Zero overhead on your web server threads.
- **🐧 Cross-Platform**: Native support for Linux (x64/ARM64) and macOS (Intel/Apple Silicon).

## 🏗️ Architecture

When you run `deploygo deploy ...`:

1.  **Validation**: It verifies all paths check out.
2.  **Spawn**: It launches a self-managed background process.
3.  **Detach**: The CLI exits immediately (Exit Code 0), letting your PHP/Web process finish instantly.
4.  **Execute**: The background process runs your script, streams logs to disk, and cleans up after itself.

## 📦 Installation

### Option 1: Build from Source (Recommended)

```bash
# Clone the repository
git clone https://github.com/yourusername/deploy-go.git
cd deploy-go

# Build using the interactive script
chmod +x build.sh
./build.sh
```

### Option 2: Quick Compile

```bash
go build -o deploygo main.go deployment.go
```

### Global Installation

Move the binary to your path to use it anywhere:

```bash
sudo mv deploygo /usr/local/bin/
sudo chmod +x /usr/local/bin/deploygo
```

## 🚀 Usage

### Basic Command

```bash
deploygo deploy \
  --project="/var/www/my-app" \
  --deployScript="/var/www/my-app/deploy.sh" \
  --logPath="/var/www/my-app/logs"
```

### Running as Web User (Recommended)

To ensure files created during deployment (caches, views) are owned by the correct user, run as `www-data`:

```bash
sudo -u www-data deploygo deploy \
  --project="/var/www/my-app" \
  --deployScript="/var/www/my-app/deploy.sh" \
  --logPath="/var/www/my-app/logs"
```

## 🛠️ Integration Guide

### Laravel / PHP Integration

Stop hitting `max_execution_time` limits. Trigger deployments from your Artisan commands or Controllers.

#### 1. Create the Command

```php
// app/Console/Commands/TriggerDeployment.php

namespace App\Console\Commands;

use Illuminate\Console\Command;
use Symfony\Component\Process\Process;

class TriggerDeployment extends Command
{
    protected $signature = 'app:deploy';
    protected $description = 'Trigger a background deployment';

    public function handle()
    {
        $binary = '/usr/local/bin/deploygo';
        
        $command = [
            $binary,
            'deploy',
            '--project=' . base_path(),
            '--deployScript=' . base_path('deploy.sh'),
            '--logPath=' . storage_path('logs'),
        ];

        // This runs instantly!
        $process = new Process($command);
        $process->run();

        if (!$process->isSuccessful()) {
            $this->error('Failed to start: ' . $process->getErrorOutput());
            return 1;
        }

        $this->info('Deployment started in background! 🚀');
    }
}
```

### Logs & Monitoring

Logs are automatically rotated. You can easily build a live log viewer in your dashboard by polling the active log file:

`storage/logs/deployment.log` (Active)
`storage/logs/deployment_20240101_120000.log` (Rotated History)

## 🔒 Security

- **Path Restriction**: The tool refuses to run if paths are not absolute.
- **Permissions**: It inherits the permissions of the user running it. Always enforce least-privilege by running as `www-data` or a dedicated deployment user, never `root`.

## 🤝 Contributing

1.  Fork the Project
2.  Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3.  Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4.  Push to the Branch (`git push origin feature/AmazingFeature`)
5.  Open a Pull Request

## 🗺️ Roadmap

- [ ] Webhook support (Slack/Discord notifications on failure).
- [ ] Configurable log retention policies.
- [ ] Dry-run mode for script validation.

## 📬 Author

**Indunil Peramuna**

- 🐙 GitHub: [@iperamuna](https://github.com/iperamuna)
- 📧 Email: [indunil@siyalude.io](mailto:indunil@siyalude.io)
- 💬 WhatsApp: [+94 77 767 1771](https://wa.me/94777671771)

## 📄 License

Distributed under the MIT License. See `LICENSE` for more information.
