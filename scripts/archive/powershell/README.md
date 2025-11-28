# Archived PowerShell Scripts

This directory contains archived PowerShell scripts that were previously used for Windows development.

## Important Notice

**These scripts are no longer actively maintained.**

For Windows users, we recommend using **Windows Subsystem for Linux (WSL)** with the bash scripts located in the main `/scripts/` directory.

### Why WSL?

- Better compatibility with blockchain tooling
- Consistent development environment across platforms
- Native support for Linux-based tools and dependencies
- Improved performance for cryptographic operations

### Setting Up WSL

1. Install WSL2 on Windows:
   ```powershell
   wsl --install
   ```

2. Install Ubuntu or your preferred Linux distribution

3. Use the bash scripts in `/scripts/` instead of these PowerShell scripts

## Archived Scripts

- **local-ci.ps1** - Local CI testing (use `/scripts/ci/local-ci.sh` instead)
- **replace.ps1** - File replacement utility (use Unix tools or bash scripts instead)

## Migration Path

If you need functionality from these scripts:
1. Check if an equivalent bash script exists in `/scripts/`
2. Use WSL to run the bash version
3. Contribute improvements to the bash scripts if needed

For questions or issues, please open a  issue.
