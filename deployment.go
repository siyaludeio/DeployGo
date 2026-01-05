package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type DeploymentTask struct {
	ProjectPath          string
	DeploymentScriptPath string
	LogPath              string
	TaskID               string
	CreatedAt            time.Time
}

func ValidatePaths(project, script, logs string) error {
	if !filepath.IsAbs(project) {
		return fmt.Errorf("project path must be absolute")
	}
	if _, err := os.Stat(project); os.IsNotExist(err) {
		return fmt.Errorf("project path does not exist")
	}

	if !filepath.IsAbs(script) {
		return fmt.Errorf("deployment script path must be absolute")
	}
	if _, err := os.Stat(script); os.IsNotExist(err) {
		return fmt.Errorf("deployment script path does not exist")
	}

	if !filepath.IsAbs(logs) {
		return fmt.Errorf("log path must be absolute")
	}
	if _, err := os.Stat(logs); os.IsNotExist(err) {
		return fmt.Errorf("log path does not exist")
	}

	return nil
}

func ExecuteDeployment(task DeploymentTask) error {
	// Open log file (truncate to create new for this deployment)
	logFilePath := filepath.Join(task.LogPath, "deployment.log")
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}
	defer logFile.Close()

	var wg sync.WaitGroup

	writeLogEntry(logFile, fmt.Sprintf("=== Deployment Started: %s ===", time.Now().Format("2006-01-02 15:04:05")))
	writeLogEntry(logFile, fmt.Sprintf("Project Path: %s", task.ProjectPath))
	writeLogEntry(logFile, fmt.Sprintf("Script Path: %s", task.DeploymentScriptPath))
	writeLogEntry(logFile, fmt.Sprintf("Task ID: %s", task.TaskID))

	// Change to project directory
	if err := os.Chdir(task.ProjectPath); err != nil {
		writeLogEntry(logFile, fmt.Sprintf("[ERROR] Failed to change directory: %v", err))
		return fmt.Errorf("failed to change to project directory: %v", err)
	}

	// Check if deployment script is executable
	scriptInfo, err := os.Stat(task.DeploymentScriptPath)
	if err != nil {
		writeLogEntry(logFile, fmt.Sprintf("[ERROR] Script not found: %v", err))
		return fmt.Errorf("deployment script not found: %v", err)
	}

	// Make script executable if needed
	if scriptInfo.Mode()&0111 == 0 {
		if err := os.Chmod(task.DeploymentScriptPath, 0755); err != nil {
			writeLogEntry(logFile, fmt.Sprintf("[WARNING] Failed to make script executable: %v", err))
		}
	}

	// Execute deployment script
	cmd := exec.Command("bash", task.DeploymentScriptPath)
	cmd.Dir = task.ProjectPath

	// Set environment variables
	cmd.Env = append(os.Environ(),
		"DEPLOYER_TASK_ID="+task.TaskID,
		"DEPLOYER_PROJECT_PATH="+task.ProjectPath,
		"DEPLOYER_LOG_PATH="+task.LogPath,
	)

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		writeLogEntry(logFile, fmt.Sprintf("[ERROR] Failed to create stdout pipe: %v", err))
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		writeLogEntry(logFile, fmt.Sprintf("[ERROR] Failed to create stderr pipe: %v", err))
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		writeLogEntry(logFile, fmt.Sprintf("[ERROR] Failed to start deployment script: %v", err))
		return fmt.Errorf("failed to start deployment script: %v", err)
	}

	// Read stdout and stderr line by line
	wg.Add(2)
	go readAndLogOutput(stdout, logFile, "STDOUT", &wg)
	go readAndLogOutput(stderr, logFile, "STDERR", &wg)

	// Wait for command to complete
	cmdErr := cmd.Wait()

	// Wait for output processing to finish
	wg.Wait()

	if cmdErr != nil {
		writeLogEntry(logFile, fmt.Sprintf("[ERROR] Deployment script exited with error: %v", cmdErr))
		return fmt.Errorf("deployment script failed: %v", cmdErr)
	}

	writeLogEntry(logFile, fmt.Sprintf("=== Deployment Completed: %s ===", time.Now().Format("2006-01-02 15:04:05")))
	return nil
}

func readAndLogOutput(pipe io.ReadCloser, logFile *os.File, prefix string, wg *sync.WaitGroup) {
	defer wg.Done()
	defer pipe.Close()
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		logEntry := fmt.Sprintf("[%s] [%s] %s\n", timestamp, prefix, line)

		if _, err := logFile.WriteString(logEntry); err != nil {
			log.Printf("Failed to write to log file: %v", err)
		}
		logFile.Sync()
	}
}

func writeLogEntry(logFile *os.File, message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s\n", timestamp, message)
	logFile.WriteString(logEntry)
	logFile.Sync()
}

func WriteLog(logPath string, message string) {
	logFilePath := filepath.Join(logPath, "deployment.log")
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Failed to open log file for writing: %v", err)
		return
	}
	defer logFile.Close()
	writeLogEntry(logFile, message)
}

func RotateLog(logDir string) error {
	activeLog := filepath.Join(logDir, "deployment.log")
	timestamp := time.Now().Format("20060102_150405")
	newLog := filepath.Join(logDir, fmt.Sprintf("deployment_%s.log", timestamp))
	if _, err := os.Stat(activeLog); os.IsNotExist(err) {
		return nil
	}
	return os.Rename(activeLog, newLog)
}
