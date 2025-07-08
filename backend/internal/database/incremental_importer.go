package database

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// IncrementalImporter handles smart importing of only changed files
type IncrementalImporter struct {
	repo   *SessionRepository
	db     *Database
	logger *logrus.Logger
	ctx    context.Context
}

// NewIncrementalImporter creates a new incremental importer
func NewIncrementalImporter(ctx context.Context, repo *SessionRepository, db *Database, logger *logrus.Logger) *IncrementalImporter {
	return &IncrementalImporter{
		repo:   repo,
		db:     db,
		logger: logger,
		ctx:    ctx,
	}
}

// ImportClaudeDirectory performs an intelligent import of only changed files
func (i *IncrementalImporter) ImportClaudeDirectory(claudeDir string, forceInitial bool) error {
	projectsDir := filepath.Join(claudeDir, "projects")
	
	// Determine run type
	runType := "incremental"
	if forceInitial {
		runType = "initial"
	} else if isFirstRun, _ := i.isFirstRun(); isFirstRun {
		runType = "initial"
	}

	// Start import run tracking
	importRun, err := i.startImportRun(runType)
	if err != nil {
		return fmt.Errorf("failed to start import run: %w", err)
	}

	i.logger.WithFields(logrus.Fields{
		"run_type": runType,
		"run_id":   importRun.ID,
	}).Info("Starting incremental import")

	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		i.finishImportRun(importRun.ID, "failed", fmt.Sprintf("failed to read projects directory: %v", err))
		return fmt.Errorf("failed to read projects directory: %w", err)
	}

	// Scan all files first to identify what needs processing
	filesToProcess, totalFiles, err := i.identifyFilesToProcess(projectsDir, entries, runType == "initial")
	if err != nil {
		i.finishImportRun(importRun.ID, "failed", fmt.Sprintf("failed to identify files: %v", err))
		return fmt.Errorf("failed to identify files to process: %w", err)
	}

	i.logger.WithFields(logrus.Fields{
		"total_files":         totalFiles,
		"files_to_process":    len(filesToProcess),
		"files_to_skip":       totalFiles - len(filesToProcess),
		"run_type":            runType,
	}).Info("File processing plan")

	if len(filesToProcess) == 0 {
		i.logger.Info("No files need processing - all up to date")
		i.finishImportRun(importRun.ID, "completed", "")
		return nil
	}

	// Process files with progress tracking
	totalSessions := 0
	totalMessages := 0
	startTime := time.Now()

	for idx, fileInfo := range filesToProcess {
		// Check for cancellation
		select {
		case <-i.ctx.Done():
			i.logger.Info("Import cancelled by context")
			i.finishImportRun(importRun.ID, "cancelled", "cancelled by user")
			return i.ctx.Err()
		default:
		}

		// Import the file
		sessions, messages, err := i.processFile(fileInfo)
		if err != nil {
			i.logger.WithError(err).WithField("file", fileInfo.FilePath).Error("Failed to process file")
			i.markFileError(fileInfo.FilePath, err.Error())
			continue
		}

		totalSessions += sessions
		totalMessages += messages

		// Log progress every 10 files or large files
		if (idx+1)%10 == 0 || fileInfo.SizeMB > 5 {
			elapsed := time.Since(startTime)
			remaining := time.Duration(float64(elapsed) * float64(len(filesToProcess)-(idx+1)) / float64(idx+1))
			
			i.logger.WithFields(logrus.Fields{
				"processed":           idx + 1,
				"total":               len(filesToProcess),
				"progress_pct":        fmt.Sprintf("%.1f%%", float64(idx+1)*100/float64(len(filesToProcess))),
				"sessions_imported":   totalSessions,
				"messages_imported":   totalMessages,
				"elapsed":             elapsed.Round(time.Second),
				"estimated_remaining": remaining.Round(time.Second),
			}).Info("Import progress")
		}
	}

	// Update import run with final stats
	_, err = i.db.Exec(`
		UPDATE import_runs 
		SET end_time = CURRENT_TIMESTAMP, 
		    status = 'completed',
		    files_processed = ?,
		    files_skipped = ?,
		    sessions_imported = ?,
		    messages_imported = ?
		WHERE id = ?
	`, len(filesToProcess), totalFiles-len(filesToProcess), totalSessions, totalMessages, importRun.ID)

	if err != nil {
		i.logger.WithError(err).Error("Failed to update import run")
	}

	duration := time.Since(startTime)
	i.logger.WithFields(logrus.Fields{
		"files_processed":   len(filesToProcess),
		"files_skipped":     totalFiles - len(filesToProcess),
		"sessions_imported": totalSessions,
		"messages_imported": totalMessages,
		"duration":          duration.Round(time.Second),
		"run_type":          runType,
	}).Info("Incremental import completed")

	return nil
}

