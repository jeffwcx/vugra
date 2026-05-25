//! Renderer-neutral system capabilities for Rust Vugra hosts.

use std::fmt;
use std::fs;
use std::io;
use std::path::{Path, PathBuf};
use std::time::SystemTime;

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Entry {
    pub name: String,
    pub path: String,
    pub kind: String,
    pub size: u64,
    pub modified_at: Option<SystemTime>,
}

pub trait FileSystem {
    fn read_dir(&self, path: &str) -> Result<Vec<Entry>, String>;
    fn stat(&self, path: &str) -> Result<Entry, FsError>;
    fn mkdir(&mut self, path: &str) -> Result<(), String>;
    fn rename(&mut self, old_path: &str, new_path: &str) -> Result<(), String>;
    fn remove(&mut self, path: &str) -> Result<(), String>;
    fn duplicate(&mut self, source: &str, target: &str) -> Result<(), String>;
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub enum FsError {
    NotExist(String),
    Other(String),
}

impl FsError {
    pub fn is_not_exist(&self) -> bool {
        matches!(self, Self::NotExist(_))
    }
}

impl fmt::Display for FsError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::NotExist(message) | Self::Other(message) => formatter.write_str(message),
        }
    }
}

impl From<io::Error> for FsError {
    fn from(error: io::Error) -> Self {
        if error.kind() == io::ErrorKind::NotFound {
            Self::NotExist(error.to_string())
        } else {
            Self::Other(error.to_string())
        }
    }
}

#[derive(Default)]
pub struct OsFileSystem;

impl FileSystem for OsFileSystem {
    fn read_dir(&self, path: &str) -> Result<Vec<Entry>, String> {
        let entries = fs::read_dir(path).map_err(|err| err.to_string())?;
        let mut rows = Vec::new();
        for entry in entries {
            let entry = entry.map_err(|err| err.to_string())?;
            let metadata = entry.metadata().map_err(|err| err.to_string())?;
            rows.push(entry_from_metadata(
                entry.file_name().to_string_lossy().as_ref(),
                &entry.path(),
                &metadata,
            ));
        }
        rows.sort_by(|left, right| {
            if left.kind != right.kind {
                return if left.kind == "folder" {
                    std::cmp::Ordering::Less
                } else {
                    std::cmp::Ordering::Greater
                };
            }
            left.name.to_lowercase().cmp(&right.name.to_lowercase())
        });
        Ok(rows)
    }

    fn stat(&self, path: &str) -> Result<Entry, FsError> {
        let metadata = fs::metadata(path).map_err(FsError::from)?;
        let name = Path::new(path)
            .file_name()
            .map(|name| name.to_string_lossy().to_string())
            .filter(|name| !name.is_empty())
            .unwrap_or_else(|| clean_path_str(path));
        Ok(entry_from_metadata(&name, Path::new(path), &metadata))
    }

    fn mkdir(&mut self, path: &str) -> Result<(), String> {
        fs::create_dir(path).map_err(|err| err.to_string())
    }

    fn rename(&mut self, old_path: &str, new_path: &str) -> Result<(), String> {
        fs::rename(old_path, new_path).map_err(|err| err.to_string())
    }

    fn remove(&mut self, path: &str) -> Result<(), String> {
        let metadata = fs::metadata(path).map_err(|err| err.to_string())?;
        if metadata.is_dir() {
            fs::remove_dir_all(path).map_err(|err| err.to_string())
        } else {
            fs::remove_file(path).map_err(|err| err.to_string())
        }
    }

    fn duplicate(&mut self, source: &str, target: &str) -> Result<(), String> {
        let metadata = fs::metadata(source).map_err(|err| err.to_string())?;
        if metadata.is_dir() {
            copy_dir(source, target)
        } else {
            fs::copy(source, target)
                .map(|_| ())
                .map_err(|err| err.to_string())
        }
    }
}

fn entry_from_metadata(name: &str, path: &Path, metadata: &fs::Metadata) -> Entry {
    Entry {
        name: name.to_string(),
        path: clean_path(path),
        kind: if metadata.is_dir() {
            "folder".to_string()
        } else {
            "file".to_string()
        },
        size: metadata.len(),
        modified_at: metadata.modified().ok(),
    }
}

pub fn clean_path(path: &Path) -> String {
    clean_path_str(&path.to_string_lossy())
}

pub fn clean_path_str(path: &str) -> String {
    let raw = if path.trim().is_empty() { "." } else { path };
    let mut parts = Vec::new();
    let absolute = raw.starts_with('/');
    for part in Path::new(raw).components() {
        match part {
            std::path::Component::RootDir => {}
            std::path::Component::CurDir => {}
            std::path::Component::ParentDir => {
                parts.pop();
            }
            std::path::Component::Normal(value) => parts.push(value.to_string_lossy().to_string()),
            _ => {}
        }
    }
    let body = parts.join("/");
    match (absolute, body.is_empty()) {
        (true, true) => "/".to_string(),
        (true, false) => format!("/{body}"),
        (false, true) => ".".to_string(),
        (false, false) => body,
    }
}

