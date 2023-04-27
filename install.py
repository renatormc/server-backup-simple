from pathlib import Path
import os
import sys

user = sys.argv[1]

APPDIR = Path(os.path.dirname(os.path.realpath(__file__)))

def find_ssh_key():
    for entry in Path(f"/home/{user}/.ssh").iterdir():
        if entry.suffix == ".pub":
            return entry.parent / entry.stem
        
sshkey = find_ssh_key()
if not sshkey or not sshkey.exists():
    print("ssh key not found")
    sys.exit(1)

template = APPDIR / "dist/server-backup-simple.service"
text = template.read_text()
text = text.replace("$USER", user)
text = text.replace("$SSHKEY", str(sshkey))
pathto = Path("/etc/systemd/system/server-backup-simple.service")
pathto.write_text(text)
