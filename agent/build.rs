// Build script to compile Protocol Buffer definitions

fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Compile telemetry.proto
    tonic_build::configure()
        .build_server(false) // Agent is client-only
        .build_client(true)
        .out_dir("src/generated")
        .compile(
            &["../proto/telemetry.proto"],
            &["../proto"],
        )?;

    println!("cargo:rerun-if-changed=../proto/telemetry.proto");

    Ok(())
}
