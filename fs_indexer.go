// Copyright 2025 Naren Yellavula
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unsafe"

	"github.com/schollz/progressbar/v3"
	"github.com/willf/bloom"
)

const (
	MaxPathLength   = 512                                                         // Fixed path length for binary representation
	CountMinWidth   = 2048                                                        // Width of Count-Min Sketch
	CountMinDepth   = 4                                                           // Depth of Count-Min Sketch
	TimestampSize   = 8                                                           // int64 timestamp (8 bytes)
	AccessCountSize = 4                                                           // int32 access count (4 bytes)
	FlagsSize       = 1                                                           // uint8 flags (1 byte)
	PathRecordSize  = MaxPathLength + TimestampSize + AccessCountSize + FlagsSize // Total: 525 bytes per record
)

// Binary flags for file metadata
const (
	FlagIsDirectory = 1 << 0
	FlagIsHidden    = 1 << 1
	FlagIsSymlink   = 1 << 2
)

type FileMetadata struct {
	Path         string
	Timestamp    *time.Time
	AccessCount  int32
	IsDirectory  bool
	IsHidden     bool
	IsSymlink    bool
	Size         int64
	LastModified time.Time
}

type RankedFile struct {
	Path     string
	Score    float64
	Metadata FileMetadata
}

// Fixed-size binary path record (525 bytes)
type PathRecord struct {
	Path        [MaxPathLength]byte // 512 bytes - null-padded path
	Timestamp   int64               // 8 bytes - Unix timestamp
	AccessCount int32               // 4 bytes - access count
	Flags       uint8               // 1 byte - flags (directory, hidden, etc.)
}

// Count-Min Sketch with fixed binary representation
type CountMinSketch struct {
	table [CountMinDepth][CountMinWidth]int32
}

func NewCountMinSketch() *CountMinSketch {
	return &CountMinSketch{}
}

func (cms *CountMinSketch) hash(item string, row int) uint32 {
	h := fnv.New32a()
	h.Write([]byte(item))
	h.Write([]byte{byte(row)}) // Salt with row number
	return h.Sum32() % CountMinWidth
}

func (cms *CountMinSketch) Add(item string, count int32) {
	for i := 0; i < CountMinDepth; i++ {
		pos := cms.hash(item, i)
		cms.table[i][pos] += count
	}
}

func (cms *CountMinSketch) Estimate(item string) int32 {
	min := cms.table[0][cms.hash(item, 0)]
	for i := 1; i < CountMinDepth; i++ {
		pos := cms.hash(item, i)
		if cms.table[i][pos] < min {
			min = cms.table[i][pos]
		}
	}
	return min
}

// Binary serialization for Count-Min Sketch
func (cms *CountMinSketch) WriteTo(w io.Writer) error {
	return binary.Write(w, binary.LittleEndian, cms.table)
}

func (cms *CountMinSketch) ReadFrom(r io.Reader) error {
	return binary.Read(r, binary.LittleEndian, &cms.table)
}

type FilesystemIndexer struct {
	bloomFilter    *bloom.BloomFilter
	countMinSketch *CountMinSketch
	pathRecords    []PathRecord
	pathIndex      map[string]int // Maps path to index in pathRecords
	rootPaths      []string       // Tracks root directories that were indexed
	config         FilesystemConfig
	isDirty        bool
}

func NewFilesystemIndexer(config FilesystemConfig) *FilesystemIndexer {
	bloomFilter := bloom.New(config.BloomFilterSize, config.BloomFilterHashes)
	countMinSketch := NewCountMinSketch()

	return &FilesystemIndexer{
		bloomFilter:    bloomFilter,
		countMinSketch: countMinSketch,
		pathRecords:    make([]PathRecord, 0, config.MaxIndexedFiles),
		pathIndex:      make(map[string]int),
		rootPaths:      make([]string, 0),
		config:         config,
		isDirty:        false,
	}
}

func (fi *FilesystemIndexer) pathToBytes(path string) [MaxPathLength]byte {
	var result [MaxPathLength]byte
	if len(path) > MaxPathLength-1 {
		path = path[:MaxPathLength-1] // Leave space for null terminator
	}
	copy(result[:], []byte(path))
	return result
}

