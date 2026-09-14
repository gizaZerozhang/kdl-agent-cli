"""为已构建的原生包补齐同版本 Skill、OpenAPI、文档及校验清单。"""

import argparse
import hashlib
import json
import re
import shutil
import subprocess
import tarfile
import zipfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def identity():
    package = json.loads((ROOT / "package.json").read_text())
    if not re.fullmatch(r"@[a-z0-9._-]+/[a-z0-9._-]+", package["name"]) or "pending" in package["name"]:
        raise ValueError("npm 包名未绑定")
    if subprocess.check_output(["git", "status", "--porcelain"], cwd=ROOT, text=True).strip():
        raise ValueError("发行需要干净源码")
    commit = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip()
    repository = package["repository"]["url"].removeprefix("git+https://github.com/").removesuffix(".git")
    for file in [ROOT / "README.md", *ROOT.glob("docs/*.md"), ROOT / "skills/kdl-agent/SKILL.md"]:
        text = file.read_text()
        if "@@" in text or "https://example.com/" in text or "/Users/" in text:
            raise ValueError(f"公开文档含占位内容：{file.name}")
    return {"version": package["version"], "commit": commit, "repository": repository, "dirty": False}


def preflight():
    record = identity()
    subprocess.run(["node", "scripts/release-policy.cjs", "v" + record["version"]], cwd=ROOT, check=True)
    directory = ROOT / ".release"
    directory.mkdir(exist_ok=True)
    (directory / "BUILD.json").write_text(json.dumps(record, indent=2) + "\n")
    print(f"源码预检通过：{record['version']} {record['commit']}")


def prepare(source, output):
    record = identity()
    package = json.loads((ROOT / "package.json").read_text())
    version = package["version"]
    if output.exists():
        raise ValueError("候选目录已存在，拒绝覆盖")
    archives = []
    for system, arch in [("darwin", "arm64"), ("darwin", "amd64"), ("linux", "arm64"), ("linux", "amd64"), ("windows", "amd64")]:
        suffix = "zip" if system == "windows" else "tar.gz"
        archive = source / f"kdl-agent_{version}_{system}_{arch}.{suffix}"
        if suffix == "zip":
            with zipfile.ZipFile(archive) as bundle:
                embedded = json.loads(bundle.read("BUILD.json"))
        else:
            with tarfile.open(archive, "r:gz") as bundle:
                embedded = json.load(bundle.extractfile("BUILD.json"))
        if embedded != record:
            raise ValueError(f"归档源码记录不一致：{archive.name}")
        archives.append(archive)
    output.mkdir(parents=True)
    for archive in archives:
        shutil.copyfile(archive, output / archive.name)
    (output / "BUILD.json").write_text(json.dumps(record, indent=2) + "\n")
    skill = ROOT / "skills/kdl-agent/SKILL.md"
    if version not in skill.read_text() or version not in (ROOT / "docs/install.md").read_text():
        raise ValueError("Skill 或安装文档版本未同步")
    with zipfile.ZipFile(output / "kdl-agent-skill.zip", "x", zipfile.ZIP_DEFLATED) as bundle:
        info = zipfile.ZipInfo("kdl-agent/SKILL.md", (1980, 1, 1, 0, 0, 0))
        info.external_attr = 0o100644 << 16
        bundle.writestr(info, skill.read_bytes(), compress_type=zipfile.ZIP_DEFLATED)
    for name in ["openapi.yaml", "install.md", "cli-guide.md", "api-guide.md"]:
        destination = output / name
        with destination.open("xb") as stream:
            stream.write((ROOT / "docs" / name).read_bytes())
    if "-beta." not in version:
        subprocess.run(["node", "scripts/release-policy.cjs", "v" + version], cwd=ROOT, check=True)
        shutil.copyfile(ROOT / "release/stable-acceptance.json", output / "stable-acceptance.json")
    (output / "release-notes.md").write_text(
        f"# 快代理 CLI {version}\n\n"
        + f"源码：{subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip()}\n\n"
        + "npm 安装器按平台下载并核对 SHA256；安装向导配套同版本 Skill。\n\n"
        + ("此为预发行验证版本，macOS 未签名/公证；未声明最低系统兼容实测结论，真实业务以验收记录为准；不作为生产业务就绪声明。\n" if "-beta." in version else "本版 macOS 程序未做 Developer ID 签名及 Apple 公证；系统可能阻止运行，未声明最低系统实测结论。发行验证记录见同版本流水线。\n"),
        encoding="utf-8",
    )
    names = sorted(path for path in output.iterdir() if path.is_file() and path.name != "SHA256SUMS")
    (output / "SHA256SUMS").write_text(
        "".join(f"{hashlib.sha256(path.read_bytes()).hexdigest()}  {path.name}\n" for path in names), encoding="ascii"
    )


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    sub.add_parser("preflight")
    collect = sub.add_parser("collect")
    collect.add_argument("source", type=Path)
    collect.add_argument("output", type=Path)
    args = parser.parse_args()
    if args.command == "preflight":
        preflight()
    else:
        prepare(args.source, args.output)
