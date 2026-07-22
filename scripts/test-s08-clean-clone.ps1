[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$changes = (& git -C $repositoryRoot status --porcelain=v1 --untracked-files=normal | Out-String).Trim()
if ($LASTEXITCODE -ne 0) { throw 'Clean-clone verification requires a valid Git worktree.' }
if ($changes.Length -ne 0) { throw 'Clean-clone verification requires a committed, clean source tree.' }
$revision = (& git -C $repositoryRoot rev-parse HEAD | Out-String).Trim()
if ($LASTEXITCODE -ne 0 -or $revision -notmatch '^[0-9a-f]{40}$') { throw 'Clean-clone source revision is invalid.' }

$cloneParent = Join-Path $repositoryRoot '.tmp/s08-clean-clone'
$cloneRoot = Join-Path $cloneParent ([Guid]::NewGuid().ToString('N'))
$sourceURI = ([Uri]::new(([IO.Path]::GetFullPath($repositoryRoot).TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar))).AbsoluteUri
New-Item -ItemType Directory -Path $cloneParent -Force | Out-Null
try {
    # A file URI forces Git's upload-pack path on Windows and avoids drive-letter
    # paths being misread as an SSH-style remote.
    & git clone --quiet --no-checkout -- $sourceURI $cloneRoot
    if ($LASTEXITCODE -ne 0) { throw 'Create isolated clean clone failed.' }
    & git -C $cloneRoot checkout --quiet --detach $revision
    if ($LASTEXITCODE -ne 0) { throw 'Check out exact clean-clone revision failed.' }
    & pwsh -NoProfile -File (Join-Path $cloneRoot 'scripts/verify-s08.ps1')
    if ($LASTEXITCODE -ne 0) { throw 'Clean-clone S08 static acceptance failed.' }
}
finally {
    if (Test-Path -LiteralPath $cloneRoot) {
        $resolvedClone = [IO.Path]::GetFullPath($cloneRoot)
        $resolvedParent = [IO.Path]::GetFullPath($cloneParent).TrimEnd([IO.Path]::DirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
        if ($resolvedClone.StartsWith($resolvedParent, [StringComparison]::OrdinalIgnoreCase)) {
            Remove-Item -LiteralPath $resolvedClone -Recurse -Force
        }
    }
}

Write-Output "s08_clean_clone_revision=$revision"
Write-Output 's08_clean_clone=PASS'
