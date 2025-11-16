use std::collections::HashMap;
use std::fs::{self, File};
use std::io::{Read, Write};
use std::path::{Path, PathBuf};
use std::time::Duration;

use anyhow::{Context, Result, anyhow, bail};
use chrono::{DateTime, Local, Utc};
use directories::BaseDirs;
use indicatif::{ProgressBar, ProgressStyle};
use walkdir::WalkDir;

use crate::config::FilesystemConfig;

const MAX_PATH_LENGTH: usize = 512;
const COUNT_MIN_WIDTH: usize = 2048;
const COUNT_MIN_DEPTH: usize = 4;
const PATH_RECORD_SIZE: usize = MAX_PATH_LENGTH + 8 + 4 + 1; // 525 bytes

const FLAG_IS_DIRECTORY: u8 = 1 << 0;
const FLAG_IS_HIDDEN: u8 = 1 << 1;
const FLAG_IS_SYMLINK: u8 = 1 << 2;

#[derive(Clone)]
#[allow(dead_code)]
pub struct FileMetadata {
    pub path: String,
    pub timestamp: Option<DateTime<Utc>>,
    pub access_count: i32,
    pub is_directory: bool,
    pub is_hidden: bool,
    pub is_symlink: bool,
    pub size: Option<u64>,
    pub last_modified: Option<DateTime<Local>>,
}

#[derive(Clone)]
#[allow(dead_code)]
pub struct RankedFile {
    pub path: String,
    pub score: f64,
    pub metadata: FileMetadata,
}

#[derive(Default)]
pub struct CleanupOptions {
    pub path: Option<String>,
    pub remove_stale: bool,
    pub older_than_days: i32,
    pub show_progress: bool,
}

#[derive(Default, Debug)]
pub struct CleanupStats {
    pub total_entries: usize,
    pub removed_entries: usize,
    pub stale_files: usize,
    pub old_files: usize,
    pub freed_kb: f64,
}

pub struct FilesystemIndexer {
    bloom_filter: BloomFilter,
    count_min: CountMinSketch,
    path_records: Vec<PathRecord>,
    path_index: HashMap<String, usize>,
    root_paths: Vec<String>,
    pub config: FilesystemConfig,
    is_dirty: bool,
}

impl FilesystemIndexer {
    pub fn new(config: FilesystemConfig) -> Self {
        let bloom_filter = BloomFilter::new(config.bloom_filter_size, config.bloom_filter_hashes);
        Self {
            bloom_filter,
            count_min: CountMinSketch::new(),
            path_records: Vec::with_capacity(config.max_indexed_files),
            path_index: HashMap::new(),
            root_paths: Vec::new(),
            config,
            is_dirty: false,
        }
    }

    pub fn add_path(
        &mut self,
        path: &str,
        timestamp: Option<DateTime<Utc>>,
        increment_access: bool,
    ) {
        let existed = self.path_index.contains_key(path);
        if existed {
            if let Some(&idx) = self.path_index.get(path) {
                if let Some(record) = self.path_records.get_mut(idx) {
                    if let Some(ts) = timestamp {
                        record.timestamp = ts.timestamp();
                    }
                    if increment_access {
                        record.access_count = record.access_count.saturating_add(1);
                        record.timestamp = Utc::now().timestamp();
                    }
                }
            }
        } else {
            if self.path_records.len() >= self.config.max_indexed_files {
                return;
            }

            let flags = match fs::symlink_metadata(path) {
                Ok(meta) => file_flags(&meta, Path::new(path)),
                Err(_) => 0,
            };

            let ts = timestamp.map(|t| t.timestamp()).unwrap_or_else(|| {
                if increment_access {
                    Utc::now().timestamp()
                } else {
                    0
                }
            });

            let mut access = 0;
            if increment_access {
                access = self.count_min.estimate(path);
                if access == 0 {
                    access = 1;
                }
            }

            let record = PathRecord::new(path, ts, access, flags);
            let idx = self.path_records.len();
            self.path_index.insert(path.to_string(), idx);
            self.path_records.push(record);
        }

        self.bloom_filter.add(path.as_bytes());
        if increment_access {
            self.count_min.add(path, 1);
        }
        self.is_dirty = true;
    }

