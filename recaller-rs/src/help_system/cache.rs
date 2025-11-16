use std::time::{Duration, Instant};

use dashmap::DashMap;

const HELP_CACHE_EXPIRATION: Duration = Duration::from_secs(30 * 60);

struct CacheEntry {
    value: String,
    expires_at: Instant,
}

impl CacheEntry {
    fn new(value: String) -> Self {
        Self {
            value,
            expires_at: Instant::now() + HELP_CACHE_EXPIRATION,
        }
    }

    fn is_expired(&self) -> bool {
        Instant::now() >= self.expires_at
    }
}

#[derive(Default)]
pub struct HelpCache {
    entries: DashMap<String, CacheEntry>,
}

impl HelpCache {
    pub fn new() -> Self {
        Self {
            entries: DashMap::new(),
        }
    }

    pub fn get(&self, key: &str) -> Option<String> {
        if let Some(entry) = self.entries.get(key) {
            if entry.is_expired() {
                drop(entry);
                self.entries.remove(key);
                return None;
            }
            return Some(entry.value.clone());
        }
        None
    }

    pub fn insert(&self, key: impl Into<String>, value: impl Into<String>) {
        let entry = CacheEntry::new(value.into());
        self.entries.insert(key.into(), entry);
    }
}
