package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"
)

func main() {
	deployCmd := flag.NewFlagSet("deploy", flag.ExitOnError)
	projectPath := deployCmd.String("project", "", "Absolute path to the project directory")
	deployScript := deployCmd.String("deployScript", "", "Absolute path to the deployment script")
	logPath := deployCmd.String("logPath", "", "Absolute path to the directory where logs will be stored")

	internalCmd := flag.NewFlagSet("internal-run", flag.ExitOnError)
	taskFile := internalCmd.String("taskFile", "", "Path to the temporary task file")

	if len(os.Args) < 2 {
		fmt.Println("Usage: deploygo deploy --project={path} --deployScript={path} --logPath={path}")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "deploy":
		deployCmd.Parse(os.Args[2:])
		handleDeploy(*projectPath, *deployScript, *logPath)
	case "internal-run":
		internalCmd.Parse(os.Args[2:])
		handleInternalRun(*taskFile)
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func handleDeploy(project, script, logs string) {
	if project == "" || script == "" || logs == "" {
		fmt.Println("All flags are required: --project, --deployScript, --logPath")
		os.Exit(1)
	}

	// Validate paths
	if err := ValidatePaths(project, script, logs); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Create task
	task := DeploymentTask{
		ProjectPath:          project,
		DeploymentScriptPath: script,
		LogPath:              logs,
		TaskID:               fmt.Sprintf("%d", time.Now().UnixNano()),
		CreatedAt:            time.Now(),
	}

	// Create temporary file for task
	tmpFile, err := os.CreateTemp("", "deploy_task_*.json")
	if err != nil {
		fmt.Printf("Error: Failed to create temporary task file: %v\n", err)
		os.Exit(1)
	}
	// We don't remove the file here, the child process will do it
	// defer os.Remove(tmpFile.Name())

	data, err := json.Marshal(task)
	if err != nil {
		fmt.Printf("Error: Failed to marshal task: %v\n", err)
		os.Exit(1)
	}

	if _, err := tmpFile.Write(data); err != nil {
		fmt.Printf("Error: Failed to write task file: %v\n", err)
		os.Exit(1)
	}
	tmpFile.Close()

	// Spawn background process
	// We use the same executable
	executable, err := os.Executable()
	if err != nil {
		fmt.Printf("Error: Failed to get executable path: %v\n", err)
		os.Exit(1)
	}

	cmd := exec.Command(executable, "internal-run", "--taskFile", tmpFile.Name())

	// Detach process
	// On Unix-like systems, this prevents the child from being killed when parent exits
	// We rely on Start() and not waiting.

	if err := cmd.Start(); err != nil {
		fmt.Printf("Error: Failed to start background process: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Deployment started in background for task %s\n", task.TaskID)
	// Parent exits now
}

func handleInternalRun(taskFile string) {
	if taskFile == "" {
		fmt.Println("Error: taskFile is required for internal-run")
		os.Exit(1)
	}

	// Read task file
	data, err := os.ReadFile(taskFile)
	if err != nil {
		// Log error to somewhere? We don't have log path yet.
		// Try to read just to get log path if possible or fail silently/stderr
		fmt.Fprintf(os.Stderr, "Failed to read task file: %v\n", err)
		os.Exit(1)
	}

	var task DeploymentTask
	if err := json.Unmarshal(data, &task); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to unmarshal task: %v\n", err)
		os.Exit(1)
	}

	// Execute deployment
	if err := ExecuteDeployment(task); err != nil {
		WriteLog(task.LogPath, fmt.Sprintf("[ERROR] Deployment failed: %v", err))
	} else {
		WriteLog(task.LogPath, "[SUCCESS] Deployment completed successfully")
	}

	// Rotate log file
	if err := RotateLog(task.LogPath); err != nil {
		// Just log error to active log if possible
	}

	// Clean up task file
	os.Remove(taskFile)
}