    pub fn index_directories_with_progress(
        &mut self,
        paths: &[PathBuf],
        show_progress: bool,
    ) -> Result<()> {
        if paths.is_empty() {
            bail!("no directories provided for indexing");
        }

        let bar = show_progress.then(|| {
            ProgressBar::new_spinner().with_style(
                ProgressStyle::with_template("{spinner} {msg}")
                    .unwrap_or_else(|_| ProgressStyle::default_spinner()),
            )
        });

        for (idx, path) in paths.iter().enumerate() {
            if let Some(bar) = bar.as_ref() {
                bar.enable_steady_tick(Duration::from_millis(120));
                bar.set_message(format!("[{}/{}] {}", idx + 1, paths.len(), path.display()));
            }

            self.add_root_path(path);

            for entry in WalkDir::new(path) {
                let entry = match entry {
                    Ok(e) => e,
                    Err(err) => {
                        if err.io_error().map(|e| e.kind())
                            == Some(std::io::ErrorKind::PermissionDenied)
                        {
                            continue;
                        }
                        return Err(err.into());
                    }
                };

                if self.path_records.len() >= self.config.max_indexed_files {
                    if let Some(bar) = bar.as_ref() {
                        bar.finish_with_message("⚠️ Max indexed files reached");
                    }
                    bail!("max indexed files limit reached");
                }

                if self.should_skip(entry.path()) {
                    if entry.file_type().is_dir() {
                        continue;
                    }
                }

                let path_str = entry.path().to_string_lossy().to_string();
                self.add_path(&path_str, None, false);
            }
        }

        if let Some(bar) = bar {
            bar.finish_with_message("✔️ Indexing completed");
        }

        Ok(())
    }

    pub fn index_directory_with_progress(&mut self, path: &Path, show: bool) -> Result<()> {
        self.index_directories_with_progress(&[path.to_path_buf()], show)
    }

    pub fn load_or_create_index(&mut self, show_progress: bool) -> Result<()> {
        let path = self.index_path();
        if !path.exists() {
            if show_progress {
                println!("No filesystem index found. A new index will be created.");
            }
            return Ok(());
        }

        if show_progress {
            println!("Loading filesystem index from {}", path.display());
        }
        self.load_from_file(&path)
    }

    pub fn persist_index(&mut self, show_progress: bool) -> Result<()> {
        if !self.is_dirty {
            return Ok(());
        }
        if show_progress {
            println!("Saving index to disk...");
        }
        let path = self.index_path();
        self.save_to_file(&path)
    }

    pub fn refresh_index(&mut self, show_progress: bool) -> Result<()> {
        if self.root_paths.is_empty() {
            bail!("no tracked paths found in index");
        }

        let paths = self
            .root_paths
            .iter()
            .filter(|p| Path::new(p).exists())
            .map(|p| PathBuf::from(p))
            .collect::<Vec<_>>();

        if paths.is_empty() {
            bail!("no valid tracked paths found");
        }

        if show_progress {
            println!("🔄 Re-indexing {} tracked paths...", paths.len());
        }

        self.index_directories_with_progress(&paths, show_progress)?;
        self.persist_index(show_progress)?;
        Ok(())
    }

    pub fn get_root_paths(&self) -> Vec<String> {
        self.root_paths.clone()
    }

    pub fn has_indexed_files(&self) -> bool {
        !self.path_records.is_empty()
    }

    pub fn entry_count(&self) -> usize {
        self.path_records.len()
    }

