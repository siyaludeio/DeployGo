# DeployGo v1.0.0 Release Notes

We are excited to announce the first major release of **DeployGo**! 🚀

This release focuses on stability, security, and ease of use for PHP/Laravel developers needing to trigger system-level deployments without timeouts.

## 🌟 Highlights

*   **CLI-First Design**: No background daemons or services required. Just run and done.
*   **Zero-Overhead**: The CLI exits immediately, leaving the heavy lifting to a background process.
*   **Safe Logging**: Logs are written to `deployment.log` and automatically rotated to `deployment_TIMESTAMP.log` to preserve history.

## 📦 Installation

```bash
git clone https://github.com/iperamuna/deploygo.git
cd deploygo
./build.sh
```

## 🛠 Usage

```bash
deploygo deploy --project=/var/www/app --deployScript=/var/www/app/deploy.sh --logPath=/var/www/app/logs
```
