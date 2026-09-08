#!/bin/sh
set -eu

IMAGE='archlinux:base-devel@sha256:68bfc3b0d277b08a99101dc9b94aaa03e5ae70cf1b4fb965c03b2b87b915760d'
PARU_COMMIT='9ac3578807a87858651e81a02586ceb947686e7c'

if [ "${AUROSCOPE_E2E_SUPPORTED_PARU:-}" != 1 ]; then
  echo 'Refusing to run without AUROSCOPE_E2E_SUPPORTED_PARU=1.' >&2
  exit 64
fi
if ! command -v docker >/dev/null 2>&1; then
  echo 'Docker is required for the supported-Paru E2E.' >&2
  exit 69
fi

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

docker run --rm -i \
  -e AUROSCOPE_E2E_INSIDE=1 \
  -e AUROSCOPE_E2E_SELECTION_ONLY="${AUROSCOPE_E2E_SELECTION_ONLY:-0}" \
  -e AUROSCOPE_E2E_SOURCE_ONLY="${AUROSCOPE_E2E_SOURCE_ONLY:-0}" \
  -e PARU_COMMIT="$PARU_COMMIT" \
  -v "$repo_root:/src:ro" \
  "$IMAGE" /bin/bash <<'BASH'
set -euo pipefail

[[ "${AUROSCOPE_E2E_INSIDE:-}" == 1 ]]
[[ "$PWD" == / ]]

# All package-manager activity is confined to this disposable container.
pacman --noconfirm --needed -Sy git cargo go sudo
useradd --create-home --shell /bin/bash builder
printf 'builder ALL=(ALL:ALL) NOPASSWD: ALL\n' >/etc/sudoers.d/auroscope-e2e
chmod 0440 /etc/sudoers.d/auroscope-e2e

# Build the exact issue-21 Paru surface against this image's libalpm 16.
git clone --quiet https://github.com/Morganamilo/paru.git /work/paru
cd /work/paru
git checkout --quiet "$PARU_COMMIT"
cargo update alpm alpm-utils
cargo build --release
install -m 0755 target/release/paru /usr/local/bin/paru-real

# Build the candidate AURoscope with the pinned CGO SQLite driver.
cp -a /src /work/auroscope
cd /work/auroscope
GOCACHE=/tmp/gocache GOMODCACHE=/tmp/gomodcache CGO_ENABLED=1 go build -o /usr/local/bin/auroscope ./cmd/auroscope

# Exercise actual color detection and uncolored target capture, including the
# user's disabled Color setting and redirected output. No recipe is executed.
AUROSCOPE_TEST_REAL_PARU=/usr/local/bin/paru-real \
  GOCACHE=/tmp/gocache GOMODCACHE=/tmp/gomodcache CGO_ENABLED=1 \
  go test ./internal/app -run '^TestSelectionRealParuColors$' -count=1 -v -timeout=120s

if [[ "${AUROSCOPE_E2E_SELECTION_ONLY:-0}" == 1 ]]; then
  echo 'supported Paru selection-only gate passed (no PKGBUILD execution)'
  exit 0
fi

