#!/usr/bin/env python3
"""Exercise the actual E2E export with a checkout and a broken linked worktree."""
import os
from pathlib import Path
import shlex
import subprocess
import tempfile
import unittest


class SourceCopyTest(unittest.TestCase):
    def test_checkout_and_linked_worktree(self):
        script = Path(__file__).with_name("e2e-supported-paru.sh").read_text()
        commands = script.split("# Build the candidate AURoscope", 1)[1]
        commands = commands.split("cd /work/auroscope", 1)[0]
        env = dict(os.environ, GOWORK="off", GOPROXY="off", CGO_ENABLED="0")
        with tempfile.TemporaryDirectory(prefix="auroscope-copy-") as temporary:
            root = Path(temporary)
            checkout = root / "checkout"
            checkout.mkdir()

            def run(*args, cwd=checkout, check=True):
                return subprocess.run(args, cwd=cwd, env=env, check=check,
                                      text=True, capture_output=True)

            run("git", "init", "-q")
            (checkout / "go.mod").write_text("module example.invalid/copy\n\ngo 1.23\n")
            (checkout / "main.go").write_text("package main\nfunc main() {}\n")
            (checkout / ".gitignore").write_text("ignored\n")
            run("git", "add", ".")
            run("git", "-c", "user.name=Test", "-c", "user.email=test@example.invalid",
                "-c", "commit.gpgsign=false", "commit", "-qm", "fixture")
            linked = root / "linked worktree"
            run("git", "worktree", "add", "--detach", str(linked))
            self.assertTrue((linked / ".git").is_file())
            # Make its real absolute Git pointer unavailable, as in the container.
            moved = root / "moved checkout"
            checkout.rename(moved)
            negative = run("git", "status", "--porcelain", cwd=linked, check=False)
            self.assertNotEqual(negative.returncode, 0, negative.stdout)
            self.assertIn("not a git repository", negative.stderr)
            for name, source in (("checkout", moved), ("linked", linked)):
                with self.subTest(name=name):
                    (source / "main.go").write_text("package main\nfunc main() { println(42) }\n")
                    (source / "ignored").write_text("ignored local content\n")
                    executable = source / "untracked"
                    executable.write_text("untracked local content\n")
                    executable.chmod(0o755)
                    (source / "link").symlink_to("untracked")
                    nested = source / "fixture" / ".git"
                    nested.mkdir(parents=True)
                    (nested / "keep").write_text("not root metadata\n")
                    destination = root / (name + " export")
                    export = commands.replace("/src", shlex.quote(str(source)))
                    export = export.replace("/work/auroscope", shlex.quote(str(destination)))
                    # First split line is the tail of the heading comment.
                    export = "\n".join(export.splitlines()[1:])
                    run("bash", "-euo", "pipefail", "-c", export, cwd=root)
                    self.assertFalse((destination / ".git").exists())
                    for relative in ("main.go", "ignored", "untracked", "fixture/.git/keep"):
                        self.assertEqual((source / relative).read_bytes(),
                                         (destination / relative).read_bytes())
                    self.assertEqual((destination / "untracked").stat().st_mode & 0o777, 0o755)
                    self.assertEqual(os.readlink(destination / "link"), "untracked")
                    binary = root / (name + "-binary")
                    run("go", "build", "-o", str(binary), ".", cwd=destination)
                    self.assertEqual(run(str(binary), cwd=root).stderr.strip(), "42")


if __name__ == "__main__":
    unittest.main(verbosity=2)
