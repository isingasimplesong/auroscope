#!/bin/sh
set -eu

IMAGE='archlinux:base-devel@sha256:68bfc3b0d277b08a99101dc9b94aaa03e5ae70cf1b4fb965c03b2b87b915760d'

if ! command -v docker >/dev/null 2>&1; then
  echo 'Docker is required for the package test.' >&2
  exit 69
fi

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
bash "$repo_root/scripts/check-freshness.sh"

docker run --rm -i \
  -v "$repo_root:/package-source:ro" \
  "$IMAGE" /bin/bash <<'BASH'
set -euo pipefail

pacman --noconfirm --needed -Sy git go sudo namcap
useradd --create-home --shell /bin/bash builder
printf 'builder ALL=(ALL:ALL) NOPASSWD: ALL\n' >/etc/sudoers.d/auroscope-package-test
chmod 0440 /etc/sudoers.d/auroscope-package-test

install_test_dependency() {
  local name=$1
  local version=$2
  local body=$3
  local root="/work/$name"
  install -d -m 0755 -o builder -g builder "$root"
  cat >"$root/PKGBUILD" <<EOF
pkgname=$name
pkgver=$version
pkgrel=1
pkgdesc='Disposable AURoscope package-test dependency'
arch=('any')
license=('LicenseRef-Test-Only')
provides=('paru')
package() {
$body
}
EOF
  chown builder:builder "$root/PKGBUILD"
  sudo -u builder -- bash -lc "cd '$root' && makepkg --noconfirm"
  pacman --noconfirm -U "$root"/*.pkg.tar.zst
}

paru_stub='  install -Dm755 /dev/stdin "$pkgdir/usr/bin/paru" <<"SCRIPT"
#!/bin/sh
if [ "${1:-}" = "--version" ]; then
  echo "paru v2.1.0"
  exit 0
fi
printf "%s\\n" "$*"
SCRIPT'

# Stable Paru satisfies the virtual dependency but is the exact package version
# whose broken stdout redirection made interactive search silent in issue #34.
install_test_dependency paru 2.1.0 "$paru_stub"

# Simulate Codex installed outside Pacman (for example through npm).
cat >/usr/local/bin/codex <<'SCRIPT'
#!/bin/sh
if [ "${1:-}" = "--version" ]; then
  echo "codex-cli 0.153.4"
  exit 0
fi
exit 64
SCRIPT
chmod 0755 /usr/local/bin/codex

cp -a /package-source /work/package
chown -R builder:builder /work/package
sudo -u builder -- bash -lc 'cd /work/package && makepkg --printsrcinfo > /tmp/generated.SRCINFO'
cmp /work/package/.SRCINFO /tmp/generated.SRCINFO
namcap /work/package/PKGBUILD
grep -Fx $'\tdepends = paru' /tmp/generated.SRCINFO
grep -Fx $'\tconflicts = paru<=2.1.0' /tmp/generated.SRCINFO
printf '%s\n' 'checking fail-fast remediation for stable Paru 2.1.0'
if sudo -u builder -- bash -lc 'cd /work/package && makepkg --syncdeps --noconfirm' >/tmp/stable-build.out 2>/tmp/stable-build.err; then
  echo 'AURoscope unexpectedly built with incompatible stable Paru installed' >&2
  exit 1
fi
grep -F 'install paru-git first, then rerun makepkg -si' /tmp/stable-build.err
if compgen -G '/work/package/auroscope-*.pkg.tar.zst' >/dev/null; then
  echo 'AURoscope package artifact exists after the incompatible-provider check' >&2
  exit 1
fi
pacman -Q paru
if pacman -Q auroscope >/dev/null 2>&1; then
  echo 'AURoscope remained installed after the incompatible-provider check' >&2
  exit 1
fi

pacman --noconfirm -R paru
install_test_dependency paru-git 2.1.0.r67.g9ac3578 "$paru_stub"
# The pinned archive carries the previous recipe. Build that actual package,
# install it, and leave its archive beside the new recipe, as after git pull.
install -d -m 0755 -o builder -g builder /work/previous
cp /work/package/src/auroscope/packaging/aur/PKGBUILD /work/previous/PKGBUILD
chown builder:builder /work/previous/PKGBUILD
sudo -u builder -- bash -lc 'cd /work/previous && makepkg --syncdeps --noconfirm'
mapfile -t previous_archives < <(sudo -u builder -- bash -lc 'cd /work/previous && makepkg --packagelist')
pacman --noconfirm -U "${previous_archives[@]}"
previous_version=$(pacman -Q auroscope)
cp "${previous_archives[@]}" /work/package/
mapfile -t new_archives < <(sudo -u builder -- bash -lc 'cd /work/package && makepkg --packagelist')
for archive in "${new_archives[@]}"; do test ! -e "$archive"; done
sudo -u builder -- bash -lc 'cd /work/package && makepkg --syncdeps --install --noconfirm'
for archive in "${previous_archives[@]}"; do
  test -f "/work/package/$(basename "$archive")"
done
new_version=$(pacman -Q auroscope)
test "$(vercmp "${new_version#auroscope }" "${previous_version#auroscope }")" -gt 0
printf 'Cached-package upgrade passed: %s -> %s (no --force)\n' "$previous_version" "$new_version"

pacman -Q auroscope
printf '%s\n' 'checking declared runtime dependencies'
pacman -Qi auroscope | grep '^Depends On' | grep -qw 'paru'
missing=$(pacman -T git glibc paru || true)
[ -z "$missing" ] || {
  printf 'unsatisfied package dependencies:\n%s\n' "$missing" >&2
  exit 1
}
if pacman -Qo /usr/local/bin/codex >/dev/null 2>&1; then
  echo 'Codex unexpectedly belongs to a Pacman package' >&2
  exit 1
fi
[ "$(/usr/local/bin/codex --version)" = 'codex-cli 0.153.4' ]

printf '%s\n' 'checking installed package files'
pacman -Ql auroscope | grep -Fx 'auroscope /usr/bin/auroscope'
pacman -Ql auroscope | grep -Fx 'auroscope /usr/share/doc/auroscope/README.md'
stat -c '%A %a %n' /usr/bin/auroscope
test -x /usr/bin/auroscope
printf '%s\n' 'checking native Paru passthrough'
output=$(sudo -u builder -- env HOME=/home/builder /usr/bin/auroscope --version)
printf '%s\n' "$output" | grep -Fx 'paru v2.1.0'

printf '%s\n' 'checking installed guard failure path'
if sudo -u builder -- env HOME=/home/builder /usr/bin/auroscope __guard /does/not/exist >/tmp/guard.out 2>/tmp/guard.err; then
  echo 'guard failure smoke unexpectedly succeeded' >&2
  exit 1
fi
grep -F 'auroscope guard:' /tmp/guard.err

printf '%s\n' 'checking installed review UX with Codex 0.153.4 admission'
cat >/tmp/package-test-codex <<'SCRIPT'
#!/bin/sh
set -eu
if [ "${1:-}" = "--version" ]; then
  echo 'codex-cli 0.153.4'
  exit 0
fi
out=''
while [ "$#" -gt 0 ]; do
  if [ "$1" = '--output-last-message' ]; then
    shift
    out=$1
  fi
  shift || true
done
[ -n "$out" ]
printf '%s' '{"summary":"packaging looks conventional","risk":"low","findings":[],"uncertainty":"","inspect":[]}' >"$out"
SCRIPT
chmod 0755 /tmp/package-test-codex

cat >/tmp/package-test-paru <<'SCRIPT'
#!/bin/sh
set -eu
if [ "${1:-} ${2:-}" = '-P --order' ]; then
  printf '%s\n' 'AUR TARGET hello hello'
  exit 0
fi
if [ "${1:-}" = '-G' ]; then
  mkdir -p "$PWD/$2"
  cd "$PWD/$2"
  git init -q
  git config user.name PackageTest
  git config user.email package-test@example.invalid
  printf 'pkgname=hello\npkgver=1\npkgrel=1\n' >PKGBUILD
  printf 'pkgbase = hello\n' >.SRCINFO
  git add PKGBUILD .SRCINFO
  git commit -qm initial
fi
exit 0
SCRIPT
chmod 0755 /tmp/package-test-paru

rm -rf /tmp/package-test-clones /tmp/package-test-state.sqlite3
review_output=$(printf 'approve\n' | sudo -u builder -- env \
  HOME=/home/builder \
  AUROSCOPE_PARU=/tmp/package-test-paru \
  AUROSCOPE_CODEX=/tmp/package-test-codex \
  AUROSCOPE_STATE=/tmp/package-test-state.sqlite3 \
  AUROSCOPE_CLONE_DIR=/tmp/package-test-clones \
  /usr/bin/auroscope -S hello)
expected_review=$'AURoscope: auditing hello with Codex (timeout 5m0s)...\nAUR audit: hello\n\n---\nAssessment: packaging looks conventional\n\nRisk: low\n---\n\nDecision : [a]pprove | [i]nspect full report | [e]dit and re-audit | [s]kip | [c]ancel '
case "$review_output" in
  *"$expected_review"*) ;;
  *)
    printf 'installed review UX mismatch:\n%s\n' "$review_output" >&2
    exit 1
    ;;
esac

echo 'AURoscope package build/install smoke passed'
BASH
