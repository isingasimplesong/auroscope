#!/bin/sh
set -eu

IMAGE='archlinux:base-devel@sha256:68bfc3b0d277b08a99101dc9b94aaa03e5ae70cf1b4fb965c03b2b87b915760d'

if ! command -v docker >/dev/null 2>&1; then
  echo 'Docker is required for the package test.' >&2
  exit 69
fi

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

docker run --rm -i \
  -v "$repo_root:/package-source:ro" \
  "$IMAGE" /bin/bash <<'BASH'
set -euo pipefail

pacman --noconfirm --needed -Sy git go sudo
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
package() {
$body
}
EOF
  chown builder:builder "$root/PKGBUILD"
  sudo -u builder -- bash -lc "cd '$root' && makepkg --noconfirm"
  pacman --noconfirm -U "$root"/*.pkg.tar.zst
}

install_test_dependency paru 2.1.0 '  install -Dm755 /dev/stdin "$pkgdir/usr/bin/paru" <<"SCRIPT"
#!/bin/sh
if [ "${1:-}" = "--version" ]; then
  echo "paru v2.1.0"
  exit 0
fi
printf "%s\\n" "$*"
SCRIPT'

# Simulate Codex installed outside Pacman (for example through npm).
cat >/usr/local/bin/codex <<'SCRIPT'
#!/bin/sh
if [ "${1:-}" = "--version" ]; then
  echo "codex-cli 0.151.0"
  exit 0
fi
exit 64
SCRIPT
chmod 0755 /usr/local/bin/codex

cp -a /package-source /work/package
chown -R builder:builder /work/package
sudo -u builder -- bash -lc 'cd /work/package && makepkg --syncdeps --noconfirm'
pacman --noconfirm -U /work/package/auroscope-*.pkg.tar.zst

pacman -Q auroscope
printf '%s\n' 'checking declared runtime dependencies'
missing=$(pacman -T git glibc 'paru>=2.1.0' || true)
[ -z "$missing" ] || {
  printf 'unsatisfied package dependencies:\n%s\n' "$missing" >&2
  exit 1
}
if pacman -Qo /usr/local/bin/codex >/dev/null 2>&1; then
  echo 'Codex unexpectedly belongs to a Pacman package' >&2
  exit 1
fi
[ "$(/usr/local/bin/codex --version)" = 'codex-cli 0.151.0' ]

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

echo 'AURoscope package build/install smoke passed'
BASH