# Install the pinned recipe, not the independently built source binary. Package
# the real compiled Paru as a disposable provider so Pacman checks dependencies.
# Source-only checks exercise unpublished worktree fixes without fetching the
# immutable package pin. They do not qualify the packaged artifact.
if [[ "${AUROSCOPE_E2E_SOURCE_ONLY:-0}" != 1 ]]; then
install -d -m 0755 -o builder -g builder /work/paru-provider
cat >/work/paru-provider/PKGBUILD <<'EOF'
pkgname=paru-git
pkgver=2.1.0.r67.g9ac3578
pkgrel=1
pkgdesc='Disposable pinned real Paru provider'
arch=('x86_64')
license=('GPL-3.0-only')
provides=('paru')
package() {
  install -Dm755 /usr/local/bin/paru-real "$pkgdir/usr/bin/paru"
}
EOF
chown builder:builder /work/paru-provider/PKGBUILD
sudo -u builder -- bash -lc 'cd /work/paru-provider && makepkg --noconfirm'
pacman --noconfirm -U /work/paru-provider/*.pkg.tar.zst
chown -R builder:builder /work/auroscope/packaging/aur
sudo -u builder -- bash -lc 'cd /work/auroscope/packaging/aur && makepkg --syncdeps --noconfirm'
pacman --noconfirm -U /work/auroscope/packaging/aur/auroscope-*.pkg.tar.zst
pacman -Q auroscope
pacman -Qo /usr/bin/auroscope
rm /usr/local/bin/auroscope
ln -s /usr/bin/auroscope /usr/local/bin/auroscope
AUROSCOPE_TEST_REAL_PARU=/usr/local/bin/paru-real \
  AUROSCOPE_TEST_BINARY=/usr/bin/auroscope \
  GOCACHE=/tmp/gocache GOMODCACHE=/tmp/gomodcache CGO_ENABLED=1 \
  go test ./internal/app -run '^TestSelectionRealParuColors$' -count=1 -v -timeout=120s
fi

cat >/usr/local/bin/paru <<'EOF'
#!/bin/sh
printf '%s\n' "$*" >>/tmp/paru-calls
exec /usr/local/bin/paru-real "$@"
EOF
chmod 0755 /usr/local/bin/paru

cat >/usr/local/bin/codex <<'EOF'
#!/bin/sh
set -eu
if [ "${1:-}" = --version ]; then
  echo 'codex-cli 0.150.1'
  exit 0
fi
out=''
while [ "$#" -gt 0 ]; do
  if [ "$1" = --output-last-message ]; then
    shift
    out=$1
  fi
  shift || true
done
[ -n "$out" ]
[ "$PWD" != /home/builder/aur/hello ]
[ -f bundle.json ]
printf 'audit\n' >>/tmp/codex-calls
if [ "${AUROSCOPE_E2E_DRIFT:-}" = 1 ] ||
   { [ "${AUROSCOPE_E2E_DRIFT:-}" = once ] && [ ! -f /tmp/auroscope-drift-once ]; }; then
  printf '\n# post-audit drift\n' >>/home/builder/aur/hello/PKGBUILD
  git -C /home/builder/aur/hello -c user.name=E2E -c user.email=e2e@example.invalid add PKGBUILD
  git -C /home/builder/aur/hello -c user.name=E2E -c user.email=e2e@example.invalid commit -m drift >/dev/null
  touch /tmp/auroscope-drift-once
fi
printf '%s' '{"summary":"supported-Paru E2E audit","risk":"low","findings":[],"uncertainty":"","inspect":["PKGBUILD"]}' >"$out"
EOF
chmod 0755 /usr/local/bin/codex

cat >/usr/local/bin/auroscope-e2e-editor <<'EOF'
#!/bin/sh
set -eu
printf '\n# reviewed local edit\n' >>"$1/PKGBUILD"
EOF
chmod 0755 /usr/local/bin/auroscope-e2e-editor

install -d -m 0700 -o builder -g builder /home/builder/aur /home/builder/state
: >/tmp/paru-calls
: >/tmp/codex-calls
chmod 0666 /tmp/paru-calls /tmp/codex-calls

run_as_builder() {
  sudo -u builder -- env \
    HOME=/home/builder \
    PATH=/usr/local/bin:/usr/bin \
    AUROSCOPE_PARU=/usr/local/bin/paru \
    AUROSCOPE_CODEX=/usr/local/bin/codex \
    AUROSCOPE_CLONE_DIR=/home/builder/aur \
    AUROSCOPE_STATE=/home/builder/state/state.sqlite3 \
    "$@"
}

# Real native search/selection, resolution, -G acquisition, Paru config/hook,
# makepkg, and Pacman. EOF accepts Paru/Pacman's native default confirmations.
{ printf '1\n'; sleep 2; printf 'approve\n'; } | run_as_builder /usr/local/bin/auroscope hello
pacman -Q hello
first_codex_count=$(wc -l </tmp/codex-calls)
[ "$first_codex_count" -eq 1 ]
grep -q -- '-Ssq --interactive hello' /tmp/paru-calls
grep -q -- '-P --order hello' /tmp/paru-calls
grep -q -- '-G hello' /tmp/paru-calls
grep -q -- '-S --skipreview -- hello' /tmp/paru-calls

# Real edit snapshots the cache worktree as a local commit, triggers a second
# audit, and reaches the same Paru worktree/guard before rebuilding.
before_edit_codex=$(wc -l </tmp/codex-calls)
printf 'edit\napprove\n' | sudo -u builder -- env \
  HOME=/home/builder PATH=/usr/local/bin:/usr/bin \
  EDITOR=/usr/local/bin/auroscope-e2e-editor \
  AUROSCOPE_PARU=/usr/local/bin/paru \
  AUROSCOPE_CODEX=/usr/local/bin/codex \
  AUROSCOPE_CLONE_DIR=/home/builder/aur \
  AUROSCOPE_STATE=/home/builder/state/state.sqlite3 \
  /usr/local/bin/auroscope -S --noconfirm --rebuild hello
after_edit_codex=$(wc -l </tmp/codex-calls)
[ "$after_edit_codex" -eq $((before_edit_codex + 2)) ]
sudo -u builder -- git -C /home/builder/aur/hello log -1 --format=%s | grep -q 'AURoscope reviewed edit'
grep -q 'reviewed local edit' /home/builder/aur/hello/PKGBUILD

# A committed mutation after bundle construction must fail at the real hook.
before_drift=$(grep -Fxc -- '-S --noconfirm --rebuild --skipreview -- hello' /tmp/paru-calls || true)
if printf 'approve\n' | sudo -u builder -- env \
  HOME=/home/builder PATH=/usr/local/bin:/usr/bin \
  AUROSCOPE_PARU=/usr/local/bin/paru \
  AUROSCOPE_CODEX=/usr/local/bin/codex \
  AUROSCOPE_CLONE_DIR=/home/builder/aur \
  AUROSCOPE_STATE=/home/builder/state/state.sqlite3 \
  AUROSCOPE_E2E_DRIFT=1 \
  /usr/local/bin/auroscope -S --noconfirm --rebuild hello >/tmp/drift-output 2>&1; then
  cat /tmp/drift-output
  echo 'guard drift scenario unexpectedly succeeded' >&2
  exit 1
fi
cat /tmp/drift-output
after_drift=$(grep -Fxc -- '-S --noconfirm --rebuild --skipreview -- hello' /tmp/paru-calls || true)
if [ "$after_drift" -ne $((before_drift + 1)) ]; then
  echo 'guard drift scenario did not reach a new final --skipreview handoff' >&2
  exit 1
fi
if ! grep -Exq 'auroscope guard: recipe identity drift for hello: got [0-9a-f]{40}/[0-9a-f]{64} want [0-9a-f]{40}/[0-9a-f]{64}' /tmp/drift-output; then
  echo 'guard drift scenario failed without the exact identity-refusal diagnostic' >&2
  exit 1
fi
echo 'guard drift scenario: new final handoff and exact identity refusal verified'

# The candidate must recover from the real hook refusal only after an explicit
# retry and a second Codex audit/approval. Keep this source gate separate from
# the immutable packaged artifact until publication advances its source pin.
if [[ "${AUROSCOPE_E2E_SOURCE_ONLY:-0}" == 1 ]]; then
  rm -f /tmp/auroscope-drift-once
  before_retry_codex=$(wc -l </tmp/codex-calls)
  printf 'approve\nretry\napprove\n' | run_as_builder env \
    AUROSCOPE_E2E_DRIFT=once \
    /usr/local/bin/auroscope -S --noconfirm --rebuild hello \
    2>&1 | tee /tmp/drift-retry-output
  after_retry_codex=$(wc -l </tmp/codex-calls)
  [ "$after_retry_codex" -eq $((before_retry_codex + 2)) ]
  grep -Fq 'the recipe for hello changed after approval' /tmp/drift-retry-output
  grep -Fq 'Decision : [r]etry | [s]kip | [c]ancel' /tmp/drift-retry-output
  pacman -Q hello
fi

# Skipping the only AUR target performs no final build resolution.
before=$(grep -c -- '--skipreview' /tmp/paru-calls || true)
printf 'skip\n' | run_as_builder /usr/local/bin/auroscope -S --noconfirm hello
after=$(grep -c -- '--skipreview' /tmp/paru-calls || true)
[ "$before" -eq "$after" ]

# Selecting an official package installs the resolved target, not the search
# terms again. Answer Pacman's confirmation explicitly: EOF cancels this path.
# Wait for the actual prompt so Paru selection cannot buffer the later answer.
before_codex=$(wc -l </tmp/codex-calls)
run_as_builder python - <<'PY'
import os
import select
import subprocess
import sys
import time

child = subprocess.Popen(['/usr/local/bin/auroscope', 'tree'],
                         stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                         stderr=subprocess.STDOUT)
child.stdin.write(b'1\n')
child.stdin.flush()
output = b''
answered = False
deadline = time.monotonic() + 120
try:
    while True:
        if time.monotonic() >= deadline:
            raise TimeoutError('official selection/install prompt timed out')
        if not select.select([child.stdout], [], [], 1)[0]:
            continue
        chunk = os.read(child.stdout.fileno(), 65536)
        if not chunk:
            break
        sys.stdout.buffer.write(chunk)
        sys.stdout.buffer.flush()
        output += chunk
        if not answered and b'Proceed with installation?' in output:
            child.stdin.write(b'y\n')
            child.stdin.flush()
            answered = True
    assert answered, 'native installation confirmation was not reached'
    assert child.wait(timeout=10) == 0, 'official installation failed'
finally:
    if child.poll() is None:
        child.kill()
        child.wait()
PY
pacman -Q tree
grep -Fx -- '-S -- tree' /tmp/paru-calls
after_codex=$(wc -l </tmp/codex-calls)
[ "$before_codex" -eq "$after_codex" ]

# An explicit official-only install also remains native without Codex.
before_codex=$(wc -l </tmp/codex-calls)
run_as_builder /usr/local/bin/auroscope -S --repo --noconfirm tree
pacman -Q tree
after_codex=$(wc -l </tmp/codex-calls)
[ "$before_codex" -eq "$after_codex" ]

echo 'supported Paru disposable-Arch E2E passed'
BASH