func (fi *FilesystemIndexer) bytesToPath(bytes [MaxPathLength]byte) string {
	// Find null terminator
	end := 0
	for i, b := range bytes {
		if b == 0 {
			end = i
			break
		}
	}
	if end == 0 {
		end = len(bytes)
	}
	return string(bytes[:end])
}

func (fi *FilesystemIndexer) AddPath(path string, timestamp time.Time) (bool, int32) {
	existed := fi.bloomFilter.TestString(path)

	fi.bloomFilter.AddString(path)
	fi.countMinSketch.Add(path, 1)
	fi.isDirty = true

	if existed {
		// Update existing record
		if idx, found := fi.pathIndex[path]; found {
			fi.pathRecords[idx].Timestamp = timestamp.Unix()
			fi.pathRecords[idx].AccessCount++
			return true, fi.pathRecords[idx].AccessCount
		}
	}

	// Add new record
	if len(fi.pathRecords) >= fi.config.MaxIndexedFiles {
		log.Printf("Warning: Maximum indexed files limit (%d) reached", fi.config.MaxIndexedFiles)
		return existed, fi.countMinSketch.Estimate(path)
	}

	info, err := os.Lstat(path)
	var flags uint8
	if err == nil {
		if info.IsDir() {
			flags |= FlagIsDirectory
		}
		if strings.HasPrefix(filepath.Base(path), ".") {
			flags |= FlagIsHidden
		}
		if info.Mode()&os.ModeSymlink != 0 {
			flags |= FlagIsSymlink
		}
	}

	record := PathRecord{
		Path:        fi.pathToBytes(path),
		Timestamp:   timestamp.Unix(),
		AccessCount: fi.countMinSketch.Estimate(path),
		Flags:       flags,
	}

	fi.pathIndex[path] = len(fi.pathRecords)
	fi.pathRecords = append(fi.pathRecords, record)

	return existed, record.AccessCount
}

func (fi *FilesystemIndexer) TestMembership(path string) bool {
	return fi.bloomFilter.TestString(path)
}

func (fi *FilesystemIndexer) GetFrequency(path string) int32 {
	return fi.countMinSketch.Estimate(path)
}

func (fi *FilesystemIndexer) GetTimestamp(path string) *time.Time {
	if idx, found := fi.pathIndex[path]; found {
		if idx < len(fi.pathRecords) {
			ts := time.Unix(fi.pathRecords[idx].Timestamp, 0)
			return &ts
		}
	}
	return nil
}

func (fi *FilesystemIndexer) IndexDirectory(rootPath string) error {
	return fi.IndexDirectoryWithProgress(rootPath, false)
}

func (fi *FilesystemIndexer) IndexDirectories(rootPaths []string) error {
	return fi.IndexDirectoriesWithProgress(rootPaths, false)
}

func (fi *FilesystemIndexer) IndexDirectoryWithProgress(rootPath string, showProgress bool) error {
	log.Printf("Starting filesystem indexing for: %s", rootPath)

	// Track this root path if not already tracked
	fi.addRootPath(rootPath)

	count := 0

	var bar *progressbar.ProgressBar
	if showProgress {
		// Create progress bar with unknown total initially
		bar = progressbar.NewOptions(-1,
			progressbar.OptionSetDescription("📁 Indexing files..."),
			progressbar.OptionSetWidth(50),
			progressbar.OptionShowCount(),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "█",
				SaucerHead:    "█",
				SaucerPadding: " ",
				BarStart:      "[",
				BarEnd:        "]",
			}),
			progressbar.OptionOnCompletion(func() {
				fmt.Printf("\n✅ Indexing completed!\n")
			}),
		)
	}

	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				return nil
			}
			return err
		}

		if fi.shouldSkipPath(path) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if count >= fi.config.MaxIndexedFiles {
			if showProgress && bar != nil {
				bar.Describe("⚠️  Max files limit reached")
				bar.Finish()
			}
			return errors.New("max indexed files limit reached")
		}

		fi.AddPath(path, time.Now())
		count++

		// Update progress bar
		if showProgress && bar != nil {
			bar.Add(1)
			// Show current file being processed (truncate if too long)
			currentFile := filepath.Base(path)
			if len(currentFile) > 30 {
				currentFile = currentFile[:27] + "..."
			}
			bar.Describe(fmt.Sprintf("📁 Indexing: %s", currentFile))
		}

		return nil
	})

	if showProgress && bar != nil {
		bar.Finish()
	}

	log.Printf("Filesystem indexing completed. Indexed %d files/directories", count)
	return err
}