    #[allow(dead_code)]
    pub fn search_files(&self, query: &str, enable_fuzzy: bool) -> Vec<RankedFile> {
        let mut candidates = Vec::new();
        let query_lower = query.to_lowercase();

        for record in &self.path_records {
            let path = record.path();
            let base = Path::new(&path)
                .file_name()
                .and_then(|s| s.to_str())
                .unwrap_or("")
                .to_lowercase();
            let path_lower = path.to_lowercase();

            let matched = if enable_fuzzy {
                base.contains(&query_lower) || path_lower.contains(&query_lower)
            } else {
                base.starts_with(&query_lower)
            };

            if matched {
                candidates.push(path);
            }
        }

        let mut ranked = Vec::with_capacity(candidates.len());
        for path in candidates {
            if let Ok(metadata) = self.get_file_metadata(&path) {
                let score = self.calculate_score(&metadata);
                ranked.push(RankedFile {
                    path,
                    score,
                    metadata,
                });
            }
        }

        ranked.sort_by(|a, b| {
            b.score
                .partial_cmp(&a.score)
                .unwrap_or(std::cmp::Ordering::Equal)
        });
        if ranked.len() > 50 {
            ranked.truncate(50);
        }
        ranked
    }

    pub fn cleanup_index(&mut self, options: CleanupOptions) -> Result<CleanupStats> {
        let mut stats = CleanupStats {
            total_entries: self.path_records.len(),
            ..CleanupStats::default()
        };

        if self.path_records.is_empty() {
            return Ok(stats);
        }

        let bar = if options.show_progress {
            Some(
                ProgressBar::new(self.path_records.len() as u64).with_style(
                    ProgressStyle::with_template("{bar:40.cyan/blue} {pos}/{len} {msg}")
                        .unwrap_or_else(|_| ProgressStyle::default_bar()),
                ),
            )
        } else {
            None
        };

        let mut valid_records = Vec::with_capacity(self.path_records.len());
        let mut valid_paths = Vec::with_capacity(self.path_records.len());
        let threshold = if options.older_than_days > 0 {
            Some(Utc::now() - chrono::Duration::days(options.older_than_days as i64))
        } else {
            None
        };

        for record in &self.path_records {
            if let Some(bar) = bar.as_ref() {
                bar.inc(1);
            }

            let path = record.path();
            if let Some(prefix) = options.path.as_ref() {
                if !path.starts_with(prefix) {
                    valid_records.push(*record);
                    valid_paths.push(path);
                    continue;
                }
            }

            let mut remove = false;
            if options.remove_stale && !Path::new(&path).exists() {
                remove = true;
                stats.stale_files += 1;
            }

            if !remove {
                if let Some(threshold) = threshold {
                    if let Some(ts) = record.timestamp_option() {
                        if ts < threshold {
                            remove = true;
                            stats.old_files += 1;
                        }
                    }
                }
            }

            if !remove {
                valid_records.push(*record);
                valid_paths.push(path);
            } else {
                stats.removed_entries += 1;
            }
        }

        if let Some(bar) = bar {
            bar.finish_with_message("Cleanup completed");
        }

        if stats.removed_entries > 0 {
            self.rebuild_structures(valid_records, valid_paths);
            stats.freed_kb = (stats.removed_entries * PATH_RECORD_SIZE) as f64 / 1024.0;
        }

        Ok(stats)
    }

    pub fn clear_index(&mut self) {
        self.path_records.clear();
        self.path_index.clear();
        self.root_paths.clear();
        self.bloom_filter = BloomFilter::new(
            self.config.bloom_filter_size,
            self.config.bloom_filter_hashes,
        );
        self.count_min = CountMinSketch::new();
        self.is_dirty = true;
    }

    pub fn get_index_stats(&self) -> String {
        let record_bytes = self.path_records.len() * PATH_RECORD_SIZE;
        let sketch_bytes = COUNT_MIN_DEPTH * COUNT_MIN_WIDTH * 4;
        let bloom_bytes = self.bloom_filter.estimated_bytes();
        format!(
            "Index Stats: {} files, Memory: {:.2}KB (Records: {:.2}KB, Metadata: {:.2}KB)",
            self.path_records.len(),
            (record_bytes + sketch_bytes + bloom_bytes) as f64 / 1024.0,
            record_bytes as f64 / 1024.0,
            (sketch_bytes + bloom_bytes) as f64 / 1024.0,
        )
    }

