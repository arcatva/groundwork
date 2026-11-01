#!/usr/bin/env bash
# Link the groundwork skill into an agent's skills directory, for agents that read SKILL.md skills
# but cannot install this repo as a plugin.
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: install-skills.sh <skills-dir> [--copy]

  <skills-dir>  The directory your agent loads skills from, for example ~/.claude/skills,
                ~/.cline/skills, or a project's .agents/skills. Created if missing.
  --copy        Copy instead of symlink (for agents that do not follow symlinks).

Re-run after `git pull` only when using --copy; a symlink tracks the checkout.
USAGE
}

[ $# -ge 1 ] || { usage >&2; exit 2; }
case "$1" in -h|--help) usage; exit 0 ;; esac

src="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)/skills/groundwork"
dest="${1%/}/groundwork"
mkdir -p "$(dirname -- "$dest")"

if [ -e "$dest" ] || [ -L "$dest" ]; then
  if [ -L "$dest" ] && [ "$(readlink "$dest")" = "$src" ]; then
    echo "already linked: $dest -> $src"; exit 0
  fi
  echo "error: $dest exists and is not this checkout's skill; remove it first" >&2; exit 1
fi

if [ "${2:-}" = "--copy" ]; then
  cp -R "$src" "$dest"; echo "copied: $src -> $dest"
else
  ln -s "$src" "$dest"; echo "linked: $dest -> $src"
fi