func (fi *FilesystemIndexer) IndexDirectoriesWithProgress(rootPaths []string, showProgress bool) error {
	if len(rootPaths) == 0 {
		return fmt.Errorf("no directories provided for indexing")
	}

	totalCount := 0
	var overallBar *progressbar.ProgressBar

	if showProgress {
		// Create overall progress bar
		overallBar = progressbar.NewOptions(-1,
			progressbar.OptionSetDescription("📁 Indexing multiple directories..."),
			progressbar.OptionSetWidth(50),
			progressbar.OptionShowCount(),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "█",
				SaucerHead:    "█",
				SaucerPadding: " ",
				BarStart:      "[",
				BarEnd:        "]",
			}),
		)
	}

	for i, rootPath := range rootPaths {
		if showProgress {
			overallBar.Describe(fmt.Sprintf("📁 [%d/%d] %s", i+1, len(rootPaths), filepath.Base(rootPath)))
		}

		// Track this root path if not already tracked
		fi.addRootPath(rootPath)

		log.Printf("Starting filesystem indexing for directory %d/%d: %s", i+1, len(rootPaths), rootPath)
		count := 0

		err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if os.IsPermission(err) {
					return nil
				}
				return err
			}

			if fi.shouldSkipPath(path) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			if totalCount >= fi.config.MaxIndexedFiles {
				if showProgress && overallBar != nil {
					overallBar.Describe("⚠️  Max files limit reached")
					overallBar.Finish()
				}
				return errors.New("max indexed files limit reached")
			}

			fi.AddPath(path, time.Now())
			count++
			totalCount++

			if showProgress && overallBar != nil {
				overallBar.Add(1)
				// Show current directory and file being processed
				currentFile := filepath.Base(path)
				if len(currentFile) > 25 {
					currentFile = currentFile[:22] + "..."
				}
				dirName := filepath.Base(rootPath)
				if len(dirName) > 15 {
					dirName = dirName[:12] + "..."
				}
				overallBar.Describe(fmt.Sprintf("📁 [%d/%d] %s: %s", i+1, len(rootPaths), dirName, currentFile))
			}

			return nil
		})

		if err != nil {
			log.Printf("Warning: Error indexing directory %s: %v", rootPath, err)
			if err.Error() == "max indexed files limit reached" {
				if showProgress && overallBar != nil {
					overallBar.Finish()
				}
				break // Stop processing remaining directories
			}
		}

		log.Printf("Completed indexing directory %s: %d files/directories", rootPath, count)
	}

	if showProgress && overallBar != nil {
		overallBar.Describe("✅ Indexing completed")
		overallBar.Finish()
	}

	log.Printf("Multi-directory indexing completed. Total indexed: %d files/directories across %d directories", totalCount, len(rootPaths))
	return nil
}

func (fi *FilesystemIndexer) shouldSkipPath(path string) bool {
	base := filepath.Base(path)

	for _, pattern := range fi.config.IgnorePatterns {
		matched, _ := filepath.Match(pattern, base)
		if matched {
			return true
		}
	}

	for _, pattern := range fi.config.IgnorePatterns {
		if strings.Contains(path, pattern) {
			return true
		}
	}

	return false
}

// addRootPath adds a root path to tracking if not already present
func (fi *FilesystemIndexer) addRootPath(rootPath string) {
	// Convert to absolute path for consistency
	absPath, err := filepath.Abs(rootPath)
	if err != nil {
		absPath = rootPath
	}

	// Check if already tracked
	for _, existing := range fi.rootPaths {
		if existing == absPath {
			return
		}
	}

	// Add new root path
	fi.rootPaths = append(fi.rootPaths, absPath)
	fi.isDirty = true
}

// GetRootPaths returns a copy of the tracked root paths
func (fi *FilesystemIndexer) GetRootPaths() []string {
	result := make([]string, len(fi.rootPaths))
	copy(result, fi.rootPaths)
	return result
}