    pub fn get_index_file_size(&self) -> Result<u64> {
        let path = self.index_path();
        Ok(fs::metadata(path).map(|m| m.len()).unwrap_or(0))
    }

    fn rebuild_structures(&mut self, records: Vec<PathRecord>, paths: Vec<String>) {
        self.path_records = records;
        self.path_index.clear();
        for (idx, path) in paths.iter().enumerate() {
            self.path_index.insert(path.clone(), idx);
        }

        self.bloom_filter = BloomFilter::new(
            self.config.bloom_filter_size,
            self.config.bloom_filter_hashes,
        );
        self.count_min = CountMinSketch::new();
        for path in paths {
            self.bloom_filter.add(path.as_bytes());
        }
        self.is_dirty = true;
    }

    #[allow(dead_code)]
    fn get_file_metadata(&self, path: &str) -> Result<FileMetadata> {
        let idx = *self
            .path_index
            .get(path)
            .ok_or_else(|| anyhow!("path not found in index"))?;
        let record = self.path_records[idx];

        let timestamp = record.timestamp_option();
        let metadata = fs::metadata(path).ok();
        let size = metadata.as_ref().map(|m| m.len());
        let last_modified = metadata
            .as_ref()
            .and_then(|meta| meta.modified().ok())
            .map(|st| DateTime::<Local>::from(st));

        Ok(FileMetadata {
            path: path.to_string(),
            timestamp,
            access_count: record.access_count,
            is_directory: (record.flags & FLAG_IS_DIRECTORY) != 0,
            is_hidden: (record.flags & FLAG_IS_HIDDEN) != 0,
            is_symlink: (record.flags & FLAG_IS_SYMLINK) != 0,
            size,
            last_modified,
        })
    }

    #[allow(dead_code)]
    fn calculate_score(&self, metadata: &FileMetadata) -> f64 {
        if metadata.timestamp.is_none() {
            return 0.0;
        }
        let now = Utc::now();
        let time_delta = now
            .signed_duration_since(metadata.timestamp.unwrap())
            .num_hours() as f64;
        let frequency_score = metadata.access_count as f64;
        let recency_score = 1.0 / (time_delta + 1.0);
        let mut score = (0.7 * frequency_score) + (0.3 * recency_score);
        if metadata.is_directory {
            score *= 0.8;
        }
        score
    }

    fn should_skip(&self, path: &Path) -> bool {
        let base = path.file_name().and_then(|s| s.to_str()).unwrap_or("");
        for pattern in &self.config.ignore_patterns {
            if wildcard_match(pattern, base) {
                return true;
            }
        }

        let path_str = path.to_string_lossy();
        for pattern in &self.config.ignore_patterns {
            if path_str.contains(pattern) {
                return true;
            }
        }
        false
    }

    fn add_root_path(&mut self, path: &Path) {
        let abs = fs::canonicalize(path).unwrap_or_else(|_| path.to_path_buf());
        let abs_str = abs.to_string_lossy().to_string();
        if !self.root_paths.iter().any(|p| p == &abs_str) {
            self.root_paths.push(abs_str);
            self.is_dirty = true;
        }
    }

    fn index_path(&self) -> PathBuf {
        let base_dirs = BaseDirs::new().expect("home directory missing");
        base_dirs.home_dir().join(".recaller_fs_index.bin")
    }

