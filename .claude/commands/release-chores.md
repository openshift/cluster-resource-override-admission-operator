---
description: Automate Cluster Resource Override Admission Operator release version bumps and dependency updates
argument-hint: <new_version> [new_go_version] [new_k8s_version] [dry_run]
allowed-tools: Read, Edit, Glob, Grep, Bash(curl:*), Bash(skopeo inspect:*), Bash(skopeo list-tags:*), Bash(SKIP_BUILD=true ./hack/generate-bundle.sh:*), Bash(go get:*), Bash(go list:*), Bash(./hack/update-vendor.sh:*), Bash(go mod tidy:*), Bash(go mod vendor:*), Bash(git checkout:*), Bash(git add:*), Bash(git commit:*), Bash(git show:*), Bash(sed:*)
---

# Cluster Resource Override Admission Operator Release Chores Automation

Automate version bumps and release chores for cluster-resource-override-admission-operator.

## Parameters

Arguments are positional (use empty strings `""` to skip optional parameters):

1. `$1` - **new_version** (required): New OpenShift version (e.g., `4.22`)
2. `$2` - **new_go_version** (optional): New Go version with patch (e.g., `1.24.6`) - auto-detected from ocp-build-data `streams.yml` if not provided
3. `$3` - **new_k8s_version** (optional): New Kubernetes go package version (e.g., `v0.33.0`) - auto-detected if not provided
4. `$4` - **dry_run** (optional): Set to `true` to preview changes without applying (default: `false`)

## Important Notes