// ReindexExistingPaths re-indexes all tracked root paths to discover new files
func (fi *FilesystemIndexer) ReindexExistingPaths(showProgress bool) error {
	if len(fi.rootPaths) == 0 {
		return nil
	}

	log.Printf("Re-indexing %d tracked root paths to discover new files", len(fi.rootPaths))

	// Filter out root paths that no longer exist
	var validRootPaths []string
	for _, rootPath := range fi.rootPaths {
		if _, err := os.Stat(rootPath); err == nil {
			validRootPaths = append(validRootPaths, rootPath)
		} else {
			log.Printf("Skipping non-existent root path: %s", rootPath)
		}
	}

	if len(validRootPaths) == 0 {
		return nil
	}

	// Update root paths to only valid ones
	fi.rootPaths = validRootPaths
	fi.isDirty = true

	// Re-index all valid root paths
	return fi.IndexDirectoriesWithProgress(validRootPaths, showProgress)
}

// RefreshIndex performs a complete refresh of all tracked paths with progress display and persistence
func (fi *FilesystemIndexer) RefreshIndex(showProgress bool, showStats bool) error {
	rootPaths := fi.GetRootPaths()
	if len(rootPaths) == 0 {
		return fmt.Errorf("no tracked paths found in index")
	}

	if showStats {
		fmt.Printf("📊 Current index: %s\n", fi.GetIndexStats())
	}

	if showProgress {
		fmt.Printf("🔄 Re-indexing %d tracked paths to discover new files...\n", len(rootPaths))
	}

	// Re-index all tracked paths
	err := fi.ReindexExistingPaths(showProgress)
	if err != nil {
		return err
	}

	// Persist the updated index
	if showProgress {
		fmt.Printf("\n💾 Saving updated index to disk...")
	}

	if persistErr := fi.PersistIndex(); persistErr != nil {
		if showProgress {
			fmt.Printf(" ❌\n")
		}
		return fmt.Errorf("failed to persist updated index: %v", persistErr)
	}

	if showProgress {
		fmt.Printf(" ✅\n")
	}

	if showStats {
		fmt.Printf("\n📊 Updated index: %s\n", fi.GetIndexStats())
	}

	return nil
}

func (fi *FilesystemIndexer) SearchFiles(query string, enableFuzzy bool) []RankedFile {
	var candidates []string
	queryLower := strings.ToLower(query)

	// Search through indexed paths
	for _, record := range fi.pathRecords {
		path := fi.bytesToPath(record.Path)

		if enableFuzzy {
			if strings.Contains(strings.ToLower(filepath.Base(path)), queryLower) ||
				strings.Contains(strings.ToLower(path), queryLower) {
				candidates = append(candidates, path)
			}
		} else {
			if strings.HasPrefix(strings.ToLower(filepath.Base(path)), queryLower) {
				candidates = append(candidates, path)
			}
		}
	}

	rankedFiles := make([]RankedFile, 0, len(candidates))

	for _, path := range candidates {
		metadata, err := fi.getFileMetadata(path)
		if err != nil {
			continue
		}

		score := fi.calculateFileScore(metadata)
		rankedFiles = append(rankedFiles, RankedFile{
			Path:     path,
			Score:    score,
			Metadata: metadata,
		})
	}

	sort.SliceStable(rankedFiles, func(i, j int) bool {
		return rankedFiles[i].Score > rankedFiles[j].Score
	})

	if len(rankedFiles) > 50 {
		rankedFiles = rankedFiles[:50]
	}

	return rankedFiles
}

func (fi *FilesystemIndexer) getFileMetadata(path string) (FileMetadata, error) {
	if idx, found := fi.pathIndex[path]; found && idx < len(fi.pathRecords) {
		record := fi.pathRecords[idx]
		timestamp := time.Unix(record.Timestamp, 0)

		metadata := FileMetadata{
			Path:        path,
			Timestamp:   &timestamp,
			AccessCount: record.AccessCount,
			IsDirectory: (record.Flags & FlagIsDirectory) != 0,
			IsHidden:    (record.Flags & FlagIsHidden) != 0,
			IsSymlink:   (record.Flags & FlagIsSymlink) != 0,
		}

		if info, err := os.Stat(path); err == nil {
			metadata.Size = info.Size()
			metadata.LastModified = info.ModTime()
		}

		return metadata, nil
	}

	return FileMetadata{}, fmt.Errorf("path not found in index: %s", path)
}

