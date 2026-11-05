"""Build an auditable dual-architecture ZIP; preserve Unix execute bits."""
import argparse
import base64
import hashlib
import json
import shutil
import stat
import struct
import uuid
import zipfile
from pathlib import Path


def verify_macho(file: Path, cpu: int):
    data = file.read_bytes()
    magic, actual_cpu = struct.unpack_from('<II', data)
    if magic != 0xFEEDFACF or actual_cpu != cpu:
        raise ValueError(f'Wrong Mach-O architecture: {file}')
    return hashlib.sha256(data).hexdigest()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--root', type=Path, required=True)
    args = parser.parse_args()
    root = args.root.resolve()
    version = json.loads((root / 'extension/package.json').read_text(encoding='utf-8'))['version']
    manifest = json.loads((root / 'extension/.output/chrome-mv3/manifest.json').read_text(encoding='utf-8'))
    if manifest['version'] != version:
        raise ValueError('Build the matching extension first')
    key = (root / 'installer/extension-key.txt').read_text(encoding='utf-8').strip()
    identity = ''.join(chr(ord('a') + int(c, 16)) for c in hashlib.sha256(base64.b64decode(key)).hexdigest()[:32])
    if identity != 'knllhbbekgpmakfppghioeknafdonoao':
        raise ValueError('Stable extension identity changed')
    manifest['key'] = key
    name = f'VideoBridge-{version}-macOS'
    stage = root / '.cache' / f'macos-package-{uuid.uuid4().hex}' / name
    stage.mkdir(parents=True)
    hashes = {}
    for arch, cpu in [('arm64', 0x100000C), ('amd64', 0x1000007)]:
        source = root / 'dist/macos' / f'darwin-{arch}' / 'videobridge'
        hashes[arch] = verify_macho(source, cpu)
        dest = stage / 'host' / f'darwin-{arch}' / 'videobridge'
        dest.parent.mkdir(parents=True)
        shutil.copyfile(source, dest)
    shutil.copytree(root / 'extension/.output/chrome-mv3', stage / 'extension')
    (stage / 'extension/manifest.json').write_text(json.dumps(manifest, ensure_ascii=False, indent=2), encoding='utf-8')
    for source in (root / 'installer/macos').iterdir():
        text = source.read_text(encoding='utf-8').replace('__VERSION__', version)
        (stage / source.name).write_text(text, encoding='utf-8', newline='\n')
    licenses = []
    for dependency, file in [
        ('golang.org/x/sys', root / '.cache/go-mod/golang.org/x/sys@v0.49.0/LICENSE'),
        ('golang.org/x/net', root / '.cache/go-mod/golang.org/x/net@v0.62.0/LICENSE'),
        ('tldts', root / 'extension/node_modules/tldts/LICENSE'),
        ('tldts-core', root / 'extension/node_modules/tldts-core/LICENSE'),
    ]:
        licenses.append(dependency + '\n' + file.read_text(encoding='utf-8'))
    (stage / 'THIRD-PARTY-LICENSES.txt').write_text('\n\n'.join(licenses), encoding='utf-8')
    files = sorted(p for p in stage.rglob('*') if p.is_file())
    checksums = '\n'.join(f'{hashlib.sha256(p.read_bytes()).hexdigest()}  {p.relative_to(stage).as_posix()}' for p in files) + '\n'
    (stage / 'SHA256SUMS').write_text(checksums, encoding='utf-8', newline='\n')
    output = root / 'dist' / f'{name}.zip'
    with zipfile.ZipFile(output, 'w', compression=zipfile.ZIP_DEFLATED, compresslevel=9) as archive:
        for file in sorted(p for p in stage.rglob('*') if p.is_file()):
            relative = file.relative_to(stage).as_posix()
            info = zipfile.ZipInfo(f'{name}/{relative}')
            info.create_system = 3
            mode = 0o755 if file.suffix == '.command' or file.name == 'videobridge' else 0o644
            info.external_attr = (stat.S_IFREG | mode) << 16
            info.compress_type = zipfile.ZIP_DEFLATED
            archive.writestr(info, file.read_bytes())
    # Verify the delivered archive, not just the staging directory.
    with zipfile.ZipFile(output) as archive:
        for line in archive.read(f'{name}/SHA256SUMS').decode().splitlines():
            digest, relative = line.split('  ', 1)
            if hashlib.sha256(archive.read(f'{name}/{relative}')).hexdigest() != digest:
                raise ValueError(f'ZIP checksum mismatch: {relative}')
        for arch in hashes:
            item = archive.getinfo(f'{name}/host/darwin-{arch}/videobridge')
            if not (item.external_attr >> 16) & stat.S_IXUSR:
                raise ValueError('Host executable mode lost')
        for script in ['Install.command', 'Uninstall.command', 'Build-Pkg.command']:
            if not (archive.getinfo(f'{name}/{script}').external_attr >> 16) & stat.S_IXUSR:
                raise ValueError('Installer executable mode lost')
    digest = hashlib.sha256(output.read_bytes()).hexdigest()
    output.with_suffix('.zip.sha256').write_text(f'{digest}  {output.name}\n', encoding='utf-8')
    (root / 'dist/macos-package-info.json').write_text(json.dumps({
        'version': version, 'extensionId': identity, 'sha256': digest,
        'hosts': hashes, 'staging': str(stage), 'macRuntimeTested': False,
    }, indent=2), encoding='utf-8')
    print(f'PASS Mach-O arm64/amd64, stable extension ID, every ZIP checksum, executable permissions\n{output}\nSHA256 {digest}')


if __name__ == '__main__':
    main()
