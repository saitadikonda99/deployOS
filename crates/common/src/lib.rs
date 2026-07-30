//! Shared types and utilities used across DeployOS Rust crates.

pub mod version;

use thiserror::Error;

/// Errors shared across DeployOS Rust components.
#[derive(Debug, Error)]
pub enum DeployOsError {
    #[error("configuration error: {0}")]
    Config(String),

    #[error("i/o error: {0}")]
    Io(#[from] std::io::Error),
}

pub type Result<T> = std::result::Result<T, DeployOsError>;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn config_error_formats_message() {
        let err = DeployOsError::Config("missing field".to_string());
        assert_eq!(err.to_string(), "configuration error: missing field");
    }
}