    fn save_to_file(&mut self, path: &Path) -> Result<()> {
        let mut file = File::create(path).context("failed to create filesystem index file")?;
        file.write_all(b"RECALLER")?;
        file.write_all(&2u32.to_le_bytes())?; // version
        file.write_all(&(self.path_records.len() as u32).to_le_bytes())?;
        file.write_all(&(self.root_paths.len() as u32).to_le_bytes())?;
        file.write_all(&[0u8; 12])?; // reserved

        for root in &self.root_paths {
            let bytes = root.as_bytes();
            file.write_all(&(bytes.len() as u32).to_le_bytes())?;
            file.write_all(bytes)?;
        }

        self.bloom_filter.write_to(&mut file)?;
        self.count_min.write_to(&mut file)?;

        for record in &self.path_records {
            record.write_to(&mut file)?;
        }

        self.is_dirty = false;
        Ok(())
    }

    fn load_from_file(&mut self, path: &Path) -> Result<()> {
        let mut file = File::open(path).context("failed to open filesystem index")?;
        let mut magic = [0u8; 8];
        file.read_exact(&mut magic)?;
        if &magic != b"RECALLER" {
            bail!("invalid filesystem index format");
        }

        let mut ver_buf = [0u8; 4];
        file.read_exact(&mut ver_buf)?;
        let version = u32::from_le_bytes(ver_buf);
        if version != 1 && version != 2 {
            bail!("unsupported filesystem index version: {version}");
        }

        let mut count_buf = [0u8; 4];
        file.read_exact(&mut count_buf)?;
        let record_count = u32::from_le_bytes(count_buf);

        let root_count = if version == 2 {
            let mut buf = [0u8; 4];
            file.read_exact(&mut buf)?;
            u32::from_le_bytes(buf)
        } else {
            let mut _bloom = [0u8; 4];
            file.read_exact(&mut _bloom)?;
            0
        };

        let mut reserved = [0u8; 12];
        file.read_exact(&mut reserved)?;

        self.root_paths.clear();
        for _ in 0..root_count {
            let mut len_buf = [0u8; 4];
            file.read_exact(&mut len_buf)?;
            let len = u32::from_le_bytes(len_buf) as usize;
            let mut buf = vec![0u8; len];
            file.read_exact(&mut buf)?;
            if let Ok(path) = String::from_utf8(buf) {
                self.root_paths.push(path);
            }
        }

        self.bloom_filter.read_from(&mut file)?;
        self.count_min.read_from(&mut file)?;

        self.path_records.clear();
        self.path_index.clear();
        for i in 0..record_count {
            let record = PathRecord::read_from(&mut file)?;
            let path = record.path();
            self.path_index.insert(path.clone(), i as usize);
            self.path_records.push(record);
        }

        self.is_dirty = false;
        Ok(())
    }
}

#[derive(Clone, Copy)]
struct PathRecord {
    path: [u8; MAX_PATH_LENGTH],
    timestamp: i64,
    access_count: i32,
    flags: u8,
}

impl PathRecord {
    fn new(path: &str, timestamp: i64, access_count: i32, flags: u8) -> Self {
        let mut buf = [0u8; MAX_PATH_LENGTH];
        let bytes = path.as_bytes();
        let len = bytes.len().min(MAX_PATH_LENGTH - 1);
        buf[..len].copy_from_slice(&bytes[..len]);
        Self {
            path: buf,
            timestamp,
            access_count,
            flags,
        }
    }

    fn path(&self) -> String {
        let end = self
            .path
            .iter()
            .position(|&b| b == 0)
            .unwrap_or(MAX_PATH_LENGTH);
        String::from_utf8_lossy(&self.path[..end]).to_string()
    }

    fn write_to<W: Write>(&self, mut writer: W) -> Result<()> {
        writer.write_all(&self.path)?;
        writer.write_all(&self.timestamp.to_le_bytes())?;
        writer.write_all(&self.access_count.to_le_bytes())?;
        writer.write_all(&[self.flags])?;
        Ok(())
    }

