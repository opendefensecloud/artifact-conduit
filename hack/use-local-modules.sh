#!/usr/bin/env bash

# use-local-modules.sh
#
# This script safely manages Go module replace directives for local development.
# Cloned modules are stored in a persistent directory and reused across runs.
#
# Usage: ./use-local-modules.sh --dir DIRECTORY [OPTIONS] [MODULE [=REPO]] [MODULE [=REPO] ...]
#
# Examples:
#   ./use-local-modules.sh --dir ~/go-modules github.com/example/module1 github.com/example/module2
#   ./use-local-modules.sh --dir ~/go-modules --version v0.2.0 github.com/ironcore-dev/ironcore
#   ./use-local-modules.sh --dir ~/go-modules github.com/ironcore-dev/ironcore=https://github.com/myorg/fork
#   ./use-local-modules.sh --restore  # Restore original go.mod
#
# Options:
#   --dir DIRECTORY     Directory to store cloned modules (required, unless using --restore)
#   --version VERSION   Git version/tag to clone (default: use version from go.mod)
#   --repo REPO         Default git repository URL (for all modules)
#   --restore           Restore go.mod from backup
#   --help              Show this help message
#
# Module Specification:
#   MODULE              Use default repo URL: https://MODULE.git
#   MODULE=REPO         Use custom repository URL: REPO
#                       Examples:
#                         github.com/ironcore-dev/ironcore=https://github.com/myorg/fork
#                         go.example.com/module=git@github.com:myorg/module.git

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT_NAME="$(basename "${BASH_SOURCE[0]}")"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
GOMOD="$PROJECT_ROOT/go.mod"
GOMOD_BACKUP="$GOMOD.backup"
GOSUM="$PROJECT_ROOT/go.sum"
GOSUM_BACKUP="$GOSUM.backup"
MODULES_DIR=""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}ℹ${NC} $*" >&2
}

log_success() {
    echo -e "${GREEN}✓${NC} $*" >&2
}

log_warn() {
    echo -e "${YELLOW}⚠${NC} $*" >&2
}

log_error() {
    echo -e "${RED}✗${NC} $*" >&2
}

# Print usage
usage() {
    cat << 'EOF'
use-local-modules.sh

This script safely manages Go module replace directives for local development.
Cloned modules are stored in a persistent directory and reused across runs.

Usage: ./use-local-modules.sh --dir DIRECTORY [OPTIONS] [MODULE [=REPO]] [MODULE [=REPO] ...]

Examples:
  # Clone modules to persistent directory
  ./use-local-modules.sh --dir ~/go-modules github.com/example/module1 github.com/example/module2

  # Clone a specific version (reuses existing clone if available)
  ./use-local-modules.sh --dir ~/go-modules --version v0.2.0 github.com/ironcore-dev/ironcore

  # Clone from a fork
  ./use-local-modules.sh --dir ~/go-modules github.com/ironcore-dev/ironcore=https://github.com/myorg/fork

  # Clone from multiple sources
  ./use-local-modules.sh --dir ~/go-modules \
    github.com/ironcore-dev/ironcore=https://github.com/myorg/fork \
    github.com/ironcore-dev/controller-utils

  # Use custom SSH URL
  ./use-local-modules.sh --dir ~/go-modules go.example.com/module=git@github.com:myorg/module.git

  # Set default repo for all modules (if no module-specific repo)
  ./use-local-modules.sh --dir ~/go-modules --repo https://github.com/myorg github.com/module1 github.com/module2

  # Restore original go.mod
  ./use-local-modules.sh --restore

Options:
  --dir DIRECTORY     Directory to store cloned modules (required, unless using --restore)
  --version VERSION   Git version/tag to clone (default: use version from go.mod)
  --repo REPO         Default git repository URL pattern (for modules without explicit =REPO)
  --restore           Restore go.mod from backup
  --help              Show this help message

Module Specification:
  MODULE              Use default URL: https://MODULE.git
  MODULE=REPO         Use custom repository URL: REPO
                      The REPO URL should be a valid git clone URL

Examples of REPO values:
  https://github.com/myorg/fork              HTTPS clone
  git@github.com:myorg/module.git            SSH clone
  file:///path/to/local/repo                 Local repository
  https://git.example.com/custom/vanity/url  Vanity URL

Directory Structure:
  Modules are stored in DIRECTORY as: DIRECTORY/MODULE/path
  Example: ~/go-modules/github.com/ironcore-dev/ironcore

Reuse Benefits:
  - Clones are checked for existence before cloning
  - If a module already exists, it will be reused if the version matches
  - New versions require re-cloning (automatic via git)
  - No temporary files or cleanup needed
EOF
}

# Extract version from go.mod for a given module
get_module_version() {
    local module="$1"

    # Look for the module in go.mod and extract its version
    # Handles formats like: "module v1.2.3" or "module v0.0.0-20250910181357-589584f1c912"
    grep "^\s*$module\s" "$GOMOD" | grep -oE "v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?" | head -1
}

