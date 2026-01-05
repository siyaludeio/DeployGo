<?php
/**
 * Example PHP code to call the Self Deployer service
 * 
 * This can be used in Laravel, Livewire, or plain PHP
 */

class DeployerClient
{
    private $baseUrl;
    
    public function __construct($baseUrl = 'http://localhost:8080')
    {
        // Change the port here if you're using a custom port
        // Example: 'http://localhost:9000'
        $this->baseUrl = $baseUrl;
    }
    
    /**
     * Queue a deployment
     * 
     * @param string $projectPath Absolute path to project directory
     * @param string $deploymentScriptPath Absolute path to deployment script
     * @param string $logPath Absolute path to log file
     * @return array Response with status and taskId
     * @throws Exception
     */
    public function deploy($projectPath, $deploymentScriptPath, $logPath)
    {
        $data = [
            'projectPath' => $projectPath,
            'deploymentScriptPath' => $deploymentScriptPath,
            'logPath' => $logPath
        ];
        
        $ch = curl_init($this->baseUrl . '/deploy');
        curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
        curl_setopt($ch, CURLOPT_POST, true);
        curl_setopt($ch, CURLOPT_POSTFIELDS, json_encode($data));
        curl_setopt($ch, CURLOPT_HTTPHEADER, [
            'Content-Type: application/json'
        ]);
        curl_setopt($ch, CURLOPT_TIMEOUT, 10);
        
        $response = curl_exec($ch);
        $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
        $error = curl_error($ch);
        curl_close($ch);
        
        if ($error) {
            throw new Exception("CURL Error: " . $error);
        }
        
        if ($httpCode !== 200) {
            throw new Exception("HTTP Error {$httpCode}: " . $response);
        }
        
        $result = json_decode($response, true);
        if (json_last_error() !== JSON_ERROR_NONE) {
            throw new Exception("Invalid JSON response: " . $response);
        }
        
        return $result;
    }
    
    /**
     * Check service health
     * 
     * @return bool
     */
    public function healthCheck()
    {
        $ch = curl_init($this->baseUrl . '/health');
        curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
        curl_setopt($ch, CURLOPT_TIMEOUT, 5);
        
        $response = curl_exec($ch);
        $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
        curl_close($ch);
        
        return $httpCode === 200 && $response === 'OK';
    }
    
    /**
     * Read deployment logs (for Livewire polling)
     * 
     * @param string $logPath Path to log file
     * @param int $lines Number of recent lines to return
     * @return string
     */
    public function readLogs($logPath, $lines = 100)
    {
        if (!file_exists($logPath)) {
            return 'No logs available yet.';
        }
        
        $file = file($logPath);
        $recentLines = array_slice($file, -$lines);
        
        return implode('', $recentLines);
    }
}

// Example usage:
try {
    $deployer = new DeployerClient('http://localhost:8080');
    
    // Check if service is running
    if (!$deployer->healthCheck()) {
        die("Deployment service is not available\n");
    }
    
    // Queue a deployment
    $result = $deployer->deploy(
        '/var/www/myproject',
        '/var/www/myproject/deploy.sh',
        '/var/www/myproject/deployment.log'
    );
    
    echo "Deployment queued successfully!\n";
    echo "Task ID: " . $result['taskId'] . "\n";
    echo "Status: " . $result['status'] . "\n";
    
    // In a Livewire component, you would poll for logs like this:
    // $logs = $deployer->readLogs('/var/www/myproject/deployment.log');
    
} catch (Exception $e) {
    echo "Error: " . $e->getMessage() . "\n";
}

