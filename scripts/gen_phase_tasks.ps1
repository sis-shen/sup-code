# gen_phase_tasks.ps1 — 生成阶段任务看板与验收报告骨架（Windows 包装）
# 真正的生成逻辑在 gen_phase_tasks.py（UTF-8 安全）。
# Usage: powershell -ExecutionPolicy Bypass -File scripts/gen_phase_tasks.ps1 [-Force]
param([switch]$Force)

$ErrorActionPreference = 'Stop'
$py = $null
foreach ($cand in @('python', 'python3', 'py')) {
    $cmd = Get-Command $cand -ErrorAction SilentlyContinue
    if ($cmd) { $py = $cmd.Source; break }
}
if (-not $py) { throw "python not found; install Python 3 or run gen_phase_tasks.sh on Linux" }

$script = Join-Path $PSScriptRoot 'gen_phase_tasks.py'
$extra = @()
if ($Force) { $extra += '--force' }
& $py $script @extra
