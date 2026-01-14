// Data Loss Prevention (DLP) Engine
// Implements Exact Data Match (EDM) using cryptographic fingerprinting.
// Performance-critical: Must scan buffers with minimal latency.

use anyhow::Result;
use dashmap::DashMap;
use sha2::{Sha256, Digest};
use blake3::Hasher as Blake3Hasher;
use std::sync::Arc;
use tracing::{debug, warn};

/// Chunk size for rolling hash fingerprinting (in bytes).
/// Smaller chunks = more granular detection but higher memory usage.
const CHUNK_SIZE: usize = 64;

/// Minimum buffer size to trigger DLP scanning (avoid overhead on tiny buffers).
const MIN_SCAN_SIZE: usize = 128;

/// DLP match severity levels.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Severity {
    Low = 1,
    Medium = 2,
    High = 3,
    Critical = 4,
}

/// Represents a matched sensitive data pattern.
#[derive(Debug, Clone)]
pub struct DlpMatch {
    pub rule_id: String,
    pub severity: Severity,
    pub matched_hash: String,
    pub offset: usize,
}

/// High-performance DLP engine using Exact Data Match (EDM).
/// Uses a hashset of cryptographic fingerprints for O(1) lookups.
pub struct DlpEngine {
    /// Fingerprint hashset: maps BLAKE3 hash -> (rule_id, severity)
    /// DashMap provides concurrent access without locks for read-heavy workloads.
    fingerprints: Arc<DashMap<String, (String, Severity)>>,

    /// Toggle for different hashing algorithms (BLAKE3 is faster than SHA-256).
    use_blake3: bool,
}

impl DlpEngine {
    /// Create a new DLP engine with an empty fingerprint database.
    pub fn new() -> Self {
        Self {
            fingerprints: Arc::new(DashMap::new()),
            use_blake3: true, // BLAKE3 is faster and suitable for EDM
        }
    }

    /// Load fingerprints from a policy file (not implemented in skeleton).
    /// In production, this would:
    /// 1. Fetch policy from the management server (gRPC or REST API)
    /// 2. Parse JSON/protobuf containing sensitive data hashes
    /// 3. Populate the fingerprints DashMap
    pub fn load_fingerprints_from_policy(&self, _policy_path: &str) -> Result<()> {
        // TODO: Implement policy loading
        // Example format:
        // {
        //   "rules": [
        //     {"id": "SSN-US", "severity": "high", "hashes": ["abc123...", "def456..."]},
        //     {"id": "CCN-VISA", "severity": "critical", "hashes": [...]}
        //   ]
        // }

        // For testing, add dummy fingerprints
        self.add_fingerprint(
            "deadbeef0000",
            "TEST_SSN".to_string(),
            Severity::High,
        );
        self.add_fingerprint(
            "cafebabe0000",
            "TEST_CCN".to_string(),
            Severity::Critical,
        );

        Ok(())
    }

    /// Add a single fingerprint to the detection database.
    pub fn add_fingerprint(&self, hash: &str, rule_id: String, severity: Severity) {
        self.fingerprints.insert(hash.to_string(), (rule_id, severity));
    }

    /// Return the number of loaded fingerprints.
    pub fn fingerprint_count(&self) -> usize {
        self.fingerprints.len()
    }

    /// Scan a buffer for sensitive data using rolling hash fingerprinting.
    /// This is the core performance-critical function.
    ///
    /// Algorithm:
    /// 1. Divide buffer into overlapping chunks of CHUNK_SIZE bytes
    /// 2. Hash each chunk using BLAKE3 (or SHA-256 fallback)
    /// 3. Check if hash exists in fingerprint database
    /// 4. Return matches with offset and severity
    pub fn scan_buffer(&self, buffer: &[u8]) -> Vec<DlpMatch> {
        if buffer.len() < MIN_SCAN_SIZE {
            return Vec::new();
        }

        let mut matches = Vec::new();

        // Rolling window scan with configurable overlap
        // Overlap ensures we catch patterns that span chunk boundaries
        let overlap = CHUNK_SIZE / 2;
        let mut offset = 0;

        while offset + CHUNK_SIZE <= buffer.len() {
            let chunk = &buffer[offset..offset + CHUNK_SIZE];
            let chunk_hash = self.hash_chunk(chunk);

            // O(1) lookup in concurrent hashmap
            if let Some(entry) = self.fingerprints.get(&chunk_hash) {
                let (rule_id, severity) = entry.value();

                debug!(
                    "DLP match: rule={}, severity={:?}, offset={}",
                    rule_id, severity, offset
                );

                matches.push(DlpMatch {
                    rule_id: rule_id.clone(),
                    severity: *severity,
                    matched_hash: chunk_hash,
                    offset,
                });
            }

            offset += overlap;
        }

        if !matches.is_empty() {
            warn!(
                "DLP scan detected {} sensitive data match(es) in {} byte buffer",
                matches.len(),
                buffer.len()
            );
        }

        matches
    }

