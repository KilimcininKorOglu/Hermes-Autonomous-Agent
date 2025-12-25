#Requires -Version 7.0

<#
.SYNOPSIS
    Git Branch Manager Module for Hermes
.DESCRIPTION
    Handles feature branch creation, task commits, and merging.
    Implements feature/FXXX-name branch naming convention.
#>

function Get-CurrentBranch {
    <#
    .SYNOPSIS
        Gets the current git branch name
    .OUTPUTS
        Branch name string or empty
    #>
    try {
        $branch = git rev-parse --abbrev-ref HEAD 2>&1
        if ($LASTEXITCODE -eq 0) {
            return $branch.Trim()
        }
    }
    catch {}
    return ""
}

function Test-GitRepository {
    <#
    .SYNOPSIS
        Checks if current directory is a git repository
    .OUTPUTS
        Boolean
    #>
    try {
        $null = git rev-parse --git-dir 2>&1
        return $LASTEXITCODE -eq 0
    }
    catch {
        return $false
    }
}

function Test-BranchExists {
    <#
    .SYNOPSIS
        Checks if a branch exists locally
    .OUTPUTS
        Boolean
    #>
    param(
        [Parameter(Mandatory)]
        [string]$Name
    )
    
    try {
        $null = git rev-parse --verify $Name 2>&1
        return $LASTEXITCODE -eq 0
    }
    catch {
        return $false
    }
}

function Test-RemoteBranchExists {
    <#
    .SYNOPSIS
        Checks if a branch exists on remote
    .OUTPUTS
        Boolean
    #>
    param(
        [Parameter(Mandatory)]
        [string]$Name,
        
        [string]$Remote = "origin"
    )
    
    try {
        $result = git ls-remote --heads $Remote $Name 2>&1
        return ($result -and $result.Length -gt 0)
    }
    catch {
        return $false
    }
}

function Get-MainBranch {
    <#
    .SYNOPSIS
        Determines the main branch name (main or master)
    .OUTPUTS
        Branch name string
    #>
    if (Test-BranchExists -Name "main") {
        return "main"
    }
    elseif (Test-BranchExists -Name "master") {
        return "master"
    }
    else {
        return "main"
    }
}

function Get-FeatureBranchName {
    <#
    .SYNOPSIS
        Generates feature branch name from feature ID and name
    .OUTPUTS
        Branch name like "feature/F001-user-registration"
    #>
    param(
        [Parameter(Mandatory)]
        [string]$FeatureId,
        
        [Parameter(Mandatory)]
        [string]$FeatureName
    )
    
    # Sanitize feature name for branch
    $safeName = $FeatureName.ToLower() -replace "[^a-z0-9]", "-"
    $safeName = $safeName -replace "-+", "-"
    $safeName = $safeName.Trim("-")
    
    # Limit length
    if ($safeName.Length -gt 30) {
        $safeName = $safeName.Substring(0, 30).TrimEnd("-")
    }
    
    return "feature/$FeatureId-$safeName"
}

function New-FeatureBranch {
    <#
    .SYNOPSIS
        Creates a new feature branch from main
    .OUTPUTS
        Boolean indicating success
    #>
    param(
        [Parameter(Mandatory)]
        [string]$FeatureId,
        
        [Parameter(Mandatory)]
        [string]$FeatureName,
        
        [switch]$FromCurrent
    )
    
    $branchName = Get-FeatureBranchName -FeatureId $FeatureId -FeatureName $FeatureName
    
    if (Test-BranchExists -Name $branchName) {
        Write-Host "[WARN] Branch already exists: $branchName" -ForegroundColor Yellow
        return $true
    }
    
    try {
        if ($FromCurrent) {
            git checkout -b $branchName 2>&1 | Out-Null
        }
        else {
            $mainBranch = Get-MainBranch
            git checkout -b $branchName $mainBranch 2>&1 | Out-Null
        }
        
        if ($LASTEXITCODE -eq 0) {
            Write-Host "[OK] Created branch: $branchName" -ForegroundColor Green
            return $true
        }
    }
    catch {}
    
    Write-Host "[ERROR] Failed to create branch: $branchName" -ForegroundColor Red
    return $false
}

