<?php

namespace App\Console\Commands;

use Illuminate\Console\Command;
use Symfony\Component\Process\Process;

class TriggerDeployment extends Command
{
    /**
     * The name and signature of the console command.
     *
     * @var string
     */
    protected $signature = 'app:deploy {--project=} {--script=} {--logs=}';

    /**
     * The console command description.
     *
     * @var string
     */
    protected $description = 'Trigger a deploygo background task';

    /**
     * Execute the console command.
     *
     * @return int
     */
    public function handle()
    {
        // Defaults
        $project = $this->option('project') ?? base_path();
        $script = $this->option('script') ?? base_path('deploy.sh');
        $logs = $this->option('logs') ?? storage_path('logs');

        // Path to the compiled deploygo binary
        // Make sure this path is correct and accessible
        $binary = '/usr/local/bin/deploygo';

        $this->info("Triggering deployment for $project...");
        $this->info("Script: $script");
        $this->info("Logs: $logs");

        // Prepare the command
        // If you are running this command as root/ubuntu but want the deployment 
        // to run as www-data (to own the files), prepend sudo:
        // $command = ['sudo', '-u', 'www-data', $binary, 'deploy', ...];

        // Default usage (running as current user):
        $command = [
            $binary,
            'deploy',
            "--project={$project}",
            "--deployScript={$script}",
            "--logPath={$logs}",
        ];

        // Using Symfony Process to execute the binary
        $process = new Process($command);
        $process->run();

        // Check if the command itself ran successfully (the binary exits immediately with 0 if queued)
        if (!$process->isSuccessful()) {
            $this->error('Failed to trigger deployment: ' . $process->getErrorOutput());
            return 1;
        }

        // Output the result (should be "queued" or similar)
        $this->info($process->getOutput());

        return 0;
    }
}
