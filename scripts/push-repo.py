#!/usr/bin/env python3
"""Push commit git HEAD lokal ke GitHub via Git Data API (tanpa `git push`).

Generik: dipakai untuk repo apa pun milik alikmakanmie, termasuk repo BARU
yang belum punya commit sama sekali.

Auth: kredensial surrogate Muse `custom.github-push` (PAT user, tidak pernah
terlihat/disimpan script ini).

Cara pakai (dari root repo yang mau di-push):
    python3 scripts/push-repo.py <nama-repo> [--branch main] [--create]
        [--allow-deletions] [--description "teks"]

  --create           : buat repo publik baru bila belum ada (POST /user/repos)
  --allow-deletions  : izinkan tree lokal MENGHILANGKAN file yang ada di remote
                       (penghapusan disengaja, mis. pindah folder ke repo lain).
                       Tanpa flag ini, push dibatalkan bila ada file remote
                       yang hilang di lokal. Aman: commit baru tetap ber-parent
                       remote HEAD, jadi riwayat tidak hilang.
"""
import base64
import json
import os
import subprocess
import sys
import urllib.request
import urllib.error

sys.path.insert(0, "/opt/muse/skills/custom-connector-setup/bin")
from dynamic_credentials import add_surrogate_to_request, read_json_response

CRED = "custom.github-push"
HOSTS = ["api.github.com"]
API = "https://api.github.com"
OWNER = "alikmakanmie"


def call(method, path, payload=None):
    data = json.dumps(payload).encode() if payload is not None else None
    headers = {"Accept": "application/vnd.github+json",
               "X-GitHub-Api-Version": "2022-11-28",
               "User-Agent": "dhodi-github-push"}
    if payload is not None:
        headers["Content-Type"] = "application/json"
    req = urllib.request.Request(API + path, data=data, headers=headers, method=method)
    add_surrogate_to_request(req, CRED, allowed_hosts=HOSTS)
    try:
        with urllib.request.urlopen(req, timeout=60) as resp:
            return resp.status, read_json_response(resp)
    except urllib.error.HTTPError as e:
        try:
            body = json.loads(e.read().decode("utf-8", "replace"))
        except Exception:
            body = {}
        return e.code, body


def git(*args):
    return subprocess.check_output(["git"] + list(args), text=True).strip()


