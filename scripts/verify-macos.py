"""在 macOS 上验证固定归档的 Developer ID、签名、公证并绑定校验清单。"""

import argparse
import hashlib
import json
import os
import re
import subprocess
import sys
import tarfile
import tempfile
from pathlib import Path


def sha256(file):
    return hashlib.sha256(file.read_bytes()).hexdigest()


def run(args):
    result = subprocess.run(args, capture_output=True, text=True, timeout=120)
    if result.returncode != 0:
        # 外部工具输出不进入异常，避免泄露环境或凭据材料。
        raise ValueError(f"macOS 验证失败：{Path(args[0]).name}")
    return result.stdout + result.stderr


def verify_signature(binary, team, execute=run):
    execute(["codesign", "--verify", "--strict", "--verbose=2", str(binary)])
    detail = execute(["codesign", "--display", "--verbose=4", str(binary)])
    if f"TeamIdentifier={team}" not in detail.splitlines():
        raise ValueError("签名 TeamIdentifier 与公司配置不一致")
    if not any(line.startswith("Authority=Developer ID Application:") for line in detail.splitlines()):
        raise ValueError("需要 Developer ID Application 签名")
    if not re.search(r"flags=.*\bruntime\b", detail) or "Timestamp=" not in detail:
        raise ValueError("签名缺少 hardened runtime 或可信时间戳")
    assessment = execute(["spctl", "--assess", "--type", "execute", "--verbose=4", str(binary)])
    if "source=Notarized Developer ID" not in assessment:
        raise ValueError("缺少 Gatekeeper 公证验证结果")


def verify(directory, team):
    if sys.platform != "darwin":
        raise ValueError("macOS 签名、公证验收必须在 macOS 运行")
    if not re.fullmatch(r"[A-Z0-9]{10}", team):
        raise ValueError("MACOS_TEAM_ID 未配置或格式无效")
    record = json.loads((directory / "BUILD.json").read_text())
    output = directory / "macos-signing-verified.json"
    if output.exists():
        raise ValueError("签名验证记录已存在，拒绝覆盖")
    archives = {}
    for arch in ("arm64", "amd64"):
        archive = directory / f"kdl-agent_{record['version']}_darwin_{arch}.tar.gz"
        with tempfile.TemporaryDirectory(prefix="kdl-macos-verify-") as temporary:
            # 只提取需要验证的普通文件，不展开归档内的其他路径或链接。
            with tarfile.open(archive, "r:gz") as bundle:
                matches = [member for member in bundle if member.name.removeprefix("./") == "kdl-agent"]
                if len(matches) != 1 or not matches[0].isfile():
                    raise ValueError("macOS 归档缺少唯一普通可执行文件")
                binary = Path(temporary) / "kdl-agent"
                binary.write_bytes(bundle.extractfile(matches[0]).read())
                binary.chmod(0o700)
            verify_signature(binary, team)
        archives[arch] = {"sha256": sha256(archive), "verified": True}
    proof = {"version": record["version"], "commit": record["commit"],
             "sha256": sha256(directory / "SHA256SUMS"), "team_id": team, "archives": archives}
    with output.open("x") as stream:
        json.dump(proof, stream, indent=2)
        stream.write("\n")
    print("macOS 双架构签名、公证验证通过，记录已绑定固定候选")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("directory", type=Path)
    args = parser.parse_args()
    try:
        verify(args.directory, os.environ.get("MACOS_TEAM_ID", ""))
    except (OSError, ValueError, subprocess.TimeoutExpired) as error:
        parser.exit(1, f"{error}\n")