function Switch-ToFeatureBranch {
    <#
    .SYNOPSIS
        Switches to an existing feature branch
    .OUTPUTS
        Boolean indicating success
    #>
    param(
        [Parameter(Mandatory)]
        [string]$BranchName
    )
    
    $currentBranch = Get-CurrentBranch
    
    if ($currentBranch -eq $BranchName) {
        return $true
    }
    
    if (-not (Test-BranchExists -Name $BranchName)) {
        Write-Host "[ERROR] Branch does not exist: $BranchName" -ForegroundColor Red
        return $false
    }
    
    try {
        git checkout $BranchName 2>&1 | Out-Null
        
        if ($LASTEXITCODE -eq 0) {
            Write-Host "[OK] Switched to branch: $BranchName" -ForegroundColor Green
            return $true
        }
    }
    catch {}
    
    Write-Host "[ERROR] Failed to switch to branch: $BranchName" -ForegroundColor Red
    return $false
}

function Switch-ToMain {
    <#
    .SYNOPSIS
        Switches to the main branch
    .OUTPUTS
        Boolean indicating success
    #>
    $mainBranch = Get-MainBranch
    
    try {
        git checkout $mainBranch 2>&1 | Out-Null
        
        if ($LASTEXITCODE -eq 0) {
            Write-Host "[OK] Switched to: $mainBranch" -ForegroundColor Green
            return $true
        }
    }
    catch {}
    
    Write-Host "[ERROR] Failed to switch to $mainBranch" -ForegroundColor Red
    return $false
}

function Test-WorkingTreeClean {
    <#
    .SYNOPSIS
        Checks if working tree is clean (no uncommitted changes)
    .OUTPUTS
        Boolean
    #>
    try {
        $status = git status --porcelain 2>&1
        return [string]::IsNullOrEmpty($status)
    }
    catch {
        return $false
    }
}

function Test-StagedChanges {
    <#
    .SYNOPSIS
        Checks if there are staged changes
    .OUTPUTS
        Boolean
    #>
    try {
        $diff = git diff --cached --name-only 2>&1
        return (-not [string]::IsNullOrEmpty($diff))
    }
    catch {
        return $false
    }
}

function Add-AllChanges {
    <#
    .SYNOPSIS
        Stages all changes
    .OUTPUTS
        Boolean indicating success
    #>
    try {
        git add -A 2>&1 | Out-Null
        return $LASTEXITCODE -eq 0
    }
    catch {
        return $false
    }
}

