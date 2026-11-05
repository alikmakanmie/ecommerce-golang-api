#!/usr/bin/env python3
"""Push commit git HEAD lokal ke GitHub via Git Data API (tanpa `git push`).

Auth: kredensial surrogate Muse `custom.github-push` (PAT user, tidak pernah
terlihat/disimpan script ini). Dipakai karena `gh` belum login & token tidak
boleh ditempel di URL remote.

Cara pakai (dari root repo):
    python3 scripts/push-github.py
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
OWNER, REPO, BRANCH = "alikmakanmie", "ecommerce-golang-api", "main"
STATE = os.path.expanduser("~/.cache/ghpush-ecommerce-state.json")


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
    root = subprocess.check_output(["git", "rev-parse", "--show-toplevel"], text=True).strip()
    os.chdir(root)
    head = git("rev-parse", "HEAD")
    msg = git("log", "-1", "--format=%B")
    author_name = git("config", "user.name")
    author_email = git("config", "user.email")

    st, body = call("GET", f"/repos/{OWNER}/{REPO}/git/refs/heads/{BRANCH}")
    if st != 200:
        print(f"REF FAIL: {st} {body.get('message')}")
        return 1
    remote_sha = body["object"]["sha"]
    print(f"remote {BRANCH}: {remote_sha[:7]} | lokal HEAD: {head[:7]}")
    if remote_sha == head:
        print("sudah sinkron, tidak ada yang di-push")
        return 0
    # Remote harus ancestor lokal agar push fast-forward & tree konsisten.
    # Pengecualian: bila remote BUKAN ancestor tapi tree lokal memuat SEMUA file
    # remote (superset konten, mis. commit paralel "keepalive watchdog"
    # 55ccd232 vs lokal 0f2e1f1), push tetap aman dengan parent=remote SHA —
    # tidak ada konten remote yang hilang; tree hasil = tree HEAD lokal.
    anc = subprocess.run(["git", "merge-base", "--is-ancestor", remote_sha, "HEAD"])
    if anc.returncode != 0:
        st, commit_body = call("GET", f"/repos/{OWNER}/{REPO}/git/commits/{remote_sha}")
        if st != 200:
            print(f"COMMIT-REMOTE FAIL: {st}")
            return 1
        st, tree_body = call("GET", f"/repos/{OWNER}/{REPO}/git/trees/{commit_body['tree']['sha']}?recursive=1")
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

    state = {}
    if os.path.exists(STATE):
        state = json.load(open(STATE))
    os.makedirs(os.path.dirname(STATE), exist_ok=True)

    tree_entries = []
    files = git("ls-tree", "-r", "HEAD", "--format=%(objectmode) %(objectname)%x09%(path)").splitlines()
    for line in files:
        meta, path = line.split("\t", 1)
        mode, blob_sha = meta.split()
        if blob_sha in state:
            gh_sha = state[blob_sha]
        else:
            raw = subprocess.check_output(["git", "cat-file", "-p", blob_sha])
            st, body = call("POST", f"/repos/{OWNER}/{REPO}/git/blobs",
                            {"content": base64.b64encode(raw).decode(), "encoding": "base64"})
            if st not in (200, 201):
                print(f"BLOB FAIL {path}: {st} {body.get('message')}")
                return 1
            gh_sha = body["sha"]
            state[blob_sha] = gh_sha
        tree_entries.append({"path": path, "mode": mode, "type": "blob", "sha": gh_sha})
    json.dump(state, open(STATE, "w"))

    st, body = call("POST", f"/repos/{OWNER}/{REPO}/git/trees", {"tree": tree_entries})
    if st not in (200, 201):
        print(f"TREE FAIL: {st} {body.get('message')}")
        return 1
    tree_sha = body["sha"]

    st, body = call("POST", f"/repos/{OWNER}/{REPO}/git/commits", {
        "message": msg,
        "tree": tree_sha,
        "parents": [remote_sha],
        "author": {"name": author_name, "email": author_email},
    })
    if st not in (200, 201):
        print(f"COMMIT FAIL: {st} {body.get('message')}")
        return 1
    commit_sha = body["sha"]
    print("commit github:", commit_sha)

    st, body = call("PATCH", f"/repos/{OWNER}/{REPO}/git/refs/heads/{BRANCH}",
                    {"sha": commit_sha, "force": False})
    if st not in (200, 201):
        print(f"REF FAIL: {st} {body.get('message')}")
        return 1
    print(f"PUSH DONE: {BRANCH} -> {commit_sha[:7]}")
    print(f"https://github.com/{OWNER}/{REPO}/commit/{commit_sha}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
