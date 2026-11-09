"""Snapshot all generated sources, regenerate, then compare (also works before first commit)."""
import hashlib,pathlib,subprocess,sys
def snapshot():
    files=list(pathlib.Path('internal/api').rglob('*.gen.go'))+list(pathlib.Path('internal/store').glob('*.go'))+list(pathlib.Path('contracts').glob('*.json'))
    return {str(p):hashlib.sha256(p.read_bytes()).hexdigest() for p in files}
before=snapshot()
subprocess.run(['powershell','-NoProfile','-File','scripts/generate.ps1'] if sys.platform=='win32' else ['make','generate'],check=True)
after=snapshot()
if before!=after:
    raise SystemExit('Generation differs: '+', '.join(p for p in before.keys()|after.keys() if before.get(p)!=after.get(p)))
print('Generation reproducible: '+str(len(after))+' files')
