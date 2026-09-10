"""构建独立 CLI 原生归档包；不执行公开上传或 npm 发布。"""

import argparse
import hashlib
import io
import json
import os
import re
import shutil
import subprocess
import tarfile
import tempfile
import zipfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
TARGETS = [("darwin", "arm64"), ("darwin", "amd64"), ("linux", "arm64"), ("linux", "amd64"), ("windows", "amd64")]


def run(args, env=None):
    return subprocess.check_output(args, cwd=ROOT, env=env, text=True).strip()


def build(version, output, local):
    if not re.fullmatch(r"(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)(?:-[A-Za-z0-9]+(?:[.-][A-Za-z0-9]+)*)?", version):
        raise ValueError("版本必须使用 X.Y.Z 或带预发行后缀的版本号")
    output = output.absolute()
    if output.exists() or output.is_symlink():
        raise ValueError("输出目录已存在，请指定新目录")
    go = shutil.which("go")
    if not go:
        raise ValueError("缺少 Go，请安装维护者工具链")
    module = run([go, "list", "-m"])
    commit, dirty = "unknown", True
    # 独立目录尚未初始化时，不使用父级仓库的 commit 冒充公开源码来源。
    if (ROOT / ".git").exists():
        commit = run(["git", "rev-parse", "HEAD"])
        dirty = bool(run(["git", "status", "--porcelain"]))
    source = commit + ("-dirty" if dirty else "")
    targets = [(run([go, "env", "GOOS"]), run([go, "env", "GOARCH"]))] if local else TARGETS
    if any(target not in TARGETS for target in targets):
        raise ValueError("本机不属于当前构建目标")
    output.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix=".kdl-build-", dir=output.parent) as temporary:
        temporary = Path(temporary)
        staging = temporary / "output"
        staging.mkdir()
        for system, arch in targets:
            print(f"构建 {system}/{arch}", flush=True)
            executable = "kdl-agent.exe" if system == "windows" else "kdl-agent"
            binary = temporary / executable
            env = {**os.environ, "CGO_ENABLED": "0", "GOOS": system, "GOARCH": arch}
            run([go, "build", "-trimpath", "-buildvcs=false", "-ldflags", f"-s -w -X {module}/cmd.Version={version} -X {module}/cmd.Commit={source}", "-o", str(binary), "."], env=env)
            metadata = {"version": version, "commit": commit, "dirty": dirty, "go": run([go, "version"]), "os": system, "arch": arch}
            files = {executable: (binary.read_bytes(), 0o755), "BUILD.json": (json.dumps(metadata, indent=2).encode() + b"\n", 0o644)}
            for name in ["LICENSE", "THIRD_PARTY_NOTICES.txt"]:
                path = ROOT / name
                if path.is_file():
                    files[name] = (path.read_bytes(), 0o644)
            # Go 运行时许可必须匹配本次实际编译器，不能沿用导出机器的工具链。
            files["licenses/Go-LICENSE.txt"] = ((Path(run([go, "env", "GOROOT"])) / "LICENSE").read_bytes(), 0o644)
            extension = "zip" if system == "windows" else "tar.gz"
            archive = staging / f"kdl-agent_{version}_{system}_{arch}.{extension}"
            if system == "windows":
                with zipfile.ZipFile(archive, "w", zipfile.ZIP_DEFLATED) as bundle:
                    for name, (data, mode) in sorted(files.items()):
                        info = zipfile.ZipInfo(name, (1980, 1, 1, 0, 0, 0))
                        info.external_attr = (0o100000 | mode) << 16
                        bundle.writestr(info, data, compress_type=zipfile.ZIP_DEFLATED)
            else:
                with tarfile.open(archive, "w:gz") as bundle:
                    for name, (data, mode) in sorted(files.items()):
                        info = tarfile.TarInfo(name)
                        info.size, info.mode, info.mtime = len(data), mode, 0
                        bundle.addfile(info, io.BytesIO(data))
        sums = [f"{hashlib.sha256(p.read_bytes()).hexdigest()}  {p.name}" for p in sorted(staging.iterdir())]
        (staging / "SHA256SUMS").write_text("\n".join(sums) + "\n", encoding="ascii")
        output.mkdir()
        try:
            shutil.copytree(staging, output, dirs_exist_ok=True)
        except Exception:
            shutil.rmtree(output)
            raise
    print(f"已生成 {len(targets)} 个平台包：{output}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--version", required=True, help="构建版本号")
    parser.add_argument("--output", type=Path, required=True, help="相对 CWD 的新输出目录")
    parser.add_argument("--local", action="store_true", help="仅构建当前平台，用于本地验证")
    args = parser.parse_args()
    try:
        build(args.version, args.output, args.local)
    except (OSError, ValueError, subprocess.CalledProcessError) as exc:
        parser.exit(1, f"构建失败：{exc}\n")


if __name__ == "__main__":
    main()
