# PowerShell version of format-all.sh for Windows
# PAW Blockchain Code Formatter

$ErrorActionPreference = "Continue"

Write-Host "========================================" -ForegroundColor Blue
Write-Host "  PAW Blockchain Code Formatter" -ForegroundColor Blue
Write-Host "========================================" -ForegroundColor Blue
Write-Host ""

# Function to check if command exists
function Test-CommandExists {
    param($command)
    $null = Get-Command $command -ErrorAction SilentlyContinue
    return $?
}

# Count files formatted
$GO_COUNT = 0
$JS_COUNT = 0
$PY_COUNT = 0
$PROTO_COUNT = 0

# Format Go files
Write-Host "Formatting Go files..." -ForegroundColor Yellow
if (Test-CommandExists gofmt) {
    $goFiles = Get-ChildItem -Recurse -Filter "*.go" | Where-Object {
        $_.FullName -notmatch "\\vendor\\" -and
        $_.FullName -notmatch "\\\\" -and
        $_.FullName -notmatch "statik\.go"
    }

    foreach ($file in $goFiles) {
        gofmt -w -s $file.FullName
        $GO_COUNT++
    }
    Write-Host "✓ Formatted $GO_COUNT Go files with gofmt" -ForegroundColor Green
} else {
    Write-Host "✗ gofmt not found" -ForegroundColor Red
}

# Fix common misspellings
if (Test-CommandExists misspell) {
    Write-Host "Fixing misspellings in Go files..." -ForegroundColor Yellow
    $goFiles = Get-ChildItem -Recurse -Filter "*.go" | Where-Object {
        $_.FullName -notmatch "\\vendor\\" -and
        $_.FullName -notmatch "\\\\"
    }

    foreach ($file in $goFiles) {
        misspell -w $file.FullName
    }
    Write-Host "✓ Misspellings fixed" -ForegroundColor Green
} else {
    Write-Host "! misspell not found, skipping" -ForegroundColor Yellow
}

# Fix imports
if (Test-CommandExists goimports) {
    Write-Host "Fixing imports in Go files..." -ForegroundColor Yellow
    $goFiles = Get-ChildItem -Recurse -Filter "*.go" | Where-Object {
        $_.FullName -notmatch "\\vendor\\" -and
        $_.FullName -notmatch "\\\\"
    }

    foreach ($file in $goFiles) {
        goimports -w -local example.com/paw-chain/paw $file.FullName
    }
    Write-Host "✓ Imports fixed" -ForegroundColor Green
} else {
    Write-Host "! goimports not found, skipping" -ForegroundColor Yellow
}

# Format JavaScript/TypeScript files
if (Test-CommandExists prettier) {
    Write-Host "`nFormatting JavaScript/TypeScript files..." -ForegroundColor Yellow
    $jsFiles = Get-ChildItem -Recurse -Include "*.js", "*.ts", "*.jsx", "*.tsx"

    if ($jsFiles.Count -gt 0) {
        prettier --write "**/*.{js,ts,jsx,tsx,json}" 2>$null
        $JS_COUNT = $jsFiles.Count
        Write-Host "✓ Formatted $JS_COUNT JavaScript/TypeScript files" -ForegroundColor Green
    } else {
        Write-Host "! No JavaScript/TypeScript files found" -ForegroundColor Yellow
    }
} else {
    Write-Host "! prettier not found, skipping JavaScript/TypeScript formatting" -ForegroundColor Yellow
}

# Format Python files
if (Test-CommandExists black) {
    Write-Host "`nFormatting Python files..." -ForegroundColor Yellow
    $pyFiles = Get-ChildItem -Recurse -Filter "*.py" | Where-Object {
        $_.FullName -notmatch "\\\\" -and
        $_.FullName -notmatch "\\venv\\" -and
        $_.FullName -notmatch "\\env\\"
    }

    if ($pyFiles.Count -gt 0) {
        black . --exclude="/(\|\.venv|venv|env|build|dist)/" 2>$null
        $PY_COUNT = $pyFiles.Count
        Write-Host "✓ Formatted $PY_COUNT Python files" -ForegroundColor Green
    } else {
        Write-Host "! No Python files found" -ForegroundColor Yellow
    }
} elseif (Test-CommandExists autopep8) {
    Write-Host "`nFormatting Python files with autopep8..." -ForegroundColor Yellow
    $pyFiles = Get-ChildItem -Recurse -Filter "*.py" | Where-Object {
        $_.FullName -notmatch "\\\\" -and
        $_.FullName -notmatch "\\venv\\"
    }

    if ($pyFiles.Count -gt 0) {
        foreach ($file in $pyFiles) {
            autopep8 --in-place --aggressive --aggressive $file.FullName
        }
        $PY_COUNT = $pyFiles.Count
        Write-Host "✓ Formatted $PY_COUNT Python files" -ForegroundColor Green
    }
} else {
    Write-Host "! black/autopep8 not found, skipping Python formatting" -ForegroundColor Yellow
}

# Format Protobuf files
if (Test-CommandExists clang-format) {
    Write-Host "`nFormatting Protobuf files..." -ForegroundColor Yellow
    $protoFiles = Get-ChildItem -Recurse -Filter "*.proto" | Where-Object {
        $_.FullName -notmatch "\\third_party\\"
    }

    if ($protoFiles.Count -gt 0) {
        foreach ($file in $protoFiles) {
            clang-format -i $file.FullName
        }
        $PROTO_COUNT = $protoFiles.Count
        Write-Host "✓ Formatted $PROTO_COUNT Protobuf files" -ForegroundColor Green
    } else {
        Write-Host "! No Protobuf files found" -ForegroundColor Yellow
    }
} else {
    Write-Host "! clang-format not found, skipping Protobuf formatting" -ForegroundColor Yellow
}

# Format Markdown files
if (Test-CommandExists prettier) {
    Write-Host "`nFormatting Markdown files..." -ForegroundColor Yellow
    $mdFiles = Get-ChildItem -Recurse -Filter "*.md"

    if ($mdFiles.Count -gt 0) {
        prettier --write "**/*.md" 2>$null
        Write-Host "✓ Markdown files formatted" -ForegroundColor Green
    }
}

# Summary
Write-Host "`n========================================" -ForegroundColor Blue
Write-Host "  Formatting Complete!" -ForegroundColor Blue
Write-Host "========================================" -ForegroundColor Blue
Write-Host "Summary:" -ForegroundColor Green
Write-Host "  Go files: $GO_COUNT"
Write-Host "  JavaScript/TypeScript files: $JS_COUNT"
Write-Host "  Python files: $PY_COUNT"
Write-Host "  Protobuf files: $PROTO_COUNT"
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "  1. Review changes: " -NoNewline
Write-Host " diff" -ForegroundColor Green
Write-Host "  2. Run linter: " -NoNewline
Write-Host "make lint" -ForegroundColor Green
Write-Host "  3. Run tests: " -NoNewline
Write-Host "make test" -ForegroundColor Green
Write-Host "  4. Commit changes: " -NoNewline
Write-Host " add . &&  commit" -ForegroundColor Green
Write-Host ""