pub fn join_clean(parent: &str, child: &str) -> String {
    let mut path = PathBuf::from(parent);
    path.push(child);
    clean_path(&path)
}

pub fn sibling_path(source: &str, name: &str) -> String {
    let parent = Path::new(source)
        .parent()
        .and_then(Path::to_str)
        .unwrap_or(".");
    join_clean(parent, name)
}

pub fn split_parent_name(path: &str) -> Option<(String, String)> {
    let path = clean_path_str(path);
    let (parent, name) = path.rsplit_once('/').unwrap_or((".", path.as_str()));
    (!name.is_empty()).then(|| (clean_path_str(parent), name.to_string()))
}

fn copy_dir(source: &str, target: &str) -> Result<(), String> {
    let source_clean = std::fs::canonicalize(source).map_err(|err| err.to_string())?;
    let target_clean = absolute_path_for_new_target(target)?;
    if target_clean == source_clean || target_clean.starts_with(&source_clean) {
        return Err(format!(
            "duplicate {source:?}: destination cannot be inside source"
        ));
    }
    fs::create_dir(target).map_err(|err| err.to_string())?;
    for entry in fs::read_dir(source).map_err(|err| err.to_string())? {
        let entry = entry.map_err(|err| err.to_string())?;
        let source_path = entry.path();
        let target_path = Path::new(target).join(entry.file_name());
        let metadata = entry.metadata().map_err(|err| err.to_string())?;
        if metadata.is_dir() {
            copy_dir(&clean_path(&source_path), &clean_path(&target_path))?;
        } else {
            fs::copy(&source_path, &target_path)
                .map(|_| ())
                .map_err(|err| err.to_string())?;
        }
    }
    Ok(())
}

fn absolute_path_for_new_target(path: &str) -> Result<PathBuf, String> {
    let path = PathBuf::from(path);
    let absolute = if path.is_absolute() {
        path
    } else {
        std::env::current_dir()
            .map(|cwd| cwd.join(path))
            .map_err(|err| err.to_string())?
    };
    let Some(parent) = absolute.parent() else {
        return Ok(absolute);
    };
    let parent = std::fs::canonicalize(parent).unwrap_or_else(|_| parent.to_path_buf());
    Ok(parent.join(
        absolute
            .file_name()
            .map(|name| name.to_owned())
            .unwrap_or_default(),
    ))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn clean_path_preserves_absolute_roots_and_normalizes_relative_segments() {
        assert_eq!(clean_path_str("/tmp/../var/./docs"), "/var/docs");
        assert_eq!(
            clean_path_str("Documents/./Design/../Roadmap.md"),
            "Documents/Roadmap.md"
        );
        assert_eq!(clean_path_str(""), ".");
    }

    #[test]
    fn os_file_system_reports_not_exist_as_structured_error() {
        let err = OsFileSystem
            .stat("/definitely/missing/vugra-file")
            .expect_err("missing path should fail");
        assert!(err.is_not_exist());
    }

    #[test]
    fn os_file_system_rejects_duplicate_directory_into_itself() {
        let root = std::env::temp_dir().join(format!("vugra-system-copy-{}", std::process::id()));
        let source = root.join("source");
        let nested = source.join("nested");
        fs::create_dir_all(&source).expect("create source");
        let mut files = OsFileSystem;
        let err = files
            .duplicate(
                source.to_string_lossy().as_ref(),
                nested.to_string_lossy().as_ref(),
            )
            .expect_err("nested duplicate should fail");
        assert!(err.contains("destination cannot be inside source"));
        let _ = fs::remove_dir_all(root);
    }

    #[test]
    fn os_file_system_duplicate_directory_preserves_nested_children() {
        let root =
            std::env::temp_dir().join(format!("vugra-system-copy-nested-{}", std::process::id()));
        let source = root.join("source");
        let nested = source.join("nested");
        let target = root.join("source copy");
        fs::create_dir_all(&nested).expect("create nested source");
        fs::write(nested.join("note.txt"), b"hello").expect("write nested file");

        let mut files = OsFileSystem;
        files
            .duplicate(
                source.to_string_lossy().as_ref(),
                target.to_string_lossy().as_ref(),
            )
            .expect("duplicate directory");

        let copied = fs::read(target.join("nested").join("note.txt")).expect("read copied file");
        assert_eq!(copied, b"hello");
        let _ = fs::remove_dir_all(root);
    }

    #[test]
    fn os_file_system_rename_directory_preserves_nested_children() {
        let root =
            std::env::temp_dir().join(format!("vugra-system-rename-nested-{}", std::process::id()));
        let source = root.join("source");
        let nested = source.join("nested");
        let target = root.join("renamed");
        fs::create_dir_all(&nested).expect("create nested source");
        fs::write(nested.join("note.txt"), b"hello").expect("write nested file");

        let mut files = OsFileSystem;
        files
            .rename(
                source.to_string_lossy().as_ref(),
                target.to_string_lossy().as_ref(),
            )
            .expect("rename directory");

        let renamed = fs::read(target.join("nested").join("note.txt")).expect("read renamed file");
        assert_eq!(renamed, b"hello");
        assert!(!source.exists());
        let _ = fs::remove_dir_all(root);
    }
}