    /// Hash a chunk using the configured algorithm.
    /// BLAKE3 is preferred for speed; SHA-256 is fallback for compliance.
    fn hash_chunk(&self, chunk: &[u8]) -> String {
        if self.use_blake3 {
            let mut hasher = Blake3Hasher::new();
            hasher.update(chunk);
            let hash = hasher.finalize();
            hex::encode(hash.as_bytes())
        } else {
            let mut hasher = Sha256::new();
            hasher.update(chunk);
            let hash = hasher.finalize();
            hex::encode(hash)
        }
    }

    /// Scan a file path for sensitive data (convenience wrapper).
    /// Reads file in chunks to avoid loading large files into memory.
    pub fn scan_file(&self, _file_path: &str) -> Result<Vec<DlpMatch>> {
        // TODO: Implement chunked file reading
        // 1. Open file with std::fs::File
        // 2. Read in CHUNK_SIZE * 1024 blocks
        // 3. Call scan_buffer on each block
        // 4. Aggregate matches

        Ok(Vec::new())
    }

    /// Fast path: scan only if buffer contains patterns of interest.
    /// Uses heuristics to skip scanning benign data (e.g., all zeros, small buffers).
    pub fn should_scan(&self, buffer: &[u8]) -> bool {
        if buffer.len() < MIN_SCAN_SIZE {
            return false;
        }

        // Skip buffers that are all zeros or whitespace (common in padding)
        let non_zero_bytes = buffer.iter().filter(|&&b| b != 0 && b != b' ').count();
        if non_zero_bytes < MIN_SCAN_SIZE / 2 {
            return false;
        }

        true
    }
}

impl Default for DlpEngine {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_dlp_engine_creation() {
        let engine = DlpEngine::new();
        assert_eq!(engine.fingerprint_count(), 0);
    }

    #[test]
    fn test_add_fingerprint() {
        let engine = DlpEngine::new();
        engine.add_fingerprint("testhash123", "TEST_RULE".to_string(), Severity::High);
        assert_eq!(engine.fingerprint_count(), 1);
    }

    #[test]
    fn test_scan_buffer_no_match() {
        let engine = DlpEngine::new();
        let buffer = b"This is a test buffer with no sensitive data.";
        let matches = engine.scan_buffer(buffer);
        assert_eq!(matches.len(), 0);
    }

    #[test]
    fn test_scan_buffer_with_match() {
        let engine = DlpEngine::new();

        // Create a test pattern
        let test_data = b"SENSITIVE_DATA_CHUNK_XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX";
        let chunk_hash = engine.hash_chunk(&test_data[..CHUNK_SIZE]);

        engine.add_fingerprint(&chunk_hash, "TEST_RULE".to_string(), Severity::Critical);

        // Scan buffer containing the pattern
        let matches = engine.scan_buffer(test_data);
        assert!(!matches.is_empty());
        assert_eq!(matches[0].rule_id, "TEST_RULE");
        assert_eq!(matches[0].severity, Severity::Critical);
    }

    #[test]
    fn test_should_scan_filters() {
        let engine = DlpEngine::new();

        // Too small
        assert!(!engine.should_scan(b"tiny"));

        // All zeros
        let zeros = vec![0u8; 256];
        assert!(!engine.should_scan(&zeros));

        // Valid buffer
        let valid = b"This is a normal buffer with actual content that should be scanned for sensitive information.";
        assert!(engine.should_scan(valid));
    }
}