    fn read_from<R: Read>(mut reader: R) -> Result<Self> {
        let mut path = [0u8; MAX_PATH_LENGTH];
        reader.read_exact(&mut path)?;

        let mut ts_buf = [0u8; 8];
        reader.read_exact(&mut ts_buf)?;
        let timestamp = i64::from_le_bytes(ts_buf);

        let mut access_buf = [0u8; 4];
        reader.read_exact(&mut access_buf)?;
        let access_count = i32::from_le_bytes(access_buf);

        let mut flag = [0u8; 1];
        reader.read_exact(&mut flag)?;

        Ok(Self {
            path,
            timestamp,
            access_count,
            flags: flag[0],
        })
    }

    fn timestamp_option(&self) -> Option<DateTime<Utc>> {
        if self.timestamp > 0 {
            DateTime::<Utc>::from_timestamp(self.timestamp, 0)
        } else {
            None
        }
    }
}

struct CountMinSketch {
    table: [[i32; COUNT_MIN_WIDTH]; COUNT_MIN_DEPTH],
}

impl CountMinSketch {
    fn new() -> Self {
        Self {
            table: [[0; COUNT_MIN_WIDTH]; COUNT_MIN_DEPTH],
        }
    }

    fn hash(item: &str, row: usize) -> usize {
        let mut hasher = FnvHasher::with_seed(row as u64);
        hasher.write(item.as_bytes());
        (hasher.finish() as usize) % COUNT_MIN_WIDTH
    }

    fn add(&mut self, item: &str, count: i32) {
        for row in 0..COUNT_MIN_DEPTH {
            let idx = Self::hash(item, row);
            self.table[row][idx] = self.table[row][idx].saturating_add(count);
        }
    }

    fn estimate(&self, item: &str) -> i32 {
        let mut min = i32::MAX;
        for row in 0..COUNT_MIN_DEPTH {
            let idx = Self::hash(item, row);
            min = min.min(self.table[row][idx]);
        }
        if min == i32::MAX { 0 } else { min }
    }

    fn write_to<W: Write>(&self, mut writer: W) -> Result<()> {
        for row in 0..COUNT_MIN_DEPTH {
            for col in 0..COUNT_MIN_WIDTH {
                writer.write_all(&self.table[row][col].to_le_bytes())?;
            }
        }
        Ok(())
    }

    fn read_from<R: Read>(&mut self, mut reader: R) -> Result<()> {
        for row in 0..COUNT_MIN_DEPTH {
            for col in 0..COUNT_MIN_WIDTH {
                let mut buf = [0u8; 4];
                reader.read_exact(&mut buf)?;
                self.table[row][col] = i32::from_le_bytes(buf);
            }
        }
        Ok(())
    }
}

struct BloomFilter {
    m: u64,
    k: u64,
    bitset: BitSet,
}

impl BloomFilter {
    fn new(m: u32, k: u32) -> Self {
        let m = m.max(1) as u64;
        let k = k.max(1) as u64;
        Self {
            m,
            k,
            bitset: BitSet::new(m),
        }
    }

    fn add(&mut self, data: &[u8]) {
        let hashes = base_hashes(data);
        for i in 0..self.k {
            let idx = location(&hashes, i) % self.m;
            self.bitset.set(idx);
        }
    }

    fn read_from<R: Read>(&mut self, mut reader: R) -> Result<()> {
        let mut m_buf = [0u8; 8];
        reader.read_exact(&mut m_buf)?;
        self.m = u64::from_be_bytes(m_buf);
        let mut k_buf = [0u8; 8];
        reader.read_exact(&mut k_buf)?;
        self.k = u64::from_be_bytes(k_buf);
        self.bitset.read_from(&mut reader)?;
        Ok(())
    }

    fn write_to<W: Write>(&self, mut writer: W) -> Result<()> {
        writer.write_all(&self.m.to_be_bytes())?;
        writer.write_all(&self.k.to_be_bytes())?;
        self.bitset.write_to(&mut writer)?;
        Ok(())
    }

    fn estimated_bytes(&self) -> usize {
        (self.bitset.data.len() * 8) + 16
    }
}

#[derive(Clone)]
struct BitSet {
    length: u64,
    data: Vec<u64>,
}