func (fi *FilesystemIndexer) calculateFileScore(metadata FileMetadata) float64 {
	if metadata.Timestamp == nil {
		return 0
	}

	now := time.Now()
	timeDelta := now.Sub(*metadata.Timestamp).Hours()

	frequencyScore := float64(metadata.AccessCount)
	recencyScore := 1 / (timeDelta + 1)

	score := (0.7 * frequencyScore) + (0.3 * recencyScore)

	if metadata.IsDirectory {
		score *= 0.8
	}

	return score
}

// Binary file format:
// Header (32 bytes):
//   - Magic number (8 bytes): "RECALLER"
//   - Version (4 bytes): uint32
//   - Record count (4 bytes): uint32
//   - Root path count (4 bytes): uint32
//   - Reserved (12 bytes)
// Root paths section (variable size):
//   - Each root path: length (4 bytes) + path string
// Bloom filter data (variable size)
// Count-Min Sketch (32KB fixed size: 4 * 2048 * 4 bytes)
// Path records (525 bytes each, fixed size)

func (fi *FilesystemIndexer) SaveToFile(filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create index file: %v", err)
	}
	defer file.Close()

	// Write header
	magic := [8]byte{'R', 'E', 'C', 'A', 'L', 'L', 'E', 'R'}
	version := uint32(2) // Increment version to support root paths
	recordCount := uint32(len(fi.pathRecords))
	rootPathCount := uint32(len(fi.rootPaths))
	reserved := [12]byte{}

	if err := binary.Write(file, binary.LittleEndian, magic); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, version); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, recordCount); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, rootPathCount); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, reserved); err != nil {
		return err
	}

	// Write root paths
	for _, rootPath := range fi.rootPaths {
		pathBytes := []byte(rootPath)
		pathLen := uint32(len(pathBytes))
		if err := binary.Write(file, binary.LittleEndian, pathLen); err != nil {
			return err
		}
		if _, err := file.Write(pathBytes); err != nil {
			return err
		}
	}

	// Write bloom filter
	if _, err := fi.bloomFilter.WriteTo(file); err != nil {
		return err
	}

	// Write Count-Min Sketch
	if err := fi.countMinSketch.WriteTo(file); err != nil {
		return err
	}

	// Write path records
	for _, record := range fi.pathRecords {
		if err := binary.Write(file, binary.LittleEndian, record); err != nil {
			return err
		}
	}

	fi.isDirty = false
	return nil
}

func (fi *FilesystemIndexer) LoadFromFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open index file: %v", err)
	}
	defer file.Close()

	// Read and verify header
	var magic [8]byte
	var version, recordCount, rootPathCount uint32
	var reserved [12]byte

	if err := binary.Read(file, binary.LittleEndian, &magic); err != nil {
		return err
	}
	if string(magic[:]) != "RECALLER" {
		return fmt.Errorf("invalid file format")
	}

	if err := binary.Read(file, binary.LittleEndian, &version); err != nil {
		return err
	}
	if version != 1 && version != 2 {
		return fmt.Errorf("unsupported file version: %d", version)
	}

	if err := binary.Read(file, binary.LittleEndian, &recordCount); err != nil {
		return err
	}

	// Handle version differences
	if version == 2 {
		if err := binary.Read(file, binary.LittleEndian, &rootPathCount); err != nil {
			return err
		}
	} else {
		// Version 1 compatibility - read old bloomSize field but ignore it
		var bloomSize uint32
		if err := binary.Read(file, binary.LittleEndian, &bloomSize); err != nil {
			return err
		}
		rootPathCount = 0
	}

	if err := binary.Read(file, binary.LittleEndian, &reserved); err != nil {
		return err
	}

	// Read root paths (only in version 2+)
	fi.rootPaths = make([]string, 0, rootPathCount)
	for i := uint32(0); i < rootPathCount; i++ {
		var pathLen uint32
		if err := binary.Read(file, binary.LittleEndian, &pathLen); err != nil {
			return err
		}
		pathBytes := make([]byte, pathLen)
		if _, err := file.Read(pathBytes); err != nil {
			return err
		}
		fi.rootPaths = append(fi.rootPaths, string(pathBytes))
	}

	// Read bloom filter
	fi.bloomFilter = bloom.New(fi.config.BloomFilterSize, fi.config.BloomFilterHashes)
	if _, err := fi.bloomFilter.ReadFrom(file); err != nil {
		return fmt.Errorf("failed to restore bloom filter: %v", err)
	}

	// Read Count-Min Sketch
	fi.countMinSketch = NewCountMinSketch()
	if err := fi.countMinSketch.ReadFrom(file); err != nil {
		return err
	}

	// Read path records
	fi.pathRecords = make([]PathRecord, recordCount)
	fi.pathIndex = make(map[string]int, recordCount)

	for i := uint32(0); i < recordCount; i++ {
		var record PathRecord
		if err := binary.Read(file, binary.LittleEndian, &record); err != nil {
			return err
		}
		fi.pathRecords[i] = record
		path := fi.bytesToPath(record.Path)
		fi.pathIndex[path] = int(i)
	}

	fi.isDirty = false
	return nil
}