- **Go version**: The Go version in `go.mod` MUST match the exact version (including patch) used in CI, which is determined by the ocp-build-data `streams.yml` file for the target OpenShift version
- **CI Operator updates**: The `.ci-operator.yaml` file only gets bumped by automatic PRs from the ART team
- **Commit messages**: Use simple descriptive titles without Jira prefixes
- **Prerequisite**: Wait for ART team to merge their PR updating `.ci-operator.yaml` first (see [example PR #194](https://github.com/openshift/cluster-resource-override-admission-operator/pull/194/files))
- **Go version source**: The workflow automatically fetches the correct Go patch version from `https://github.com/openshift-eng/ocp-build-data/blob/openshift-{VERSION}/streams.yml`

## ⚠️ SECURITY WARNING

**This workflow executes commands, modifies files, and creates git commits in your repository.**

- The workflow runs scripts from the repository (`./hack/update-vendor.sh`, `./hack/generate-bundle.sh`)
- The workflow downloads Go dependencies from the internet (`go get`, `go mod tidy`)
- The workflow creates a git branch, stages changes, and creates a commit
- The workflow does NOT push changes - you must review and push manually

**Best practice**: Make sure you have a clean working tree and are on a branch you can easily reset if needed.

## Your Task

Execute the following steps to perform the release chores for OpenShift version **$1**:

### Step 1: Pre-flight Checks

1. Parse and validate arguments:
   - `NEW_VERSION="$1"` (required)
   - `NEW_GO_VERSION_OVERRIDE="$2"` (optional)
   - `NEW_K8S_VERSION_OVERRIDE="$3"` (optional)
   - `DRY_RUN="${4:-false}"` (optional, default: false)

2. Validate inputs:
   - Ensure `NEW_VERSION` is provided and matches format `X.Y` (e.g., `4.22`)
   - If `NEW_GO_VERSION_OVERRIDE` is provided, ensure it matches format `X.Y` or `X.Y.Z` (e.g., `1.24` or `1.24.6`)
   - If `NEW_K8S_VERSION_OVERRIDE` is provided, ensure it matches format `vX.Y.Z` (e.g., `v0.33.0`)
   - If `DRY_RUN` is provided, ensure it's `true` or `false`

3. Verify environment:
   - Check that `@manifests/clusterresourceoverride-operator.package.yaml` exists (confirms correct repository)
   - Check for uncommitted changes with `git status` and warn user if any exist
   - If uncommitted changes exist, **STOP** and ask user to:
     1. Commit or stash existing changes first
     2. Or explicitly approve continuing (changes will be included in the update commit)

### Step 2: Detect Current Versions

Use `grep` to extract current versions from files:

1. **Current OpenShift version**: Extract from `@Makefile` using pattern `IMAGE_VERSION\s*:=\s*\K[0-9]+\.[0-9]+`
2. **Current Go version (CI)**: Extract from `@.ci-operator.yaml` using pattern `golang-\K[0-9]+\.[0-9]+`
3. **Current Go version (Dockerfile)**: Extract from `@images/ci/Dockerfile` using pattern `golang-\K[0-9]+\.[0-9]+`
4. **Current Kubernetes version**: Extract from `@hack/update-vendor.sh` using pattern `kube_release="\K[^"]+`
5. **Current operator-framework version**: Extract from `@images/operator-registry/Dockerfile.registry.ci` using pattern `upstream-registry-builder:\K[^" ]+`
6. **Current UBI version**: Extract from `@images/dev/Dockerfile.dev` using pattern `ubi9-minimal:\K[0-9.]+`
7. **Current RHEL version**: Extract from `@.ci-operator.yaml` using pattern `rhel-\K[0-9]+` (e.g., "9" from "rhel-9-release-golang")

Display all detected versions to the user.

### Step 3: Verify .ci-operator.yaml Has Been Updated

1. Extract OpenShift version from `@.ci-operator.yaml` using pattern `openshift-\K[0-9]+\.[0-9]+`
2. If the version doesn't match `$1` (NEW_VERSION):
   - **WARN** user that `.ci-operator.yaml` hasn't been updated yet
   - Explain that the ART team must create a PR first (reference PR #194 as example)
   - Display current status:
     ```
     .ci-operator.yaml:  [CURRENT_CI_VERSION] ← Still on old version
     Target version:     [NEW_VERSION] ← Cannot proceed safely
     ```
   - **Ask user if they want to**:
     1. **Abort** - Wait for ART PR to be merged (RECOMMENDED)
     2. **Continue anyway** - Bypass this check and use the Go version from `.ci-operator.yaml` or from command-line override
        - If `$2` (NEW_GO_VERSION_OVERRIDE) is provided, use that Go version
        - Otherwise, use the Go version currently in `.ci-operator.yaml` (even though it's for the old OpenShift version)
        - **WARN**: This is risky and may cause CI failures if the Go version is incorrect for the new OpenShift version
   - If user chooses to abort, exit with error
   - If user chooses to continue, proceed but add a note to the commit message indicating `.ci-operator.yaml` was not yet updated
3. If version matches, extract the Go minor version from `.ci-operator.yaml` - this is the authoritative Go minor version to use

### Step 3.5: Determine Full Go Version (with patch) from ocp-build-data

To ensure `go.mod` uses the most up to date Go version that CI uses, fetch the patch version from ocp-build-data:

1. **Construct the streams.yml URL**:
   - URL: `https://raw.githubusercontent.com/openshift-eng/ocp-build-data/openshift-$NEW_VERSION/streams.yml`
   - Example: For version 4.22, use `openshift-4.22` branch

2. **Fetch and parse streams.yml**:
   - Use `curl` to fetch the file: `curl -sSL https://raw.githubusercontent.com/openshift-eng/ocp-build-data/openshift-$NEW_VERSION/streams.yml`
   - Look for the entry matching the RHEL version from `.ci-operator.yaml` (e.g., `rhel-9-golang:`)
   - Extract the `image:` field value
   - Example entry from streams.yml:
     ```yaml
     rhel-9-golang:
       image: quay.io/redhat-user-workloads/ocp-art-tenant/art-images:golang-builder-v1.24.6-202511041143.g4284440.el9
     ```

3. **Parse the Go version from the image tag**:
   - Extract version from pattern: `golang-builder-v\K[0-9]+\.[0-9]+\.[0-9]+`
   - Example: From `golang-builder-v1.24.6-...` extract `1.24.6`
   - Store as `$GO_FULL_VERSION` (e.g., `1.24.6`)
   - The minor version portion must match what's in `.ci-operator.yaml`

4. **Handle Go version override and fetch failures**:
   - If `$2` (NEW_GO_VERSION_OVERRIDE) was provided:
     - If it includes patch version (X.Y.Z format), use it directly as `$GO_FULL_VERSION`
     - If it's minor version only (X.Y format), still attempt to fetch streams.yml to get the patch
     - Extract minor version for `$GO_MINOR_VERSION`
   - If curl fails (network error, 404, etc.) AND no full override was provided:
     - Fall back to `$GO_MINOR_VERSION.0` (append .0 for patch)
     - Warn user that streams.yml couldn't be fetched and patch version is assumed to be .0
     - Mark in commit message that Go patch version couldn't be verified

5. **Store the result**:
   - `$GO_MINOR_VERSION` = minor version only (e.g., `1.24`)
   - `$GO_FULL_VERSION` = full version with patch (e.g., `1.24.6`)
   - Use `$GO_MINOR_VERSION` for Dockerfile updates (they use minor versions)
   - Use `$GO_FULL_VERSION` for `go.mod` updates (they need exact versions)

### Step 4: Auto-detect Version Increments

For versions not provided by user, auto-detect the increments:

1. **Kubernetes version** (if `$3` is empty):
   - Get available Kubernetes versions: `go list -mod=readonly -m -versions k8s.io/api | sed 's/ /\n/g'`
   - Extract the current K8s version from `@hack/update-vendor.sh` (the `kube_release` variable)
   - Find the latest patch version available for the next minor version
   - Example: If current is `v0.31.2`, look for latest `v0.32.x` in the available versions list
   - If this command fails or returns no results, fall back to incrementing minor version by 1 and using `.0` patch (e.g., `v0.31.2` → `v0.32.0`)

2. **Operator-framework version**:
   - Extract minor version from the Kubernetes version (e.g., `v0.33.0` → `33`)
   - Form new version using that minor version: `v1.33.0`
   - If Kubernetes version is not available, don't update the operator-framework image

3. **UBI version**:
   - Get the latest UBI9 minimal version from the registry
   - Command: `skopeo list-tags docker://registry.access.redhat.com/ubi9/ubi-minimal 2>&1 | jq -r '.Tags[]' | grep -E '^9\.[0-9]+$' | sort -V | tail -1`
   - This returns the latest minor version tag (e.g., `9.7`)
   - **Note**: UBI images are versioned along RHEL minor versions. There are no backports - you must adopt the latest to consume CVE fixes.
   - If skopeo or jq fails (not installed, auth error, network issue):
     - Warn user about the issue
     - Fall back to keeping the current UBI version (no change)
     - Mark UBI image with * in verification status
     - Suggest user install tools: `skopeo` and `jq`

### Step 4.5: Verify Container Images Exist

Before proceeding, verify that the new container images actually exist in their respective registries.

**IMPORTANT**: Create a verification status tracker to mark which images have been verified:
- ✓ = Image verified and exists
- * = Image could not be verified (auth required or other error)
- ✗ = Image verified but does NOT exist

**Images to verify**:

Try to verify each image using `skopeo inspect`. If skopeo fails for any reason (not installed, network error, auth error, etc.), handle the error appropriately.

1. **UBI Minimal Image**:
   - Image: `registry.access.redhat.com/ubi9/ubi-minimal:$NEW_UBI_VERSION`
   - Command: `skopeo inspect docker://registry.access.redhat.com/ubi9/ubi-minimal:$NEW_UBI_VERSION --no-tags 2>&1`
   - Mark status: ✓ if success, * if auth/network/skopeo-not-found error, ✗ if 404/not found

2. **Operator Registry Builder Image**:
   - Image: `quay.io/operator-framework/upstream-registry-builder:$NEW_OPERATOR_VERSION`
   - Command: `skopeo inspect docker://quay.io/operator-framework/upstream-registry-builder:$NEW_OPERATOR_VERSION --no-tags 2>&1`
   - Mark status: ✓ if success, * if auth/network/skopeo-not-found error, ✗ if 404/not found

3. **Golang Builder Images**:
   - Image: `registry.ci.openshift.org/ocp/builder:rhel-9-golang-$NEW_GO_VERSION-openshift-$NEW_VERSION`
   - Command: `skopeo inspect docker://registry.ci.openshift.org/ocp/builder:rhel-9-golang-$NEW_GO_VERSION-openshift-$NEW_VERSION --no-tags 2>&1`
   - Mark status: ✓ if success, * if auth/network/skopeo-not-found error, ✗ if 404/not found

**Error Handling**:
Just run skopeo directly. If it fails, check the error output:
- "command not found" or similar → Mark image with *
- "unauthorized", "authentication required" → Mark image with *
- "manifest unknown", "not found", "404" → Mark image with ✗
- Other errors (network, timeout, etc.) → Mark image with *

**Store verification results** for use in Step 5 summary and final commit message. Do not prompt or stop - continue with the workflow.

### Step 5: Display Change Summary

Present a summary table showing versions and their verification status:

```
OpenShift:          [CURRENT_VERSION] → [NEW_VERSION]
Go (Dockerfiles):   [DOCKERFILE_GO_VERSION] → [GO_MINOR_VERSION] (from .ci-operator.yaml) [GO_IMAGE_STATUS]
Go (go.mod):        [CURRENT_GO_MOD_VERSION] → [GO_FULL_VERSION] (from ocp-build-data streams.yml)
Kubernetes:         [CURRENT_K8S_VERSION] → [NEW_K8S_VERSION]
Operator Framework: [CURRENT_OPERATOR_VERSION] → [NEW_OPERATOR_VERSION] [OPERATOR_IMAGE_STATUS]
UBI Minimal:        [CURRENT_UBI_VERSION] → [NEW_UBI_VERSION] [UBI_IMAGE_STATUS]
```

**Status Indicators**:
- ✓ = Image verified and exists
- * = Image could not be verified (requires authentication or check manually)
- ✗ = Image does NOT exist (warning!)
- (no marker) = Not verified or not applicable

List all files to be updated (only if these files are ACTUALLY considered for update):
- `Makefile`
- `hack/update-vendor.sh`
- `images/ci/Dockerfile`
- `images/operator-registry/Dockerfile.registry.ci`
- `images/dev/Dockerfile.dev`
- `manifests/art.yaml`
- `manifests/clusterresourceoverride-operator.package.yaml`
- `manifests/stable/clusterresourceoverride-operator.clusterserviceversion.yaml`
- `manifests/stable/image-references`
- `go.mod` (clean replace directives, update Go version)
- `go.sum` (via `go mod tidy`)
- `vendor/` (via `go mod vendor`)

**Note about image verification**:
- If any images are marked with ✗ or *, note them in the summary but continue without prompting
- These will be documented in the final commit message
- User can review and address any issues after the commit is created

**If `DRY_RUN` is `true`**: Display summary (including verification status) and exit without making changes.

**Otherwise**: Ask user for confirmation before proceeding with the following warning:

```
⚠️  WARNING: This will modify multiple files in your repository.
    Make sure you have a backup branch or clean working tree before proceeding.

    Recommended: Create a backup branch first:
    git branch backup-before-release-$NEW_VERSION
```

### Step 6: Apply Automated Changes

Execute the following file updates using `sed` via Bash:

1. **Clean go.mod**:
   - Remove all `replace` directives: `sed -i '/^replace / d' go.mod`

2. **Update OpenShift version** (replace `$CURRENT_VERSION` with `$NEW_VERSION`) in:
   - `Makefile`
   - `hack/update-vendor.sh`
   - `images/ci/Dockerfile`
   - `images/operator-registry/Dockerfile.registry.ci`
   - `manifests/art.yaml`
   - `manifests/clusterresourceoverride-operator.package.yaml`
   - `manifests/stable/clusterresourceoverride-operator.clusterserviceversion.yaml`
   - `manifests/stable/image-references`

3. **Update Go version** in `@images/ci/Dockerfile`:
   - Replace `$DOCKERFILE_GO_VERSION` with `$GO_MINOR_VERSION` (minor version only, e.g., `1.24`)
   - Only if they differ

4. **Update Kubernetes version** in `@hack/update-vendor.sh`:
   - Replace `$CURRENT_K8S_VERSION` with `$NEW_K8S_VERSION`

5. **Update operator-framework version** in `@images/operator-registry/Dockerfile.registry.ci`:
   - Replace `upstream-registry-builder:$CURRENT_OPERATOR_VERSION` with `upstream-registry-builder:$NEW_OPERATOR_VERSION`

6. **Update UBI version** in `@images/dev/Dockerfile.dev`:
   - Replace `ubi-minimal:$CURRENT_UBI_VERSION` with `ubi-minimal:$NEW_UBI_VERSION`

7. **Update go.mod Go version**:
   - Extract current Go version from `go.mod` using pattern `^go \K[0-9]+\.[0-9]+\.[0-9]+`
   - Update it to `go $GO_FULL_VERSION` (full version with patch, e.g., `1.24.6`)
   - Also update `toolchain` directive if present: `toolchain go$GO_FULL_VERSION`
   - **IMPORTANT**: Use the full version from streams.yml to ensure go.mod matches what CI uses

### Step 7: Update Go Dependencies

Run the following commands in sequence using the Bash tool:

1. **Update build-machinery-go**:
   ```bash
   go get -u github.com/openshift/build-machinery-go
   ```
   - If this fails, warn but continue (not critical)

2. **Run update-vendor.sh script**:
   ```bash
   ./hack/update-vendor.sh
   ```
   - This script updates Kubernetes dependencies to the version specified in the script
   - **If this fails**: Exit with error and inform user to restore changes with `git reset --hard` if needed

3. **Tidy modules**:
   ```bash
   go mod tidy
   ```
   - **If this fails**: Exit with error and inform user to restore changes with `git reset --hard` if needed

4. **Vendor dependencies**:
   ```bash
   go mod vendor
   ```
   - **If this fails**: Exit with error and inform user to restore changes with `git reset --hard` if needed

5. **Regenerate bundle manifests**:
   ```bash
   SKIP_BUILD=true ./hack/generate-bundle.sh
   ```
   - This regenerates the operator bundle manifests with updated versions
   - **If this fails**: Exit with error and inform user to restore changes with `git reset --hard` if needed

### Step 8: Create Git Commit

1. Display changes made:
   ```bash
   git status --short
   git diff --name-only
   ```

2. Verify go.mod matches streams.yml version:
   ```bash
   grep -E '^go ' go.mod
   ```
   - Display the Go version from go.mod and confirm it matches `$GO_FULL_VERSION`

3. Create feature branch:
   ```bash
   git checkout -b update-images-$NEW_VERSION
   ```

4. Stage all changes:
   ```bash
   git add -A
   ```

5. Generate commit message (see Step 9 format below) and commit:
   ```bash
   git commit -m "<generated message from Step 9>

Co-Authored-By: Claude <noreply@anthropic.com>"
   ```
   - Use the commit message format from Step 9
   - Always include the Co-Authored-By trailer to attribute Claude's contribution

6. Display the created commit:
   ```bash
   git show HEAD --stat
   ```

7. Display final summary with image verification results:
   - Show a summary of all version changes made
   - If any images were marked with ✗ or * during verification, list them with explanations:
     - ✗ = Image confirmed not to exist (may need to wait for image to be published)
     - * = Could not verify (authentication, network, or skopeo not installed)
   - Note that these are documented in the commit message for review

8. Provide next steps to user:
   - Review the commit: `git show HEAD`
   - Review all changes: `git diff HEAD~1`
   - If any images had issues, verify manually before merging PR
   - Run tests: `make test`
   - Build locally: `make build`
   - Push and create PR: `git push origin update-images-$NEW_VERSION`
   - If changes needed, amend commit: `git commit --amend`

### Step 9: Commit Message Format

Generate the commit message in this format (this will be used in Step 8's `git commit` command):

```
Updates for $NEW_VERSION

- Update OpenShift version from $CURRENT_VERSION to $NEW_VERSION
- Update Go version from $DOCKERFILE_GO_VERSION to $GO_MINOR_VERSION (Dockerfiles)
- Update Go version from $CURRENT_GO_MOD_VERSION to $GO_FULL_VERSION (go.mod, from ocp-build-data)
- Update Kubernetes version from $CURRENT_K8S_VERSION to $NEW_K8S_VERSION
- Update operator-framework to $NEW_OPERATOR_VERSION [add * if unverified or ✗ if not found]
- Update UBI minimal to $NEW_UBI_VERSION [add * if unverified or ✗ if not found]
- Update build-machinery-go
- Clean up go.mod replace directives

[If .ci-operator.yaml was NOT updated and user chose to bypass, add this section:]
⚠️ WARNING: .ci-operator.yaml has not been updated to $NEW_VERSION yet.
   This update was performed before the ART team's PR was merged.
   The Go version used may be incorrect for OpenShift $NEW_VERSION.
   Review carefully and wait for ART PR before merging.

[If streams.yml couldn't be fetched, add this section:]
⚠️ NOTE: Could not fetch ocp-build-data streams.yml for OpenShift $NEW_VERSION.
   Using Go patch version .0 as fallback. Verify the actual Go version matches CI.

[If any images marked with * or ✗, add this section:]
Note: Some image versions could not be verified:
  * operator-framework image v1.35.0 - authentication required, verify manually
  * UBI minimal 9.11 - could not verify, check availability

Changes:
  - [list of modified files from git diff --name-only]

Co-Authored-By: Claude <noreply@anthropic.com>
```

**Instructions for commit message**:
- If `.ci-operator.yaml` bypass was used, add the ⚠️ WARNING section about .ci-operator.yaml
- If `streams.yml` couldn't be fetched and Go patch version is a fallback, add the ⚠️ NOTE section
- Add * symbol next to any version that couldn't be verified (auth/network issues)
- Add ✗ symbol next to any version that was confirmed to NOT exist
- Include a "Note:" section listing all unverified images if any exist
- Suggest user verify these images manually before merging the PR
- ALWAYS include the "Co-Authored-By: Claude <noreply@anthropic.com>" trailer at the end

## Examples

```bash
# Basic usage - auto-detect all version increments (after ART PR is merged)
/release-chores 4.22

# Dry run - preview changes without applying
/release-chores 4.22 "" "" true

# Specify custom Kubernetes version
/release-chores 4.22 "" v0.35.1

# Specify all parameters explicitly (with full Go version including patch)
/release-chores 4.22 1.24.6 v0.35.1 false

# Use minor Go version only (patch will be auto-detected from streams.yml)
/release-chores 4.22 1.24 v0.35.1 false
```

## Troubleshooting

### Error: ".ci-operator.yaml is still on old version"
- The ART team hasn't created their PR yet
- Wait for ART PR updating `.ci-operator.yaml`
- Merge that PR, then run this command

### Error: "go mod tidy failed"

Possible causes:
- Kubernetes version doesn't exist yet
- Dependency version incompatibilities
- Network issues downloading dependencies

Solutions:
- Check if K8s version exists: `go list -m -versions k8s.io/api`
- Try with different version parameters
- Reset changes with `git reset --hard` and wait for dependencies to be available

### Image Verification Issues

**"skopeo: command not found"**:
- Install skopeo: `sudo dnf install skopeo` (Fedora/RHEL) or `sudo apt install skopeo` (Ubuntu)
- Or skip verification - images will be marked with *

**"unauthorized: authentication required"**:
- Login to the registry:
  - Red Hat: `podman login registry.access.redhat.com`
  - Quay.io: `podman login quay.io`
  - OpenShift CI: `oc registry login` or check your `~/.docker/config.json`
- Or continue anyway - images will be marked with * for manual verification

**"manifest unknown" or "requested image not found"**:
- The image version doesn't exist yet
- Check the registry manually:
  - UBI: `skopeo list-tags docker://registry.access.redhat.com/ubi9/ubi9-minimal`
  - Operator Framework: `skopeo list-tags docker://quay.io/operator-framework/upstream-registry-builder`
- Either wait for the image to be published or use a different version

**Network/timeout errors**:
- Check your internet connection
- Try again later
- Or continue anyway - images will be marked with *