# Validate go.mod exists
validate_gomod() {
    if [[ ! -f "$GOMOD" ]]; then
        log_error "go.mod not found at: $GOMOD"
        exit 1
    fi
    log_success "Found go.mod at: $GOMOD"
}

# Create backup
backup_gomod() {
    if [[ -f "$GOMOD_BACKUP" ]]; then
        log_warn "Backup already exists: $GOMOD_BACKUP"
        log_info "Removing old backup..."
        rm "$GOMOD_BACKUP"
    fi

    cp "$GOMOD" "$GOMOD_BACKUP"
    log_success "Created backup: $GOMOD_BACKUP"

    # Also backup go.sum if it exists
    if [[ -f "$GOSUM" ]]; then
        if [[ -f "$GOSUM_BACKUP" ]]; then
            rm "$GOSUM_BACKUP"
        fi
        cp "$GOSUM" "$GOSUM_BACKUP"
        log_success "Created backup: $GOSUM_BACKUP"
    fi
}

# Restore from backup
restore_gomod() {
    if [[ ! -f "$GOMOD_BACKUP" ]]; then
        log_error "No backup found: $GOMOD_BACKUP"
        exit 1
    fi

    cp "$GOMOD_BACKUP" "$GOMOD"
    rm "$GOMOD_BACKUP"
    log_success "Restored go.mod from backup"

    # Also restore go.sum if backup exists
    if [[ -f "$GOSUM_BACKUP" ]]; then
        cp "$GOSUM_BACKUP" "$GOSUM"
        rm "$GOSUM_BACKUP"
        log_success "Restored go.sum from backup"
    fi
}

# Clone a module to modules directory (or reuse if it exists)
clone_module() {
    local module="$1"
    local repo_url="${2:-}"
    local version="${3:-}"

    # If no repo URL provided, construct default from module name
    if [[ -z "$repo_url" ]]; then
        repo_url="https://$module.git"
    fi

    local clone_dir="$MODULES_DIR/$module"

    # Check if module already exists
    if [[ -d "$clone_dir/.git" ]]; then
        log_info "Module already cloned: $module"

        # If a specific version is requested and it differs, update it
        if [[ -n "$version" ]]; then
            log_info "Checking out version: $version"
            if ! git -C "$clone_dir" checkout --quiet "$version" 2>/dev/null; then
                log_error "Failed to checkout version $version for $module"
                return 1
            fi
        else
            log_info "Using existing clone at: $clone_dir"
        fi

        echo "$clone_dir"
        return 0
    fi

    log_info "Cloning $module..."
    log_info "Repository: $repo_url"
    log_info "This may take a while for large repositories..."

    # Create parent directories
    mkdir -p "$(dirname "$clone_dir")"

    # Clone the repository with progress reporting to stderr only
    # Ensure stdout is not polluted
    if ! git clone --progress "$repo_url" "$clone_dir" 2>&1 | cat >&2; then
        log_error "Failed to clone: $repo_url"
        return 1
    fi

    # Checkout specific version if provided
    if [[ -n "$version" ]]; then
        log_info "Checking out version: $version"
        if ! git -C "$clone_dir" checkout --quiet "$version" 2>/dev/null; then
            log_error "Failed to checkout version $version for $module"
            return 1
        fi
    fi

    log_success "Cloned $module to: $clone_dir"
    echo "$clone_dir"
}

# Check if module is in go.mod
module_exists_in_gomod() {
    local module="$1"
    grep -qE "^[[:space:]]*$(printf '%s\n' "$module" | sed 's/[[\.*^$/]/\\&/g')[[:space:]]" "$GOMOD"
}

# Remove module from go.mod
remove_module_from_gomod() {
    local module="$1"

    if ! module_exists_in_gomod "$module"; then
        log_warn "Module not found in go.mod: $module"
        return 0
    fi

    # Use sed to remove the line containing the module
    # Escape forward slashes in the module name for use in sed regex
    local escaped_module="${module//\//\\/}"
    sed -i.bak "/^[[:space:]]*${escaped_module}[[:space:]]/d" "$GOMOD"
    rm -f "$GOMOD.bak"

    log_success "Removed $module from go.mod"
}

# Add replace directive to go.mod
add_replace_directive() {
    local module="$1"
    local local_path="$2"

    # Escape forward slashes in module name for use in grep
    local escaped_module="${module//\//\\/}"

    # Check if replace already exists
    if grep -qE "^[[:space:]]*replace[[:space:]]+${escaped_module}[[:space:]]*=>" "$GOMOD"; then
        log_warn "Replace directive already exists for: $module"
        return 0
    fi

    # Find the last "require" or "replace" line and add after it
    # If no requires block exists, add at the end
    if grep -qE "^require|^replace" "$GOMOD"; then
        # Add replace directive at the end of the file
        echo "replace $module => $local_path" >> "$GOMOD"
    else
        # Add replace directive before the first block
        sed -i.bak "1i\\replace $module => $local_path" "$GOMOD"
        rm -f "$GOMOD.bak"
    fi

    log_success "Added replace directive for: $module => $local_path"
}