func (fi *FilesystemIndexer) GetIndexPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".recaller_fs_index.bin"
	}
	return filepath.Join(homeDir, ".recaller_fs_index.bin")
}

func (fi *FilesystemIndexer) LoadOrCreateIndex() error {
	indexPath := fi.GetIndexPath()

	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		log.Printf("No existing filesystem index found, will create new one")
		return nil
	}

	log.Printf("Loading existing filesystem index from: %s", indexPath)
	return fi.LoadFromFile(indexPath)
}

func (fi *FilesystemIndexer) PersistIndex() error {
	if !fi.isDirty {
		return nil
	}

	indexPath := fi.GetIndexPath()
	log.Printf("Persisting filesystem index to: %s", indexPath)
	return fi.SaveToFile(indexPath)
}

func (fi *FilesystemIndexer) GetIndexStats() string {
	indexSize := len(fi.pathRecords) * int(unsafe.Sizeof(PathRecord{}))
	sketchSize := CountMinDepth * CountMinWidth * 4 // int32 = 4 bytes
	bloomSize := int(fi.bloomFilter.Cap() / 8)      // Approximate bloom filter size in bytes

	return fmt.Sprintf("Index Stats: %d files, Memory: %.2fKB (Records: %.2fKB, Sketch: %.2fKB, Bloom: %.2fKB)",
		len(fi.pathRecords),
		float64(indexSize+sketchSize+bloomSize)/1024,
		float64(indexSize)/1024,
		float64(sketchSize)/1024,
		float64(bloomSize)/1024)
}

// CleanupOptions defines options for index cleanup
type CleanupOptions struct {
	Path          string // Optional path prefix filter
	RemoveStale   bool   // Remove non-existent files
	OlderThanDays int    // Remove entries older than N days
	ShowProgress  bool   // Show progress bar
}

// CleanupStats contains statistics from cleanup operation
type CleanupStats struct {
	TotalEntries   int
	RemovedEntries int
	StaleFiles     int
	OldFiles       int
	FreedKB        float64
}