function New-TaskCommit {
    <#
    .SYNOPSIS
        Creates a commit for a completed task
    .DESCRIPTION
        Commit format: feat(TXXX): Task name completed
    #>
    param(
        [Parameter(Mandatory)]
        [string]$TaskId,
        
        [Parameter(Mandatory)]
        [string]$TaskName,
        
        [string[]]$FilesModified = @(),
        
        [string[]]$SuccessCriteria = @(),
        
        [switch]$AddAll
    )
    
    if ($AddAll) {
        Add-AllChanges | Out-Null
    }
    
    if (-not (Test-StagedChanges)) {
        Write-Host "[WARN] No staged changes to commit" -ForegroundColor Yellow
        return $false
    }
    
    # Build commit message
    $subject = "feat($TaskId): $TaskName completed"
    
    $body = @()
    
    if ($SuccessCriteria.Count -gt 0) {
        $body += "Completed:"
        foreach ($criteria in $SuccessCriteria) {
            $body += "- [x] $criteria"
        }
        $body += ""
    }
    
    if ($FilesModified.Count -gt 0) {
        $body += "Files:"
        foreach ($file in $FilesModified) {
            $body += "- $file"
        }
    }
    
    $message = if ($body.Count -gt 0) {
        "$subject`n`n$($body -join "`n")"
    } else {
        $subject
    }
    
    try {
        git commit -m $message 2>&1 | Out-Null
        
        if ($LASTEXITCODE -eq 0) {
            Write-Host "[OK] Committed: $subject" -ForegroundColor Green
            return $true
        }
    }
    catch {}
    
    Write-Host "[ERROR] Failed to commit: $subject" -ForegroundColor Red
    return $false
}

function New-FeatureCommit {
    <#
    .SYNOPSIS
        Creates a feature completion commit
    #>
    param(
        [Parameter(Mandatory)]
        [string]$FeatureId,
        
        [Parameter(Mandatory)]
        [string]$FeatureName,
        
        [int]$TaskCount = 0,
        
        [switch]$AddAll
    )
    
    if ($AddAll) {
        Add-AllChanges | Out-Null
    }
    
    if (-not (Test-StagedChanges)) {
        # No changes to commit, that's ok for feature completion
        return $true
    }
    
    $subject = "feat($FeatureId): $FeatureName feature completed"
    $body = if ($TaskCount -gt 0) { "`nCompleted $TaskCount tasks." } else { "" }
    $message = "$subject$body"
    
    try {
        git commit -m $message 2>&1 | Out-Null
        
        if ($LASTEXITCODE -eq 0) {
            Write-Host "[OK] Feature committed: $subject" -ForegroundColor Green
            return $true
        }
    }
    catch {}
    
    Write-Host "[WARN] No changes to commit for feature" -ForegroundColor Yellow
    return $true
}

function Merge-FeatureToMain {
    <#
    .SYNOPSIS
        Merges feature branch to main using --no-ff
    .DESCRIPTION
        Switches to main, merges feature branch, then optionally deletes feature branch
    #>
    param(
        [Parameter(Mandatory)]
        [string]$BranchName,
        
        [string]$FeatureId = "",
        
        [string]$FeatureName = "",
        
        [switch]$DeleteBranch
    )
    
    $currentBranch = Get-CurrentBranch
    $mainBranch = Get-MainBranch
    
    # Switch to main
    if (-not (Switch-ToMain)) {
        return $false
    }
    
    # Merge with --no-ff
    $mergeMessage = if ($FeatureId -and $FeatureName) {
        "Merge feature $FeatureId - $FeatureName"
    } else {
        "Merge branch '$BranchName'"
    }
    
    try {
        git merge --no-ff $BranchName -m $mergeMessage 2>&1 | Out-Null
        
        if ($LASTEXITCODE -eq 0) {
            Write-Host "[OK] Merged $BranchName to $mainBranch" -ForegroundColor Green
            
            if ($DeleteBranch) {
                git branch -d $BranchName 2>&1 | Out-Null
                Write-Host "[OK] Deleted branch: $BranchName" -ForegroundColor Green
            }
            
            return $true
        }
        else {
            Write-Host "[ERROR] Merge conflict detected" -ForegroundColor Red
            # Return to original branch on conflict
            git checkout $currentBranch 2>&1 | Out-Null
            return $false
        }
    }
    catch {
        Write-Host "[ERROR] Merge failed: $_" -ForegroundColor Red
        git checkout $currentBranch 2>&1 | Out-Null
        return $false
    }
}

function Get-LastCommitHash {
    <#
    .SYNOPSIS
        Gets the hash of the last commit
    .OUTPUTS
        Short commit hash string
    #>
    try {
        $hash = git rev-parse --short HEAD 2>&1
        if ($LASTEXITCODE -eq 0) {
            return $hash.Trim()
        }
    }
    catch {}
    return ""
}

function Get-CommitsSinceMain {
    <#
    .SYNOPSIS
        Gets the number of commits ahead of main
    .OUTPUTS
        Integer count
    #>
    $mainBranch = Get-MainBranch
    
    try {
        $count = git rev-list --count "$mainBranch..HEAD" 2>&1
        if ($LASTEXITCODE -eq 0) {
            return [int]$count.Trim()
        }
    }
    catch {}
    return 0
}

function Get-ModifiedFiles {
    <#
    .SYNOPSIS
        Gets list of modified files (staged and unstaged)
    .OUTPUTS
        Array of file paths
    #>
    try {
        $files = git diff --name-only HEAD 2>&1
        if ($LASTEXITCODE -eq 0 -and $files) {
            return @($files -split "`n" | Where-Object { $_ })
        }
    }
    catch {}
    return @()
}

function Get-StagedFiles {
    <#
    .SYNOPSIS
        Gets list of staged files
    .OUTPUTS
        Array of file paths
    #>
    try {
        $files = git diff --cached --name-only 2>&1
        if ($LASTEXITCODE -eq 0 -and $files) {
            return @($files -split "`n" | Where-Object { $_ })
        }
    }
    catch {}
    return @()
}

function Undo-LastCommit {
    <#
    .SYNOPSIS
        Undoes the last commit, keeping changes staged
    .OUTPUTS
        Boolean indicating success
    #>
    try {
        git reset --soft HEAD~1 2>&1 | Out-Null
        if ($LASTEXITCODE -eq 0) {
            Write-Host "[OK] Undid last commit" -ForegroundColor Green
            return $true
        }
    }
    catch {}
    
    Write-Host "[ERROR] Failed to undo commit" -ForegroundColor Red
    return $false
}

function Stash-Changes {
    <#
    .SYNOPSIS
        Stashes current changes
    .OUTPUTS
        Boolean indicating success
    #>
    param(
        [string]$Message = "Hermes auto-stash"
    )
    
    try {
        git stash push -m $Message 2>&1 | Out-Null
        return $LASTEXITCODE -eq 0
    }
    catch {
        return $false
    }
}

function Pop-Stash {
    <#
    .SYNOPSIS
        Pops the latest stash
    .OUTPUTS
        Boolean indicating success
    #>
    try {
        git stash pop 2>&1 | Out-Null
        return $LASTEXITCODE -eq 0
    }
    catch {
        return $false
    }
}

function Abort-Merge {
    <#
    .SYNOPSIS
        Aborts an in-progress merge
    .OUTPUTS
        Boolean indicating success
    #>
    try {
        git merge --abort 2>&1 | Out-Null
        return $LASTEXITCODE -eq 0
    }
    catch {
        return $false
    }
}

function Test-MergeInProgress {
    <#
    .SYNOPSIS
        Checks if a merge is in progress
    .OUTPUTS
        Boolean
    #>
    $gitDir = git rev-parse --git-dir 2>&1
    if ($LASTEXITCODE -eq 0) {
        return Test-Path (Join-Path $gitDir "MERGE_HEAD")
    }
    return $false
}

function Get-BranchesForFeature {
    <#
    .SYNOPSIS
        Gets all branches for a feature ID
    .OUTPUTS
        Array of branch names
    #>
    param(
        [Parameter(Mandatory)]
        [string]$FeatureId
    )
    
    try {
        $branches = git branch --list "feature/$FeatureId-*" 2>&1
        if ($LASTEXITCODE -eq 0 -and $branches) {
            return @($branches -replace "^\*?\s+", "" | Where-Object { $_ })
        }
    }
    catch {}
    return @()
}

function Push-CurrentBranch {
    <#
    .SYNOPSIS
        Pushes current branch to remote
    .OUTPUTS
        Boolean indicating success
    #>
    param(
        [switch]$SetUpstream
    )
    
    $branch = Get-CurrentBranch
    
    try {
        if ($SetUpstream) {
            git push -u origin $branch 2>&1 | Out-Null
        }
        else {
            git push 2>&1 | Out-Null
        }
        
        if ($LASTEXITCODE -eq 0) {
            Write-Host "[OK] Pushed $branch to remote" -ForegroundColor Green
            return $true
        }
    }
    catch {}
    
    Write-Host "[ERROR] Failed to push $branch" -ForegroundColor Red
    return $false
}

# Export functions for dot-sourcing
