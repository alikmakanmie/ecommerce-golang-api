#!/usr/bin/env python3
"""Sync foto produk ecommerce-api: asli di VM (master), salinan web di VPS.

  python3 scripts/sync-uploads.py sync            # buat salinan web + rsync ke VPS
  python3 scripts/sync-uploads.py ingest <file> [nama]   # masukkan foto baru, sync, cetak URL publik

Salinan web: sisi terpanjang maks 1280px, JPEG q82 / PNG & WebP dioptimasi.
VPS disinkron dengan rsync --delete lewat kunci SSH khusus (tanpa password).
"""
import os
import re
import shutil
import subprocess
import sys
import time
from pathlib import Path

BASE = Path(__file__).resolve().parent.parent
ORIG_DIR = BASE / "uploads-original"
WEB_DIR = BASE / "uploads"
LOG = BASE / "sync-uploads.log"

VPS_HOST = os.environ.get("VPS_HOST", "203.175.11.164")
VPS_PORT = os.environ.get("VPS_PORT", "7655")
VPS_USER = os.environ.get("VPS_USER", "root")
VPS_DIR = os.environ.get("VPS_DIR", "/opt/ecommerce-api/uploads")
KEY = Path(os.environ.get("SYNC_KEY", str(Path.home() / ".ssh" / "ecommerce-vps")))
PROXY = Path.home() / "workspace" / "ssh-vps" / "proxy-ssh.py"

MAX_SIDE = 1280
JPEG_QUALITY = 82
MAX_ORIGINAL_BYTES = 25 * 1024 * 1024
ALLOWED_EXT = {".jpg", ".jpeg", ".png", ".webp", ".gif"}

PUBLIC_IP_URL = "http://203.175.11.164:2150/uploads"
PUBLIC_DOMAIN_URL = "https://api.alikmakanmie.my.id/uploads"


def log(msg):
    line = f"{time.strftime('%Y-%m-%d %H:%M:%S')} {msg}"
    print(line, flush=True)
    with open(LOG, "a") as f:
        f.write(line + "\n")


def slugify(name):
    stem = Path(name).stem.lower()
    stem = re.sub(r"[^a-z0-9]+", "-", stem).strip("-") or "foto"
    return stem[:80]


def web_name(orig_path):
    ext = orig_path.suffix.lower()
    if ext == ".jpeg":
        ext = ".jpg"
    return f"{slugify(orig_path.name)}{ext}"


def make_web_copy(orig_path, web_path):
    """Buat salinan web dari file asli. Return True bila dibuat/diubah."""
    if web_path.exists() and web_path.stat().st_mtime >= orig_path.stat().st_mtime:
        return False
    from PIL import Image

    ext = orig_path.suffix.lower()
    if ext == ".gif":
        shutil.copyfile(orig_path, web_path)  # GIF disalin apa adanya (animasi)
        return True
    with Image.open(orig_path) as im:
        im.thumbnail((MAX_SIDE, MAX_SIDE), Image.LANCZOS)
        if ext in (".jpg", ".jpeg"):
            if im.mode in ("RGBA", "LA", "P"):
                im = im.convert("RGB")
            im.save(web_path, "JPEG", quality=JPEG_QUALITY, optimize=True)
        elif ext == ".png":
            im.save(web_path, "PNG", optimize=True)
        elif ext == ".webp":
            im.save(web_path, "WEBP", quality=JPEG_QUALITY, method=6)
        else:
            raise ValueError(f"ekstensi tak didukung: {ext}")
    # samakan mtime dengan aslinya agar skip berikutnya akurat
    st = orig_path.stat()
    os.utime(web_path, (st.st_atime, st.st_mtime))
    return True


def build_web_copies():
    ORIG_DIR.mkdir(parents=True, exist_ok=True)
    WEB_DIR.mkdir(parents=True, exist_ok=True)
    made, kept = 0, 0
    for orig in sorted(ORIG_DIR.iterdir()):
        if not orig.is_file() or orig.suffix.lower() not in ALLOWED_EXT:
            continue
        if orig.stat().st_size > MAX_ORIGINAL_BYTES:
            log(f"LEWAT (>{MAX_ORIGINAL_BYTES // 1024 // 1024}MB): {orig.name}")
            continue
        web_path = WEB_DIR / web_name(orig)
        if make_web_copy(orig, web_path):
            made += 1
            log(f"web: {orig.name} -> {web_path.name} ({web_path.stat().st_size // 1024} KB)")
        else:
            kept += 1
    # hapus salinan web yang aslinya sudah tidak ada
    orig_names = {web_name(p) for p in ORIG_DIR.iterdir() if p.is_file()}
    for web in list(WEB_DIR.iterdir()):
        if web.is_file() and web.name not in orig_names:
            web.unlink()
            log(f"hapus salinan yatim: {web.name}")
    return made, kept


def rsync_to_vps():
    ssh_cmd = (
        f"ssh -i {KEY} -p {VPS_PORT} "
        f"-o ProxyCommand=\"{PROXY} %h %p\" "
        "-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null "
        "-o BatchMode=yes -o ConnectTimeout=20"
    )
    cmd = [
        "rsync", "-az", "--delete",
        "-e", ssh_cmd,
        f"{WEB_DIR}/",
        f"{VPS_USER}@{VPS_HOST}:{VPS_DIR}/",
    ]
    r = subprocess.run(cmd, capture_output=True, text=True, timeout=300)
    if r.returncode != 0:
        log(f"rsync GAGAL ({r.returncode}): {r.stderr.strip()[-500:]}")
        return False
    log("rsync OK: salinan web sudah di VPS")
    return True


def do_sync():
    made, kept = build_web_copies()
    log(f"salinan web: {made} dibuat, {kept} sudah terbaru")
    return rsync_to_vps()


def do_ingest(src, name=None):
    src = Path(src)
    if not src.is_file():
        print(f"file tidak ada: {src}", file=sys.stderr)
        return 1
    if src.suffix.lower() not in ALLOWED_EXT:
        print(f"bukan gambar (jpg/png/webp/gif): {src}", file=sys.stderr)
        return 1
    if src.stat().st_size > MAX_ORIGINAL_BYTES:
        print("file >25MB, tolak", file=sys.stderr)
        return 1
    ORIG_DIR.mkdir(parents=True, exist_ok=True)
    base = slugify(name or src.name)
    stamp = time.strftime("%Y%m%d-%H%M%S")
    dest = ORIG_DIR / f"{stamp}-{base}{src.suffix.lower()}"
    shutil.copyfile(src, dest)
    log(f"asli disimpan: {dest.name} ({dest.stat().st_size // 1024} KB)")
    if not do_sync():
        return 1
    web = web_name(dest)
    print(f"URL publik : {PUBLIC_IP_URL}/{web}")
    print(f"URL domain : {PUBLIC_DOMAIN_URL}/{web}  (aktif setelah domain live)")
    return 0


if __name__ == "__main__":
    if len(sys.argv) < 2 or sys.argv[1] not in ("sync", "ingest"):
        print(__doc__)
        sys.exit(2)
    if sys.argv[1] == "sync":
        sys.exit(0 if do_sync() else 1)
    sys.exit(do_ingest(sys.argv[2], sys.argv[3] if len(sys.argv) > 3 else None))