// CleanupIndex removes stale and old entries from the filesystem index
func (fi *FilesystemIndexer) CleanupIndex(options CleanupOptions) (*CleanupStats, error) {
	stats := &CleanupStats{
		TotalEntries: len(fi.pathRecords),
	}

	var bar *progressbar.ProgressBar
	if options.ShowProgress {
		bar = progressbar.NewOptions(len(fi.pathRecords),
			progressbar.OptionSetDescription("🧹 Cleaning index..."),
			progressbar.OptionSetWidth(50),
			progressbar.OptionShowCount(),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "█",
				SaucerHead:    "█",
				SaucerPadding: " ",
				BarStart:      "[",
				BarEnd:        "]",
			}),
		)
	}

	oldThreshold := time.Now().AddDate(0, 0, -options.OlderThanDays)
	var validRecords []PathRecord
	var validPaths []string
	removedPaths := make(map[string]bool)

	for _, record := range fi.pathRecords {
		if bar != nil {
			bar.Add(1)
		}

		path := fi.bytesToPath(record.Path)
		shouldRemove := false

		// Check path prefix filter
		if options.Path != "" {
			matched := strings.HasPrefix(path, options.Path)
			if matched {
				shouldRemove = true
				stats.RemovedEntries++
			}
		}

		// Check if file still exists (stale check)
		if !shouldRemove && options.RemoveStale {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				shouldRemove = true
				stats.StaleFiles++
				stats.RemovedEntries++
			}
		}

		// Check age threshold
		if !shouldRemove && options.OlderThanDays > 0 {
			recordTime := time.Unix(record.Timestamp, 0)
			if recordTime.Before(oldThreshold) {
				shouldRemove = true
				stats.OldFiles++
				stats.RemovedEntries++
			}
		}

		if shouldRemove {
			removedPaths[path] = true
		} else {
			validRecords = append(validRecords, record)
			validPaths = append(validPaths, path)
		}
	}

	if bar != nil {
		bar.Finish()
	}

	// Calculate freed space
	removedCount := len(fi.pathRecords) - len(validRecords)
	stats.FreedKB = float64(removedCount*int(unsafe.Sizeof(PathRecord{}))) / 1024

	// Rebuild index structures if anything was removed
	if len(removedPaths) > 0 {
		// Rebuild path index map
		newPathIndex := make(map[string]int)
		for i, path := range validPaths {
			newPathIndex[path] = i
		}

		// Create new bloom filter and count-min sketch
		newBloomFilter := bloom.New(fi.config.BloomFilterSize, fi.config.BloomFilterHashes)
		newCountMinSketch := NewCountMinSketch()

		// Re-populate bloom filter and sketch with valid entries
		for _, record := range validRecords {
			path := fi.bytesToPath(record.Path)
			newBloomFilter.AddString(path)
			newCountMinSketch.Add(path, record.AccessCount)
		}

		// Update indexer state
		fi.pathRecords = validRecords
		fi.pathIndex = newPathIndex
		fi.bloomFilter = newBloomFilter
		fi.countMinSketch = newCountMinSketch
		fi.isDirty = true
	}

	return stats, nil
}

// CleanupByPath removes all entries matching a specific path prefix
func (fi *FilesystemIndexer) CleanupByPath(pathPrefix string, showProgress bool) (*CleanupStats, error) {
	return fi.CleanupIndex(CleanupOptions{
		Path:         pathPrefix,
		ShowProgress: showProgress,
	})
}

// CleanupStaleEntries removes entries for files that no longer exist
func (fi *FilesystemIndexer) CleanupStaleEntries(showProgress bool) (*CleanupStats, error) {
	return fi.CleanupIndex(CleanupOptions{
		RemoveStale:  true,
		ShowProgress: showProgress,
	})
}

// CleanupOldEntries removes entries older than specified days
func (fi *FilesystemIndexer) CleanupOldEntries(olderThanDays int, showProgress bool) (*CleanupStats, error) {
	return fi.CleanupIndex(CleanupOptions{
		OlderThanDays: olderThanDays,
		ShowProgress:  showProgress,
	})
}

// FullCleanup performs comprehensive cleanup (stale + old entries)
func (fi *FilesystemIndexer) FullCleanup(olderThanDays int, showProgress bool) (*CleanupStats, error) {
	return fi.CleanupIndex(CleanupOptions{
		RemoveStale:   true,
		OlderThanDays: olderThanDays,
		ShowProgress:  showProgress,
	})
}

// ClearIndex completely clears the filesystem index
func (fi *FilesystemIndexer) ClearIndex() error {
	fi.pathRecords = fi.pathRecords[:0]
	fi.pathIndex = make(map[string]int)
	fi.rootPaths = fi.rootPaths[:0]
	fi.bloomFilter = bloom.New(fi.config.BloomFilterSize, fi.config.BloomFilterHashes)
	fi.countMinSketch = NewCountMinSketch()
	fi.isDirty = true
	return nil
}

// GetIndexFileSize returns the size of the index file on disk
func (fi *FilesystemIndexer) GetIndexFileSize() (int64, error) {
	indexPath := fi.GetIndexPath()
	if info, err := os.Stat(indexPath); err == nil {
		return info.Size(), nil
	} else if os.IsNotExist(err) {
		return 0, nil
	} else {
		return 0, err
	}
}

// HasIndexedFiles returns true if the index contains any files
func (fi *FilesystemIndexer) HasIndexedFiles() bool {
	return len(fi.pathRecords) > 0
}
