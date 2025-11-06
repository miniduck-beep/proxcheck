#!/bin/bash

# Script to setup GitHub repository for Proxy Test API
# Usage: ./scripts/setup-github.sh [repository-name]

set -e

# Default values
REPO_NAME="${1:-proxy-test-api}"
GITHUB_USER="your-username"  # Change this to your GitHub username
PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if Git is installed
    if ! command -v git &> /dev/null; then
        log_error "Git is not installed. Please install Git first."
        exit 1
    fi
    
    # Check if GitHub CLI is installed (optional but recommended)
    if command -v gh &> /dev/null; then
        log_success "GitHub CLI is installed"
        GITHUB_CLI=true
    else
        log_warning "GitHub CLI is not installed. Manual repository creation will be required."
        GITHUB_CLI=false
    fi
    
    # Check if we're in a Git repository
    if git rev-parse --git-dir > /dev/null 2>&1; then
        log_info "Already in a Git repository"
        IS_GIT_REPO=true
    else
        IS_GIT_REPO=false
    fi
    
    log_success "Prerequisites check completed"
}

# Initialize Git repository
init_git_repo() {
    log_info "Initializing Git repository..."
    
    if [ "$IS_GIT_REPO" = false ]; then
        git init
        log_success "Git repository initialized"
    else
        log_info "Git repository already exists"
    fi
}

# Create GitHub repository
create_github_repo() {
    log_info "Creating GitHub repository..."
    
    if [ "$GITHUB_CLI" = true ]; then
        # Use GitHub CLI to create repository
        gh repo create "$REPO_NAME" \
            --description "Proxy Test API - Professional proxy testing system with REST API" \
            --public \
            --push
        
        if [ $? -eq 0 ]; then
            log_success "GitHub repository created successfully"
        else
            log_error "Failed to create GitHub repository"
            log_info "Please create the repository manually at: https://github.com/new"
        fi
    else
        log_warning "GitHub CLI not available"
        log_info "Please create the repository manually:"
        echo "  - Go to: https://github.com/new"
        echo "  - Repository name: $REPO_NAME"
        echo "  - Description: Proxy Test API - Professional proxy testing system with REST API"
        echo "  - Public repository"
        echo "  - Don't initialize with README (we'll add our files)"
    fi
}

# Add files and make initial commit
add_files_and_commit() {
    log_info "Adding files to Git..."
    
    # Add all files
    git add .
    
    # Check if there are any changes
    if git diff --cached --quiet; then
        log_warning "No changes to commit"
        return
    fi
    
    # Create initial commit
    git commit -m "feat: initial commit - Proxy Test API

- REST API for proxy testing
- CLI client for easy management
- Comprehensive documentation
- Ready for production use"
    
    log_success "Initial commit created"
}

# Set remote origin and push
setup_remote_and_push() {
    log_info "Setting up remote repository..."
    
    # Check if remote already exists
    if git remote get-url origin > /dev/null 2>&1; then
        log_info "Remote origin already exists"
    else
        # Set remote origin
        git remote add origin "https://github.com/$GITHUB_USER/$REPO_NAME.git"
        log_success "Remote origin set"
    fi
    
    # Push to GitHub
    log_info "Pushing to GitHub..."
    git push -u origin main
    
    if [ $? -eq 0 ]; then
        log_success "Code pushed to GitHub successfully"
    else
        log_error "Failed to push to GitHub"
        log_info "You may need to create the repository first"
    fi
}

# Create GitHub workflow for CI/CD
create_github_workflow() {
    log_info "Creating GitHub workflow..."
    
    mkdir -p .github/workflows
    
    cat > .github/workflows/ci.yml << 'EOF'
name: CI

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    name: Test
    runs-on: ubuntu-latest
    
    steps:
    - name: Checkout code
      uses: actions/checkout@v4
      
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
        
    - name: Install dependencies
      run: go mod tidy
      
    - name: Run tests
      run: go test ./...
      
    - name: Build
      run: |
        go build -o bin/api_server cmd/api/improved_api.go
        go build -o bin/api_client cmd/api/improved_client.go
        
    - name: Upload artifacts
      uses: actions/upload-artifact@v4
      with:
        name: binaries
        path: bin/
        retention-days: 7

  lint:
    name: Lint
    runs-on: ubuntu-latest
    
    steps:
    - name: Checkout code
      uses: actions/checkout@v4
      
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
        
    - name: Run golangci-lint
      uses: golangci/golangci-lint-action@v3
      with:
        version: latest
EOF
    
    log_success "GitHub workflow created"
}

# Display final instructions
show_final_instructions() {
    log_success "🎉 Project setup completed!"
    echo ""
    echo "📋 Next steps:"
    echo ""
    echo "1. Repository URL:"
    echo "   https://github.com/$GITHUB_USER/$REPO_NAME"
    echo ""
    echo "2. If repository creation failed, create it manually:"
    echo "   https://github.com/new"
    echo ""
    echo "3. Push your code:"
    echo "   git push -u origin main"
    echo ""
    echo "4. Update the README.md with your actual GitHub username"
    echo ""
    echo "5. Configure GitHub Pages (optional):"
    echo "   Settings → Pages → Source: GitHub Actions"
    echo ""
    echo "6. Share your project!"
    echo ""
}

# Main execution
main() {
    echo ""
    echo "🚀 Proxy Test API - GitHub Setup Script"
    echo "========================================"
    echo ""
    
    cd "$PROJECT_DIR"
    
    check_prerequisites
    init_git_repo
    create_github_repo
    add_files_and_commit
    create_github_workflow
    setup_remote_and_push
    show_final_instructions
}

# Run main function
main "$@"
