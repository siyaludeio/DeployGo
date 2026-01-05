package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

func startFileWatcher() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Failed to create file watcher: %v", err)
	}
	defer watcher.Close()

	// Watch the queue directory
	if err := watcher.Add(queueDir); err != nil {
		log.Fatalf("Failed to watch queue directory: %v", err)
	}

	log.Printf("File watcher started, monitoring: %s", queueDir)

	// Process existing files in queue
	processExistingFiles()

	// Watch for new files
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Create == fsnotify.Create {
				if strings.HasPrefix(filepath.Base(event.Name), "task_") && strings.HasSuffix(event.Name, ".json") {
					log.Printf("New deployment task detected: %s", event.Name)
					// Small delay to ensure file is fully written
					time.Sleep(100 * time.Millisecond)
					go processDeploymentTask(event.Name)
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("File watcher error: %v", err)
		}
	}
}

func processExistingFiles() {
	files, err := os.ReadDir(queueDir)
	if err != nil {
		log.Printf("Failed to read queue directory: %v", err)
		return
	}

	for _, file := range files {
		if strings.HasPrefix(file.Name(), "task_") && strings.HasSuffix(file.Name(), ".json") {
			taskFile := filepath.Join(queueDir, file.Name())
			go processDeploymentTask(taskFile)
		}
	}
}

func processDeploymentTask(taskFile string) {
	// Read task file
	data, err := os.ReadFile(taskFile)
	if err != nil {
		log.Printf("Failed to read task file %s: %v", taskFile, err)
		return
	}

	var task DeploymentTask
	if err := json.Unmarshal(data, &task); err != nil {
		log.Printf("Failed to unmarshal task file %s: %v", taskFile, err)
		return
	}

	log.Printf("Processing deployment task: %s", task.TaskID)

	// Execute deployment
	if err := executeDeployment(task); err != nil {
		log.Printf("Deployment failed for task %s: %v", task.TaskID, err)
		writeLog(task.LogPath, fmt.Sprintf("[ERROR] Deployment failed: %v", err))
	} else {
		log.Printf("Deployment completed successfully for task %s", task.TaskID)
		writeLog(task.LogPath, "[SUCCESS] Deployment completed successfully")
	}

	// Rotate log file
	if err := rotateLog(task.LogPath); err != nil {
		log.Printf("Failed to rotate log file: %v", err)
	}

	// Remove task file after processing
	os.Remove(taskFile)
}

func executeDeployment(task DeploymentTask) error {
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

	// Set environment variables for zero downtime deployment
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

		// Write to log file (non-blocking, file is opened in append mode)
		if _, err := logFile.WriteString(logEntry); err != nil {
			log.Printf("Failed to write to log file: %v", err)
		}

		// Also flush to ensure data is written immediately
		logFile.Sync()
	}
}

func writeLogEntry(logFile *os.File, message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s\n", timestamp, message)
	logFile.WriteString(logEntry)
	logFile.Sync()
}

func writeLog(logPath string, message string) {
	logFilePath := filepath.Join(logPath, "deployment.log")
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Failed to open log file for writing: %v", err)
		return
	}
	defer logFile.Close()
	writeLogEntry(logFile, message)
}

func rotateLog(logDir string) error {
	activeLog := filepath.Join(logDir, "deployment.log")
	timestamp := time.Now().Format("20060102_150405")
	newLog := filepath.Join(logDir, fmt.Sprintf("deployment_%s.log", timestamp))
	return os.Rename(activeLog, newLog)
}
