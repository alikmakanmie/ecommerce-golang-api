#!/usr/bin/env python3
"""Sync foto produk ecommerce-api (dua arah, aman): asli di VM (master),
salinan web di VPS.

  python3 scripts/sync-uploads.py sync            # pull VPS -> VM, buat salinan web, push VM -> VPS
  python3 scripts/sync-uploads.py ingest <file> [nama]   # masukkan foto baru, sync, cetak URL publik

Alur sync (dua arah, aman):
  1. Tarik dulu foto baru dari VPS (upload via Website/Mobile Admin) ke
     staging, lalu masukkan yang belum dikenal ke uploads-original/ —
     jadi VM selalu menyimpan salinan cadangan penuh.
  2. Buat salinan web (sisi terpanjang maks 1280px, JPEG q82).
  3. Dorong koleksi lengkap ke VPS (rsync --delete-after).

Pengaman (insiden 6 Nov 2026: rsync 255 berulang + domain 530 saat
badai RST egress & VPS 1 vCPU penuh):
  - flock non-blocking: bila sync sebelumnya belum selesai, jalankan
    berikutnya langsung lewati (tidak menumpuk proses rsync).
  - rsync --timeout=20, SSH ConnectTimeout=10,
    ServerAliveInterval=15, ControlMaster=no (tidak gantung).
    (--contimeout tidak berlaku untuk rsync via SSH, hanya daemon.)
  - Bila proxy IPv6 gagal, retry sekali via proxy IPv4 literal.
  - File dummy benchmark (bench-*) tidak pernah ditarik ke VM.
"""
import fcntl
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
STAGING_DIR = BASE / "uploads-from-vps"
LOG = BASE / "sync-uploads.log"
LOCK_FILE = Path("/tmp/photo-sync.lock")

VPS_HOST = os.environ.get("VPS_HOST", "203.175.11.164")
VPS_PORT = os.environ.get("VPS_PORT", "7655")
VPS_USER = os.environ.get("VPS_USER", "root")
VPS_DIR = os.environ.get("VPS_DIR", "/opt/ecommerce-api/uploads")
KEY = Path(os.environ.get("SYNC_KEY", str(Path.home() / ".ssh" / "ecommerce-vps")))
PROXY = Path.home() / "workspace" / "ssh-vps" / "proxy-ssh.py"
PROXY_V4 = Path("/tmp/proxy-ssh-v4.py")

MAX_SIDE = 1280
JPEG_QUALITY = 82
MAX_ORIGINAL_BYTES = 25 * 1024 * 1024
ALLOWED_EXT = {".jpg", ".jpeg", ".png", ".webp", ".gif"}
# File dummy benchmark tidak pernah masuk ke koleksi foto.
EXCLUDE_PATTERNS = ["bench-*"]

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


def _v4_proxy_path():
    """Buat varian proxy IPv4 literal bila belum ada (fallback badai RST)."""
    if PROXY_V4.exists():
        return PROXY_V4
    try:
        text = PROXY.read_text()
        PROXY_V4.write_text(text.replace("hatch-egress-proxy", "198.19.0.1"))
        PROXY_V4.chmod(0o755)  # wajib executable, kalau tidak rsync gagal "Permission denied"
        return PROXY_V4
    except OSError:
        return None


def _ssh_cmd(proxy):
    return (
        f"ssh -i {KEY} -p {VPS_PORT} "
        f"-o ProxyCommand=\"{proxy} %h %p\" "
        "-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null "
        "-o BatchMode=yes -o ConnectTimeout=10 "
        "-o ServerAliveInterval=15 -o ControlMaster=no"
    )


def _run_rsync(args, what):
    """Jalankan rsync; bila gagal, retry sekali via proxy IPv4 literal."""
    proxies = [PROXY]
    v4 = _v4_proxy_path()
    if v4 is not None:
        proxies.append(v4)
    last_err = ""
    for i, proxy in enumerate(proxies):
        cmd = ["rsync"] + args(_ssh_cmd(proxy))
        r = subprocess.run(cmd, capture_output=True, text=True, timeout=300)
        if r.returncode == 0:
            return True
        last_err = (r.stderr or "").strip()[-500:]
        log(f"rsync {what} GAGAL ({r.returncode}) via {proxy.name}: {last_err}")
        if i + 1 < len(proxies):
            log(f"rsync {what}: retry via proxy IPv4 literal...")
    log(f"rsync {what} GAGAL total: {last_err}")
    return False


def pull_from_vps():
    """Tarik foto VPS -> staging (hanya yang baru). Return jumlah file baru."""
    STAGING_DIR.mkdir(parents=True, exist_ok=True)
    # Bersihkan staging dari run sebelumnya agar tidak ada file basi.
    for old in STAGING_DIR.iterdir():
        if old.is_file():
            old.unlink()

    def args(ssh_cmd):
        return [
            "-auz", "--timeout=20",
            *sum([["--exclude", p] for p in EXCLUDE_PATTERNS], []),
            "-e", ssh_cmd,
            f"{VPS_USER}@{VPS_HOST}:{VPS_DIR}/",
            f"{STAGING_DIR}/",
        ]

    if not _run_rsync(args, "pull"):
        return -1
    known_web = {p.name for p in WEB_DIR.iterdir() if p.is_file()} if WEB_DIR.exists() else set()
    known_orig = {web_name(p) for p in ORIG_DIR.iterdir() if p.is_file()} if ORIG_DIR.exists() else set()
    known = known_web | known_orig
    new = 0
    for staged in sorted(STAGING_DIR.iterdir()):
        if not staged.is_file() or staged.suffix.lower() not in ALLOWED_EXT:
            continue
        if staged.name in known or web_name(staged) in known:
            continue
        dest = ORIG_DIR / staged.name
        if dest.exists():
            continue
        shutil.copyfile(staged, dest)
        log(f"pull: foto baru dari VPS -> asli: {staged.name}")
        new += 1
    if new == 0:
        log("pull: tidak ada foto baru dari VPS")
    return new


def rsync_to_vps():
    def args(ssh_cmd):
        return [
            "-az", "--timeout=20", "--delete-after",
            "-e", ssh_cmd,
            f"{WEB_DIR}/",
            f"{VPS_USER}@{VPS_HOST}:{VPS_DIR}/",
        ]

    if _run_rsync(args, "push"):
        log("rsync OK: salinan web sudah di VPS")
        return True
    return False


def do_sync():
    pulled = pull_from_vps()
    if pulled < 0:
        log("sync GAGAL di tahap pull; push dibatalkan agar tidak menimpa VPS dengan data basi")
        return False
    made, kept = build_web_copies()
    log(f"salinan web: {made} dibuat, {kept} sudah terbaru (pull: {pulled} baru)")
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


def acquire_lock():
    """flock non-blocking: return file handle bila dapat kunci, None bila terkunci."""
    f = open(LOCK_FILE, "w")
    try:
        fcntl.flock(f, fcntl.LOCK_EX | fcntl.LOCK_NB)
        return f
    except BlockingIOError:
        f.close()
        return None


if __name__ == "__main__":
    if len(sys.argv) < 2 or sys.argv[1] not in ("sync", "ingest"):
        print(__doc__)
        sys.exit(2)
    lock = acquire_lock()
    if lock is None:
        log("lewati: sync sebelumnya masih jalan (flock terkunci)")
        sys.exit(0)
    try:
        if sys.argv[1] == "sync":
            sys.exit(0 if do_sync() else 1)
        sys.exit(do_ingest(sys.argv[2], sys.argv[3] if len(sys.argv) > 3 else None))
    finally:
        lock.close()
