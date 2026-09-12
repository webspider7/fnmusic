Write-Host "=== 1. Building Frontend ===" -ForegroundColor Cyan
Set-Location -Path "g:\fnapp\lx-player\frontend"
npm run build
if ($LASTEXITCODE -ne 0) {
    Write-Error "Frontend build failed!"
    exit 1
}

Write-Host "=== 2. Cross Compiling Linux amd64 Binary ===" -ForegroundColor Cyan
Set-Location -Path "g:\fnapp\lx-player\backend"
$env:GOOS = "linux"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"
$goExe = "G:\fnapp\go_sdk\go\bin\go.exe"

& $goExe build -ldflags="-s -w" -o "g:\fnapp\lx-player\fpk-package\app\fn-lx-player" main.go
if ($LASTEXITCODE -ne 0) {
    Write-Error "Backend compilation failed!"
    exit 1
}

# 复制一份到根目录保持兼容
Copy-Item -Path "g:\fnapp\lx-player\fpk-package\app\fn-lx-player" -Destination "g:\fnapp\lx-player\fpk-package\fn-lx-player" -Force

$binItem = Get-Item "g:\fnapp\lx-player\fpk-package\app\fn-lx-player"
Write-Host "Compiled binary size: $($binItem.Length) bytes, Time: $($binItem.LastWriteTime)" -ForegroundColor Green

Write-Host "=== 3. Packaging FPK with fnpack ===" -ForegroundColor Cyan
Set-Location -Path "g:\fnapp"
& "g:\fnapp\scripts\fnpack.exe" build -d "g:\fnapp\lx-player\fpk-package"
if ($LASTEXITCODE -ne 0) {
    Write-Error "fnpack build failed!"
    exit 1
}

Write-Host "=== 4. Verifying app.tgz inside FPK ===" -ForegroundColor Cyan
python -c "
import tarfile

with tarfile.open(r'g:\fnapp\fn-lx-player.fpk', 'r:*') as t:
    app_member = t.getmember('app.tgz')
    app_f = t.extractfile(app_member)
    with tarfile.open(fileobj=app_f, mode='r:*') as at:
        for m in at.getmembers():
            if m.name == 'fn-lx-player':
                print(f'VERIFIED: app.tgz contains fn-lx-player, Size: {m.size} bytes, mtime: {m.mtime}')
"
Write-Host "=== SUCCESS: Build and Package Completed! ===" -ForegroundColor Green
