package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	tempDir     = "/tmp/deployer"
	queueDir    = "/tmp/deployer/queue"
	defaultPort = "8080"
)

type DeploymentRequest struct {
	ProjectPath          string `json:"projectPath"`
	DeploymentScriptPath string `json:"deploymentScriptPath"`
	LogPath              string `json:"logPath"`
}

type DeploymentTask struct {
	ProjectPath          string
	DeploymentScriptPath string
	LogPath              string
	TaskID               string
	CreatedAt            time.Time
}

func main() {
	// Create necessary directories
	if err := os.MkdirAll(queueDir, 0755); err != nil {
		log.Fatalf("Failed to create queue directory: %v", err)
	}

	// Start file watcher in background
	go startFileWatcher()

	// Start HTTP server
	http.HandleFunc("/deploy", handleDeployRequest)
	http.HandleFunc("/health", handleHealthCheck)

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	log.Printf("Deployment service started on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func handleDeployRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DeploymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Validate paths
	if err := validatePaths(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create deployment task
	task := DeploymentTask{
		ProjectPath:          req.ProjectPath,
		DeploymentScriptPath: req.DeploymentScriptPath,
		LogPath:              req.LogPath,
		TaskID:               fmt.Sprintf("%d", time.Now().UnixNano()),
		CreatedAt:            time.Now(),
	}

	// Write to temporary file
	if err := writeTaskToFile(task); err != nil {
		http.Error(w, fmt.Sprintf("Failed to queue deployment: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "queued",
		"taskId":  task.TaskID,
		"message": "Deployment queued successfully",
	})
}

func validatePaths(req DeploymentRequest) error {
	// Validate project path
	if !filepath.IsAbs(req.ProjectPath) {
		return fmt.Errorf("projectPath must be an absolute path")
	}
	if _, err := os.Stat(req.ProjectPath); os.IsNotExist(err) {
		return fmt.Errorf("projectPath does not exist: %s", req.ProjectPath)
	}

	// Validate deployment script path
	if !filepath.IsAbs(req.DeploymentScriptPath) {
		return fmt.Errorf("deploymentScriptPath must be an absolute path")
	}
	if _, err := os.Stat(req.DeploymentScriptPath); os.IsNotExist(err) {
		return fmt.Errorf("deploymentScriptPath does not exist: %s", req.DeploymentScriptPath)
	}

	// Validate log path directory exists and is writable
	logDir := req.LogPath
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		return fmt.Errorf("logPath directory does not exist: %s", logDir)
	}

	// Check if log directory is writable
	testFile := filepath.Join(logDir, ".deployer_test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("logPath directory is not writable: %s", logDir)
	}
	os.Remove(testFile)

	return nil
}

func writeTaskToFile(task DeploymentTask) error {
	taskFile := filepath.Join(queueDir, fmt.Sprintf("task_%s.json", task.TaskID))
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %v", err)
	}

	if err := os.WriteFile(taskFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write task file: %v", err)
	}

	return nil
}
