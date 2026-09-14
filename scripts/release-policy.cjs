'use strict';
const fs = require('node:fs');
const path = require('node:path');

// 客户端人工平台验收不作为发行门；构建、运行烟测和权限回归继续由 CI 执行。
const CHECKS = ['business-e2e', 'production-operations', 'public-entry'];


function releaseChannel(version) {
  const number = '(0|[1-9][0-9]*)';
  const base = `${number}\\.${number}\\.${number}`;
  if (new RegExp(`^${base}-beta\\.${number}$`).test(version)) return 'beta';
  if (new RegExp(`^${base}$`).test(version)) return 'latest';
  throw new Error('发行版本仅支持 X.Y.Z 或 X.Y.Z-beta.N');
}

function validateAcceptance(record, version) {
  if (record?.schema_version !== 1 || record.version !== version ||
      typeof record.reviewer !== 'string' || !record.reviewer.trim() || !Array.isArray(record.checks)) {
    throw new Error('稳定验收记录缺失、版本不符或未登记审核人');
  }
  if (record.checks.length !== CHECKS.length || new Set(record.checks.map(item => item.id)).size !== CHECKS.length) {
    throw new Error('稳定验收记录的检查项缺失或重复');
  }
  for (const id of CHECKS) {
    const item = record.checks.find(check => check.id === id);
    if (item?.status !== 'passed' || !/^https:\/\/[^\s]+$/.test(item.evidence || '')) {
      throw new Error(`稳定验收尚未完成或缺少证据索引：${id}`);
    }
  }
  return record;
}

function checkRelease(tag, { root = path.resolve(__dirname, '..') } = {}) {
  const pkg = JSON.parse(fs.readFileSync(path.join(root, 'package.json')));
  const channel = releaseChannel(pkg.version);
  if (tag !== `v${pkg.version}`) throw new Error('tag 与 package.json 版本不一致');
  if (channel === 'latest') {
    const file = path.join(root, 'release/stable-acceptance.json');
    if (!fs.existsSync(file)) throw new Error('稳定发行缺少 release/stable-acceptance.json 验收记录');
    validateAcceptance(JSON.parse(fs.readFileSync(file)), pkg.version);

  }
  return channel;
}

function validateSigning(record, source, sumsHash, sums) {
  if (record?.version !== source.version || record.commit !== source.commit || record.sha256 !== sumsHash ||
      !/^[A-Z0-9]{10}$/.test(record.team_id || '')) throw new Error('签名验证记录不属于当前候选');
  for (const arch of ['arm64', 'amd64']) {
    const item = record.archives?.[arch];
    const expected = sums.get(`kdl-agent_${source.version}_darwin_${arch}.tar.gz`);
    if (item?.verified !== true || !expected || item.sha256 !== expected) throw new Error('macOS 双架构签名验证记录缺失或内容不符');
  }
}

if (require.main === module) {
  try {
    const channel = checkRelease(process.argv[2]);
    console.log(`发行策略校验通过：${channel}`);
  } catch (error) { console.error(error.message); process.exitCode = 1; }
}
module.exports = { releaseChannel, validateAcceptance, validateSigning, checkRelease, CHECKS };
