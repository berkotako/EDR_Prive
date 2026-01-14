// Integration tests for DLP engine
// Tests realistic scenarios with sensitive data patterns

// Note: These tests will work after the agent is properly configured as a library
// For now, they serve as specifications for the DLP engine behavior

#[test]
fn test_social_security_number_detection() {
    let engine = DlpEngine::new();

    // Create a test SSN pattern (simplified for testing)
    let ssn_data = b"SSN: 123-45-6789 for John Doe, address 123 Main St";
    let chunk_hash = engine.hash_chunk(&ssn_data[5..5+64.min(ssn_data.len()-5)]);

    engine.add_fingerprint(&chunk_hash, "SSN-US".to_string(), Severity::Critical);

    // Scan buffer containing the SSN
    let matches = engine.scan_buffer(ssn_data);

    // Should detect the fingerprint
    assert!(!matches.is_empty(), "Should detect SSN pattern");
    if !matches.is_empty() {
        assert_eq!(matches[0].rule_id, "SSN-US");
        assert_eq!(matches[0].severity, Severity::Critical);
    }
}

#[test]
fn test_credit_card_number_detection() {
    let engine = DlpEngine::new();

    // Create a test credit card pattern
    let ccn_data = b"Payment Card: 4532-1234-5678-9010 Exp: 12/25 CVV: 123 Name: Jane Smith";
    let chunk_hash = engine.hash_chunk(&ccn_data[14..14+64.min(ccn_data.len()-14)]);

    engine.add_fingerprint(&chunk_hash, "CCN-VISA".to_string(), Severity::Critical);

    // Scan buffer
    let matches = engine.scan_buffer(ccn_data);

    assert!(!matches.is_empty(), "Should detect credit card pattern");
    if !matches.is_empty() {
        assert_eq!(matches[0].rule_id, "CCN-VISA");
    }
}

#[test]
fn test_no_false_positives_on_benign_data() {
    let engine = DlpEngine::new();

    // Add some fingerprints
    engine.add_fingerprint("abc123", "TEST_RULE".to_string(), Severity::High);

    // Scan benign data
    let benign = b"This is a normal business document with no sensitive information whatsoever.";
    let matches = engine.scan_buffer(benign);

    assert_eq!(matches.len(), 0, "Should not have false positives on benign data");
}

#[test]
fn test_large_buffer_performance() {
    let engine = DlpEngine::new();

    // Create a large buffer (1MB)
    let large_buffer = vec![b'X'; 1024 * 1024];

    let start = std::time::Instant::now();
    let matches = engine.scan_buffer(&large_buffer);
    let duration = start.elapsed();

    // Should complete in reasonable time (<100ms for 1MB)
    assert!(duration.as_millis() < 100, "Scanning 1MB should be fast: {:?}", duration);
    assert_eq!(matches.len(), 0);
}

#[test]
fn test_multiple_patterns_in_buffer() {
    let engine = DlpEngine::new();

    // Create two distinct patterns
    let pattern1 = b"SENSITIVE_DATA_PATTERN_ALPHA_XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX";
    let pattern2 = b"SENSITIVE_DATA_PATTERN_BETA_XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX";

    let hash1 = engine.hash_chunk(&pattern1[..64]);
    let hash2 = engine.hash_chunk(&pattern2[..64]);

    engine.add_fingerprint(&hash1, "PATTERN_ALPHA".to_string(), Severity::High);
    engine.add_fingerprint(&hash2, "PATTERN_BETA".to_string(), Severity::Critical);

    // Create buffer with both patterns
    let mut combined = Vec::new();
    combined.extend_from_slice(b"Some header text here... ");
    combined.extend_from_slice(pattern1);
    combined.extend_from_slice(b" middle content ");
    combined.extend_from_slice(pattern2);
    combined.extend_from_slice(b" footer text");

    let matches = engine.scan_buffer(&combined);

    // Should detect both patterns
    assert!(matches.len() >= 2, "Should detect multiple patterns");

    let rule_ids: Vec<String> = matches.iter().map(|m| m.rule_id.clone()).collect();
    assert!(rule_ids.contains(&"PATTERN_ALPHA".to_string()));
    assert!(rule_ids.contains(&"PATTERN_BETA".to_string()));
}

#[test]
fn test_should_scan_filters() {
    let engine = DlpEngine::new();

    // Too small buffer
    let tiny = b"hi";
    assert!(!engine.should_scan(tiny), "Should skip tiny buffers");

    // All zeros
    let zeros = vec![0u8; 256];
    assert!(!engine.should_scan(&zeros), "Should skip zero-filled buffers");

    // All spaces
    let spaces = vec![b' '; 256];
    assert!(!engine.should_scan(&spaces), "Should skip whitespace-only buffers");

    // Valid buffer
    let valid = b"This is a normal text document with actual content that should be analyzed for sensitive data patterns.";
    assert!(engine.should_scan(valid), "Should scan valid buffers");
}

#[test]
fn test_overlapping_pattern_detection() {
    let engine = DlpEngine::new();

    // Create a repeating pattern that spans chunk boundaries
    let pattern = b"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";
    let mut buffer = Vec::new();

    // Repeat pattern to span multiple chunks
    for _ in 0..10 {
        buffer.extend_from_slice(pattern);
    }

    // Add fingerprint for the pattern
    let hash = engine.hash_chunk(&buffer[..64]);
    engine.add_fingerprint(&hash, "REPEATING_PATTERN".to_string(), Severity::Medium);

    let matches = engine.scan_buffer(&buffer);

    // With overlapping windows, should detect the pattern multiple times
    assert!(!matches.is_empty(), "Should detect pattern with overlapping scan");
}

#[test]
fn test_hash_consistency() {
    let engine = DlpEngine::new();

    // Same data should produce same hash
    let data = b"Consistent hashing test data for DLP fingerprinting with BLAKE3";

    let hash1 = engine.hash_chunk(data);
    let hash2 = engine.hash_chunk(data);

    assert_eq!(hash1, hash2, "Hash should be deterministic");

    // Different data should produce different hash
    let different = b"Different data should produce a completely different hash value";
    let hash3 = engine.hash_chunk(different);

    assert_ne!(hash1, hash3, "Different data should have different hashes");
}