# Validate module reference
validate_module_reference() {
    local module="$1"

    # Basic validation: should be in format like github.com/org/repo
    if ! [[ "$module" =~ ^[a-zA-Z0-9.-]+/[a-zA-Z0-9._-]+/[a-zA-Z0-9._-]+$ ]]; then
        # Also accept shorter forms like github.com/ironcore-dev/ironcore
        if ! [[ "$module" =~ ^[a-zA-Z0-9.-]+/[a-zA-Z0-9._-]+$ ]]; then
            log_error "Invalid module reference: $module"
            log_info "Expected format: domain/org/repo or domain/org"
            return 1
        fi
    fi

    return 0
}

# Validate git repository URL
validate_repo_url() {
    local repo_url="$1"

    # Accept http, https, git, ssh, and file protocols
    if [[ "$repo_url" =~ ^(https?|git|ssh|file)(:\/\/|@) ]] || [[ "$repo_url" =~ ^(git@|file://) ]]; then
        return 0
    fi

    log_error "Invalid repository URL: $repo_url"
    log_info "Expected format: https://..., git@..., ssh://..., or file://..."
    return 1
}

# Parse module specification
# Input: "module" or "module=repo_url"
# Output: "module repo_url" (repo_url empty if not specified)
parse_module_spec() {
    local spec="$1"

    if [[ "$spec" == *"="* ]]; then
        # Module with explicit repository
        local module="${spec%%=*}"
        local repo="${spec#*=}"

        # Validate module
        if ! validate_module_reference "$module"; then
            return 1
        fi

        # Validate repo URL
        if ! validate_repo_url "$repo"; then
            return 1
        fi

        echo "$module $repo"
    else
        # Module with default repository
        if ! validate_module_reference "$spec"; then
            return 1
        fi

        echo "$spec"
    fi
}

# Main function
main() {
    local modules=()
    local version=""
    local default_repo=""
    local restore_mode=false

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dir)
                shift
                MODULES_DIR="$1"
                shift
                ;;
            --version)
                shift
                version="$1"
                shift
                ;;
            --repo)
                shift
                default_repo="$1"
                shift
                ;;
            --restore)
                restore_mode=true
                shift
                ;;
            --help|-h)
                usage
                exit 0
                ;;
            -*)
                log_error "Unknown option: $1"
                usage
                exit 1
                ;;
            *)
                modules+=("$1")
                shift
                ;;
        esac
    done

    # Handle restore mode first (doesn't need directory)
    if [[ $restore_mode == true ]]; then
        validate_gomod
        restore_gomod
        exit 0
    fi

    # For module processing, --dir is required
    if [[ -z "$MODULES_DIR" ]]; then
        log_error "Error: --dir parameter is required (except for --restore mode)"
        log_error ""
        log_error "Usage: $SCRIPT_NAME --dir /path/to/modules [modules...]"
        log_error ""
        log_error "The --dir parameter specifies where cloned modules will be stored and reused."
        exit 1
    fi

    # Validate and create modules directory if it doesn't exist
    if ! mkdir -p "$MODULES_DIR" 2>/dev/null; then
        log_error "Cannot create or access modules directory: $MODULES_DIR"
        exit 1
    fi

    # Validate requirements
    validate_gomod

    # Check that at least one module was provided
    if [[ ${#modules[@]} -eq 0 ]]; then
        log_error "No modules specified"
        usage
        exit 1
    fi

    # Create backup
    backup_gomod

    # Process each module
    local cloned_modules=()
    for module_spec in "${modules[@]}"; do
        log_info "Processing module specification: $module_spec"

        # Parse module specification
        local parsed
        if ! parsed=$(parse_module_spec "$module_spec"); then
            log_error "Skipping invalid module specification: $module_spec"
            continue
        fi

        # Extract module and optional repository
        local module repo_url
        read -r module repo_url <<< "$parsed"

        # Use default repo if not specified
        if [[ -z "$repo_url" && -n "$default_repo" ]]; then
            repo_url="$default_repo"
            log_info "Using default repository for $module: $repo_url"
        fi

        # Determine version to use: explicit --version option, or extract from go.mod
        local version_to_use="$version"
        if [[ -z "$version_to_use" ]]; then
            version_to_use=$(get_module_version "$module")
            if [[ -n "$version_to_use" ]]; then
                log_info "Using version from go.mod for $module: $version_to_use"
            fi
        fi

        # Clone the module
        local clone_path
        if clone_path=$(clone_module "$module" "$repo_url" "$version_to_use"); then
            cloned_modules+=("$module:$clone_path")
        else
            log_error "Failed to clone $module, restoring backup"
            restore_gomod
            exit 1
        fi
    done

    # Update go.mod with replace directives
    log_info "Updating go.mod with replace directives..."
    for entry in "${cloned_modules[@]}"; do
        IFS=':' read -r module path <<< "$entry"

        # Remove from require block (if present)
        remove_module_from_gomod "$module"

        # Add replace directive
        add_replace_directive "$module" "$path"
    done

    log_success "Successfully configured local module replacements"
    log_info "Original go.mod backed up to: $GOMOD_BACKUP"
    log_info ""
    log_info "To restore original go.mod, run:"
    log_info "  $0 --restore"
}

main "$@"