impl BitSet {
    fn new(length: u64) -> Self {
        let words = ((length + 63) / 64) as usize;
        Self {
            length,
            data: vec![0; words],
        }
    }

    fn set(&mut self, idx: u64) {
        let word = (idx / 64) as usize;
        let bit = idx % 64;
        if word < self.data.len() {
            self.data[word] |= 1u64 << bit;
        }
    }

    fn write_to<W: Write>(&self, mut writer: W) -> Result<()> {
        writer.write_all(&self.length.to_be_bytes())?;
        for &value in &self.data {
            writer.write_all(&value.to_be_bytes())?;
        }
        Ok(())
    }

    fn read_from<R: Read>(&mut self, mut reader: R) -> Result<()> {
        let mut len_buf = [0u8; 8];
        reader.read_exact(&mut len_buf)?;
        self.length = u64::from_be_bytes(len_buf);
        let words = ((self.length + 63) / 64) as usize;
        self.data = vec![0; words];
        for i in 0..words {
            let mut buf = [0u8; 8];
            reader.read_exact(&mut buf)?;
            self.data[i] = u64::from_be_bytes(buf);
        }
        Ok(())
    }
}

#[derive(Default)]
struct FnvHasher {
    hash: u64,
}

impl FnvHasher {
    fn with_seed(seed: u64) -> Self {
        let mut hasher = Self {
            hash: 0xcbf29ce484222325,
        };
        hasher.write(&seed.to_le_bytes());
        hasher
    }

    fn write(&mut self, bytes: &[u8]) {
        for &b in bytes {
            self.hash ^= b as u64;
            self.hash = self.hash.wrapping_mul(0x100000001b3);
        }
    }

    fn finish(&self) -> u64 {
        self.hash
    }
}

fn base_hashes(data: &[u8]) -> [u64; 4] {
    let mut hasher = Murmur3::new();
    hasher.write(data);
    let (v1, v2) = hasher.sum128();
    hasher.write(&[1]);
    let (v3, v4) = hasher.sum128();
    [v1, v2, v3, v4]
}

fn location(h: &[u64; 4], i: u64) -> u64 {
    let ii = i;
    let idx = 2 + (((ii + (ii % 2)) % 4) / 2) as usize;
    h[(ii % 2) as usize].wrapping_add(ii.wrapping_mul(h[idx]))
}

struct Murmur3 {
    buffer: Vec<u8>,
}

impl Murmur3 {
    fn new() -> Self {
        Self { buffer: Vec::new() }
    }

    fn write(&mut self, data: &[u8]) {
        self.buffer.extend_from_slice(data);
    }

    fn sum128(&self) -> (u64, u64) {
        murmur3_x64_128(&self.buffer)
    }
}