def main():
    args = sys.argv[1:]
    if not args or args[0].startswith("-"):
        print("pakai: push-repo.py <nama-repo> [--branch main] [--create] [--allow-deletions] [--description ...]")
        return 2
    repo = args[0]
    branch = "main"
    create = False
    allow_deletions = False
    description = ""
    i = 1
    while i < len(args):
        if args[i] == "--branch":
            branch = args[i + 1]
            i += 2
        elif args[i] == "--create":
            create = True
            i += 1
        elif args[i] == "--allow-deletions":
            allow_deletions = True
            i += 1
        elif args[i] == "--description":
            description = args[i + 1]
            i += 2
        else:
            print(f"argumen tidak dikenal: {args[i]}")
            return 2

    root = subprocess.check_output(["git", "rev-parse", "--show-toplevel"], text=True).strip()
    os.chdir(root)
    head = git("rev-parse", "HEAD")
    msg = git("log", "-1", "--format=%B")
    author_name = git("config", "user.name")
    author_email = git("config", "user.email")

    st, body = call("GET", f"/repos/{OWNER}/{repo}")
    if st == 404:
        if not create:
            print(f"repo {OWNER}/{repo} tidak ada (404); pakai --create untuk membuatnya")
            return 1
        st, body = call("POST", "/user/repos", {
            "name": repo, "private": False,
            "description": description or f"{repo} — SLARAS NUSA FOODLAB",
            "auto_init": False,
        })
        if st not in (200, 201):
            print(f"CREATE REPO FAIL: {st} {body.get('message')}")
            return 1
        print(f"repo dibuat: {body.get('html_url')}")
        remote_sha = None
    elif st != 200:
        print(f"REPO FAIL: {st} {body.get('message')}")
        return 1
    else:
        remote_sha = None
        st, body = call("GET", f"/repos/{OWNER}/{repo}/git/refs/heads/{branch}")
        if st == 200:
            remote_sha = body["object"]["sha"]
            print(f"remote {branch}: {remote_sha[:7]} | lokal HEAD: {head[:7]}")
            if remote_sha == head:
                print("sudah sinkron, tidak ada yang di-push")
                return 0
            anc = subprocess.run(["git", "merge-base", "--is-ancestor", remote_sha, "HEAD"],
                                 capture_output=True)
            if anc.returncode != 0:
                if allow_deletions:
                    print("remote bukan ancestor — lanjut (allow-deletions: penghapusan disengaja)")
                else:
                    st, commit_body = call("GET", f"/repos/{OWNER}/{repo}/git/commits/{remote_sha}")
                    if st != 200:
                        print(f"COMMIT-REMOTE FAIL: {st}")
                        return 1
                    st, tree_body = call("GET", f"/repos/{OWNER}/{repo}/git/trees/{commit_body['tree']['sha']}?recursive=1")
                    if st != 200:
                        print(f"TREE-REMOTE FAIL: {st}")
                        return 1
                    remote_paths = {e["path"] for e in tree_body["tree"] if e["type"] == "blob"}
                    local_paths = set(git("ls-tree", "-r", "HEAD", "--name-only").splitlines())
                    missing = sorted(remote_paths - local_paths)
                    if missing:
                        print(f"remote BUKAN ancestor & {len(missing)} file remote hilang di lokal — push dibatalkan: {missing[:5]}")
                        return 1
                    print(f"remote bukan ancestor, tapi tree lokal superset ({len(remote_paths)} file remote semua ada) — lanjut dengan parent=remote")
        elif st == 404:
            print(f"repo ada tapi branch {branch} belum punya commit — commit awal")
            remote_sha = None
        else:
            print(f"REF FAIL: {st} {body.get('message')}")
            return 1

    state_path = os.path.expanduser(f"~/.cache/ghpush-{repo}-state.json")
    state = {}
    if os.path.exists(state_path):
        state = json.load(open(state_path))
    os.makedirs(os.path.dirname(state_path), exist_ok=True)

    tree_entries = []
    files = git("ls-tree", "-r", "HEAD", "--format=%(objectmode) %(objectname)%x09%(path)").splitlines()
    for line in files:
        meta, path = line.split("\t", 1)
        mode, blob_sha = meta.split()
        if blob_sha in state:
            gh_sha = state[blob_sha]
        else:
            raw = subprocess.check_output(["git", "cat-file", "-p", blob_sha])
            st, body = call("POST", f"/repos/{OWNER}/{repo}/git/blobs",
                            {"content": base64.b64encode(raw).decode(), "encoding": "base64"})
            if st not in (200, 201):
                print(f"BLOB FAIL {path}: {st} {body.get('message')}")
                return 1
            gh_sha = body["sha"]
            state[blob_sha] = gh_sha
        tree_entries.append({"path": path, "mode": mode, "type": "blob", "sha": gh_sha})
    json.dump(state, open(state_path, "w"))

    st, body = call("POST", f"/repos/{OWNER}/{repo}/git/trees", {"tree": tree_entries})
    if st not in (200, 201):
        print(f"TREE FAIL: {st} {body.get('message')}")
        return 1
    tree_sha = body["sha"]

    commit_payload = {
        "message": msg,
        "tree": tree_sha,
        "parents": [remote_sha] if remote_sha else [],
        "author": {"name": author_name, "email": author_email},
    }
    st, body = call("POST", f"/repos/{OWNER}/{repo}/git/commits", commit_payload)
    if st not in (200, 201):
        print(f"COMMIT FAIL: {st} {body.get('message')}")
        return 1
    commit_sha = body["sha"]
    print("commit github:", commit_sha)

    if remote_sha:
        st, body = call("PATCH", f"/repos/{OWNER}/{repo}/git/refs/heads/{branch}",
                        {"sha": commit_sha, "force": False})
        action = "PATCH ref"
    else:
        st, body = call("POST", f"/repos/{OWNER}/{repo}/git/refs",
                        {"ref": f"refs/heads/{branch}", "sha": commit_sha})
        action = "CREATE ref"
    if st not in (200, 201):
        print(f"{action} FAIL: {st} {body.get('message')}")
        return 1
    print(f"PUSH DONE: {branch} -> {commit_sha[:7]}")
    print(f"https://github.com/{OWNER}/{repo}/commit/{commit_sha}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