// FileToProcess represents a file that needs to be imported
type FileToProcess struct {
	FilePath    string
	ProjectInfo ProjectInfo
	SizeMB      float64
	ModTime     time.Time
	NeedsUpdate bool
}

// identifyFilesToProcess scans directories and determines which files need processing
func (i *IncrementalImporter) identifyFilesToProcess(projectsDir string, entries []os.DirEntry, forceAll bool) ([]FileToProcess, int, error) {
	var filesToProcess []FileToProcess
	totalFiles := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectPath := entry.Name()
		projectDir := filepath.Join(projectsDir, projectPath)
		projectInfo := i.parseProjectPath(projectPath)
		
		sessionFiles, err := os.ReadDir(projectDir)
		if err != nil {
			i.logger.WithError(err).WithField("project", projectPath).Warn("Failed to read project directory")
			continue
		}

		for _, sessionFile := range sessionFiles {
			if !strings.HasSuffix(sessionFile.Name(), ".jsonl") {
				continue
			}

			totalFiles++
			sessionPath := filepath.Join(projectDir, sessionFile.Name())
			
			// Get file info
			fileInfo, err := sessionFile.Info()
			if err != nil {
				continue
			}

			// Check if file needs processing
			needsProcessing, err := i.fileNeedsProcessing(sessionPath, fileInfo.ModTime(), fileInfo.Size(), forceAll)
			if err != nil {
				i.logger.WithError(err).WithField("file", sessionPath).Debug("Error checking file status, will process")
				needsProcessing = true
			}

			if needsProcessing {
				filesToProcess = append(filesToProcess, FileToProcess{
					FilePath:    sessionPath,
					ProjectInfo: projectInfo,
					SizeMB:      float64(fileInfo.Size()) / (1024 * 1024),
					ModTime:     fileInfo.ModTime(),
					NeedsUpdate: true,
				})
			}
		}
	}

	return filesToProcess, totalFiles, nil
}

// fileNeedsProcessing checks if a file needs to be imported/updated
func (i *IncrementalImporter) fileNeedsProcessing(filePath string, modTime time.Time, size int64, forceAll bool) (bool, error) {
	if forceAll {
		return true, nil
	}

	var fw FileWatcher
	err := i.db.Get(&fw, "SELECT * FROM file_watchers WHERE file_path = ?", filePath)
	if err != nil {
		// File not tracked yet, needs processing
		return true, nil
	}

	// Check if file has been modified since last processing
	if modTime.After(fw.LastModified) || size != fw.FileSize {
		return true, nil
	}

	// Check if import was successful
	if fw.ImportStatus != "completed" {
		return true, nil
	}

	return false, nil
}

// processFile imports a single file and updates tracking
func (i *IncrementalImporter) processFile(fileInfo FileToProcess) (int, int, error) {
	// Create importer for this file
	importer := NewImporterWithContext(i.ctx, i.repo, i.logger)
	
	// Mark file as being processed
	i.markFileProcessing(fileInfo.FilePath, fileInfo.ModTime, int64(fileInfo.SizeMB*1024*1024))
	
	// Import the file
	sessions, messages, err := importer.ImportJSONLFile(fileInfo.FilePath, fileInfo.ProjectInfo)
	if err != nil {
		i.markFileError(fileInfo.FilePath, err.Error())
		return 0, 0, err
	}

	// Mark file as completed
	i.markFileCompleted(fileInfo.FilePath, sessions, messages)
	
	return sessions, messages, nil
}

// markFileProcessing marks a file as being processed
func (i *IncrementalImporter) markFileProcessing(filePath string, modTime time.Time, size int64) {
	_, err := i.db.Exec(`
		INSERT OR REPLACE INTO file_watchers 
		(file_path, last_modified, file_size, import_status, updated_at)
		VALUES (?, ?, ?, 'processing', CURRENT_TIMESTAMP)
	`, filePath, modTime, size)
	
	if err != nil {
		i.logger.WithError(err).WithField("file", filePath).Error("Failed to mark file as processing")
	}
}