fn murmur3_x64_128(data: &[u8]) -> (u64, u64) {
    const C1: u64 = 0x87c37b91114253d5;
    const C2: u64 = 0x4cf5ad432745937f;

    let mut h1 = 0u64;
    let mut h2 = 0u64;
    let mut chunks = data.chunks_exact(16);

    for chunk in &mut chunks {
        let mut k1 = u64::from_le_bytes(chunk[0..8].try_into().unwrap());
        let mut k2 = u64::from_le_bytes(chunk[8..16].try_into().unwrap());

        k1 = k1.wrapping_mul(C1);
        k1 = k1.rotate_left(31);
        k1 = k1.wrapping_mul(C2);
        h1 ^= k1;

        h1 = h1.rotate_left(27);
        h1 = h1.wrapping_add(h2);
        h1 = h1.wrapping_mul(5).wrapping_add(0x52dce729);

        k2 = k2.wrapping_mul(C2);
        k2 = k2.rotate_left(33);
        k2 = k2.wrapping_mul(C1);
        h2 ^= k2;

        h2 = h2.rotate_left(31);
        h2 = h2.wrapping_add(h1);
        h2 = h2.wrapping_mul(5).wrapping_add(0x38495ab5);
    }

    let tail = chunks.remainder();
    let mut k1 = 0u64;
    let mut k2 = 0u64;
    let tail_len = tail.len() & 15;

    if tail_len >= 15 {
        k2 ^= (tail[14] as u64) << 48;
    }
    if tail_len >= 14 {
        k2 ^= (tail[13] as u64) << 40;
    }
    if tail_len >= 13 {
        k2 ^= (tail[12] as u64) << 32;
    }
    if tail_len >= 12 {
        k2 ^= (tail[11] as u64) << 24;
    }
    if tail_len >= 11 {
        k2 ^= (tail[10] as u64) << 16;
    }
    if tail_len >= 10 {
        k2 ^= (tail[9] as u64) << 8;
    }
    if tail_len >= 9 {
        k2 ^= tail[8] as u64;
        k2 = k2.wrapping_mul(C2);
        k2 = k2.rotate_left(33);
        k2 = k2.wrapping_mul(C1);
        h2 ^= k2;
    }

    if tail_len >= 8 {
        k1 ^= (tail[7] as u64) << 56;
    }
    if tail_len >= 7 {
        k1 ^= (tail[6] as u64) << 48;
    }
    if tail_len >= 6 {
        k1 ^= (tail[5] as u64) << 40;
    }
    if tail_len >= 5 {
        k1 ^= (tail[4] as u64) << 32;
    }
    if tail_len >= 4 {
        k1 ^= (tail[3] as u64) << 24;
    }
    if tail_len >= 3 {
        k1 ^= (tail[2] as u64) << 16;
    }
    if tail_len >= 2 {
        k1 ^= (tail[1] as u64) << 8;
    }
    if tail_len >= 1 {
        k1 ^= tail[0] as u64;
        k1 = k1.wrapping_mul(C1);
        k1 = k1.rotate_left(31);
        k1 = k1.wrapping_mul(C2);
        h1 ^= k1;
    }

    h1 ^= data.len() as u64;
    h2 ^= data.len() as u64;

    h1 = h1.wrapping_add(h2);
    h2 = h2.wrapping_add(h1);

    h1 = fmix64(h1);
    h2 = fmix64(h2);

    h1 = h1.wrapping_add(h2);
    h2 = h2.wrapping_add(h1);

    (h1, h2)
}

fn fmix64(mut k: u64) -> u64 {
    k ^= k >> 33;
    k = k.wrapping_mul(0xff51afd7ed558ccd);
    k ^= k >> 33;
    k = k.wrapping_mul(0xc4ceb9fe1a85ec53);
    k ^= k >> 33;
    k
}

fn wildcard_match(pattern: &str, value: &str) -> bool {
    if !pattern.contains('*') && !pattern.contains('?') {
        return pattern == value;
    }
    let (p, t) = (pattern.as_bytes(), value.as_bytes());
    let (mut i, mut j, mut star_idx, mut match_idx) = (0usize, 0usize, None, 0usize);

    while j < t.len() {
        if i < p.len() && (p[i] == b'?' || p[i] == t[j]) {
            i += 1;
            j += 1;
        } else if i < p.len() && p[i] == b'*' {
            star_idx = Some(i);
            match_idx = j;
            i += 1;
        } else if let Some(star) = star_idx {
            i = star + 1;
            match_idx += 1;
            j = match_idx;
        } else {
            return false;
        }
    }

    while i < p.len() && p[i] == b'*' {
        i += 1;
    }

    i == p.len()
}

fn file_flags(metadata: &fs::Metadata, path: &Path) -> u8 {
    let mut flags = 0;
    if metadata.is_dir() {
        flags |= FLAG_IS_DIRECTORY;
    }
    if path
        .file_name()
        .and_then(|s| s.to_str())
        .map(|s| s.starts_with('.'))
        .unwrap_or(false)
    {
        flags |= FLAG_IS_HIDDEN;
    }
    if metadata.file_type().is_symlink() {
        flags |= FLAG_IS_SYMLINK;
    }
    flags
}
