"""Сохранить проверенный GNU candidate в совместимых native release roles."""

import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import sys

SOURCE = 'dd7a14cc1c88199b5515f5c891100dd72ef211fd'
TARGETS = {'x86_64-pc-windows-gnullvm': 'ktav_cabi-windows-amd64.dll', 'aarch64-pc-windows-gnullvm': 'ktav_cabi-windows-arm64.dll'}


def stage(evidence, output, target):
    if target not in TARGETS or output.exists():
        raise ValueError('reviewed target and new release directory required')
    binary = evidence / 'ktav_cabi.dll'
    if binary.is_symlink() or not binary.is_file() or binary.stat().st_size > 128 * 1024 * 1024:
        raise ValueError('bounded regular candidate DLL required')
    report = json.loads(subprocess.check_output([sys.executable, '-I', str(Path(__file__).with_name('check_embedded_notices.py')), str(binary), str(evidence / 'NOTICE.json')]))
    runtime = json.loads((evidence / 'native-runtime-verification.json').read_bytes())
    source = json.loads((evidence / 'source-provenance.json').read_bytes())
    bundle = json.loads((evidence / 'NOTICE.json').read_bytes())
    if source['source_ref'] != SOURCE or source['target'] != target or runtime['target'] != target or runtime['binary_sha256'] != report['binary_sha256']:
        raise ValueError('source-bound GNU producer identity differs')
    if source['sdk_archive_sha256'] != 'fd5462b7f5d93b2ff981fcd3f9c7e427e47f3203c64ea91c27bcc444eebd74cd' or hashlib.sha256((evidence / 'Cargo.lock').read_bytes()).hexdigest() != 'a7066ecc92727ae12c3905cbf1a2b73266da609773356f725e931682381b7a8b':
        raise ValueError('reviewed SDK or original Go lock differs')
    compiler = bundle['rust_compiler_builtins']
    runtimes = bundle['non_cargo_runtimes']
    if compiler['profile_sha256'] != '39cbf18420d5e0c0a35fb2d5e5fbed4db755488ea0ca11414944ead9fbabab92' or len(compiler['materials']) != 14 or compiler['compiler_builtins_license'] != 'MIT AND Apache-2.0 WITH LLVM-exception':
        raise ValueError('complete reviewed compiler originals required')
    if len(runtimes) != 1 or runtimes[0]['target'] != target or runtimes[0]['catalog_sha256'] != '793c61f930c31a27f987c169eec4a07cc8c140dd16e91653fec3d029f803a816' or len(runtimes[0]['source_files']) != 58 or len(runtimes[0]['original_documents']) != 6:
        raise ValueError('complete reviewed runtime grants required')
    for item in compiler['materials']:
        raw = item['text'].encode('utf-8')
        if hashlib.sha256(raw).hexdigest() != item['sha256'] or len(raw) != item['bytes']:
            raise ValueError('delivered compiler original text differs')
    for item in runtimes[0]['original_documents']:
        if hashlib.sha256(item['text'].encode('utf-8')).hexdigest() != item['sha256']:
            raise ValueError('delivered runtime original text differs')
    output.mkdir(parents=True)
    asset = output / TARGETS[target]
    shutil.copyfile(binary, asset)
    shutil.copyfile(evidence / 'NOTICE.json', output / (asset.name + '.NOTICE.json'))
    manifest = {**report, 'binary': asset.name, 'source_ref': SOURCE, 'tooling_ref': source['tooling_ref'], 'target': target, 'sdk_archive_sha256': source['sdk_archive_sha256'], 'coverage': 'GNU native distribution role; original metadata hook only. Complete runtime and compiler grant texts remain embedded and in the adjacent NOTICE.json. No historical asset repair.'}
    (output / (asset.name + '.NOTICE.manifest.json')).write_text(json.dumps(manifest, sort_keys=True, indent=2) + '\n', encoding='utf-8')
    if asset.read_bytes() != binary.read_bytes():
        raise ValueError('staged release DLL differs')
    print(json.dumps(manifest, sort_keys=True))


if __name__ == '__main__':
    if len(sys.argv) != 4:
        raise SystemExit('usage: stage_go_gnu_release.py EVIDENCE NEW_OUTPUT TARGET')
    stage(Path(sys.argv[1]), Path(sys.argv[2]), sys.argv[3])