// markFileCompleted marks a file as successfully processed
func (i *IncrementalImporter) markFileCompleted(filePath string, sessions, messages int) {
	_, err := i.db.Exec(`
		UPDATE file_watchers 
		SET import_status = 'completed',
		    sessions_imported = ?,
		    messages_imported = ?,
		    last_processed = CURRENT_TIMESTAMP,
		    last_error = NULL,
		    updated_at = CURRENT_TIMESTAMP
		WHERE file_path = ?
	`, sessions, messages, filePath)
	
	if err != nil {
		i.logger.WithError(err).WithField("file", filePath).Error("Failed to mark file as completed")
	}
}

// markFileError marks a file as failed to process
func (i *IncrementalImporter) markFileError(filePath string, errorMsg string) {
	_, err := i.db.Exec(`
		UPDATE file_watchers 
		SET import_status = 'failed',
		    last_error = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE file_path = ?
	`, errorMsg, filePath)
	
	if err != nil {
		i.logger.WithError(err).WithField("file", filePath).Error("Failed to mark file as failed")
	}
}

// startImportRun creates a new import run record
func (i *IncrementalImporter) startImportRun(runType string) (*ImportRun, error) {
	result, err := i.db.Exec(`
		INSERT INTO import_runs (run_type, start_time, status)
		VALUES (?, CURRENT_TIMESTAMP, 'running')
	`, runType)
	
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &ImportRun{
		ID:        int(id),
		RunType:   runType,
		StartTime: time.Now(),
		Status:    "running",
	}, nil
}

// finishImportRun updates an import run with final status
func (i *IncrementalImporter) finishImportRun(runID int, status, errorMsg string) {
	var errorPtr *string
	if errorMsg != "" {
		errorPtr = &errorMsg
	}

	_, err := i.db.Exec(`
		UPDATE import_runs 
		SET end_time = CURRENT_TIMESTAMP, status = ?, error_message = ?
		WHERE id = ?
	`, status, errorPtr, runID)
	
	if err != nil {
		i.logger.WithError(err).Error("Failed to finish import run")
	}
}

// isFirstRun checks if this is the first time we're running an import
func (i *IncrementalImporter) isFirstRun() (bool, error) {
	var count int
	err := i.db.Get(&count, "SELECT COUNT(*) FROM import_runs WHERE status = 'completed'")
	if err != nil {
		return true, err // Assume first run if we can't check
	}
	return count == 0, nil
}

// parseProjectPath extracts project information (reused from original importer)
func (i *IncrementalImporter) parseProjectPath(projectPath string) ProjectInfo {
	// Remove leading hyphen
	decodedPath := projectPath
	if strings.HasPrefix(decodedPath, "-") {
		decodedPath = strings.TrimPrefix(decodedPath, "-")
	}
	
	// Extract project name using the same logic as the original importer
	parts := strings.Split(decodedPath, "-")
	var projectName string
	
	if len(parts) >= 4 {
		if strings.Contains(decodedPath, "Documents-GitHub") {
			githubIndex := -1
			for idx, part := range parts {
				if part == "GitHub" {
					githubIndex = idx
					break
				}
			}
			if githubIndex >= 0 && githubIndex < len(parts)-1 {
				projectName = strings.Join(parts[githubIndex+1:], "-")
			} else {
				projectName = parts[len(parts)-1]
			}
		} else if strings.Contains(decodedPath, "ccswitch-worktrees") {
			worktreeIndex := -1
			for idx, part := range parts {
				if part == "worktrees" {
					worktreeIndex = idx
					break
				}
			}
			if worktreeIndex >= 0 && worktreeIndex < len(parts)-1 {
				projectName = strings.Join(parts[worktreeIndex+1:], "-")
			} else {
				projectName = parts[len(parts)-1]
			}
		} else {
			projectName = parts[len(parts)-1]
		}
	} else {
		projectName = parts[len(parts)-1]
	}

	actualPath := strings.ReplaceAll(decodedPath, "-", "/")

	return ProjectInfo{
		ProjectPath: actualPath,
		ProjectName: projectName,
		FilePath:    projectPath,
	}
}

// calculateFileHash calculates a simple hash of file content for change detection
func (i *IncrementalImporter) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}