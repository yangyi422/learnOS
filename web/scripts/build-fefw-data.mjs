#!/usr/bin/env node
// 从攻略站生成挖角手帐用的角色技能索引。
//
// 站点：https://cn.fire-emblem-fw.site（fire-emblem-fw.site 的中文镜像）
// 数据：/data/characters.json（角色·个人特技·血印·可习得战技/魔法/技能/等级）
//       /data/skills.json（特技/战技/魔法的名称与效果文本）
//
// 运行时**不抓取**任何网络数据：本脚本把结果写成
// web/public/library-tools/fefw-planner/data-skills.js，工具页离线可用，
// 也不受站点防爬策略影响。wiki 更新后重跑本脚本即可。
//
// 用法：node web/scripts/build-fefw-data.mjs
import { writeFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const BASE = 'https://cn.fire-emblem-fw.site/data'
const OUT = resolve(dirname(fileURLToPath(import.meta.url)), '../public/library-tools/fefw-planner/data-skills.js')

async function fetchJson(name) {
  const response = await fetch(`${BASE}/${name}`, { headers: { 'User-Agent': 'LifeOS-fefw-data/1.0' } })
  if (!response.ok) throw new Error(`${name}: HTTP ${response.status}`)
  return response.json()
}

const [characters, skills] = await Promise.all([fetchJson('characters.json'), fetchJson('skills.json')])

// 名称 → 效果文本，用于「可习得技能」补全说明（characters.json 只给名字）。
const skillText = new Map()
for (const list of [skills.skills, skills.arts, skills.magic]) {
  for (const item of list) skillText.set(item.name, item)
}

// 战技按武器分组排序：剑 → 枪 → 斧 → 弓 → 手甲 → 圣焰 → 任意 → 其他。
// 攻略站没有给出每个战技的熟练需求等级，因此同类内按消耗、威力、名称排。
const WEAPON_ORDER = ['剑', '枪', '斧', '弓', '手甲', '圣焰', '任意']
const weaponRank = weapon => {
  const i = WEAPON_ORDER.indexOf(weapon)
  return i === -1 ? WEAPON_ORDER.length : i
}
const rankOrder = rank => {
  const i = ['D', 'C', 'B', 'A', 'S', 'S+'].indexOf(String(rank || '').replace(/[^A-Z+]/g, ''))
  return i === -1 ? 99 : i
}
const num = value => {
  const n = Number(String(value ?? '').replace(/[^0-9.]/g, ''))
  return Number.isFinite(n) ? n : 999
}

// 个人特技之外的「等级特技」：从 skills.json 的 kind=等级 反查，按 Lv 升序。
const levelSkills = new Map() // 角色名 → [{name, level, text}]
for (const skill of skills.skills) {
  if (skill.kind !== '等级') continue
  for (const line of skill.learn ?? []) {
    const match = /^(.+?)：Lv\.(\d+)$/.exec(String(line).trim())
    if (!match) continue
    const [, char, level] = match
    if (!levelSkills.has(char)) levelSkills.set(char, [])
    levelSkills.get(char).push({ name: skill.name, level: Number(level), text: skill.text ?? '' })
  }
  // 少数角色专属 Lv 特技在站点上标了「等级待补」，只记名字。
  if ((skill.learn ?? []).length === 0 && Array.isArray(skill.chars) && skill.chars.length === 1) {
    const char = skill.chars[0]
    if (!levelSkills.has(char)) levelSkills.set(char, [])
    levelSkills.get(char).push({ name: skill.name, level: null, text: skill.text ?? '' })
  }
}

// 「谁能学」反查表（只保留本工具收录的角色）。
const known = new Set(characters.map(c => c.name))
const holders = { arts: {}, magic: {} }
for (const art of skills.arts) {
  const names = (art.chars ?? []).filter(n => known.has(n))
  if (names.length) holders.arts[art.name] = names
}
for (const spell of skills.magic) {
  const names = (spell.chars ?? []).filter(n => known.has(n))
  if (names.length) holders.magic[spell.name] = names
}

const index = {}
for (const character of characters) {
  if (!character.name) continue
  if (!Array.isArray(character.learn?.skills)) continue // 众神/剧情人物没有可习得数据

  const learn = character.learn
  const skills = (learn.skills ?? []).map(name => {
    const meta = skillText.get(name) ?? {}
    return { name, kind: meta.kind ?? '', text: meta.text ?? '' }
  })
  const arts = (learn.arts ?? []).map(name => {
    const meta = skillText.get(name) ?? {}
    return {
      name,
      weapon: meta.weapon ?? '',
      cost: meta.cost ?? '',
      might: meta.might ?? '',
      hit: meta.hit ?? '',
      text: meta.text ?? '',
    }
  })
  const magic = (learn.magic ?? []).map(name => {
    const meta = skillText.get(name) ?? {}
    const rank = learn.ranks?.[name]
    return {
      name,
      type: meta.type ?? '',
      uses: meta.uses ?? '',
      might: meta.might ?? '',
      hit: meta.hit ?? '',
      range: meta.range ?? '',
      text: meta.text ?? '',
      rank: rank ? `${rank.sk ?? ''} ${rank.r ?? ''}`.trim() : '',
    }
  })

  arts.sort((a, b) =>
    weaponRank(a.weapon) - weaponRank(b.weapon) ||
    num(a.cost) - num(b.cost) ||
    num(b.might) - num(a.might) ||
    a.name.localeCompare(b.name, 'zh'))
  magic.sort((a, b) => rankOrder(a.rank) - rankOrder(b.rank) || a.name.localeCompare(b.name, 'zh'))

  index[character.name] = {
    personal: character.personalSkill ?? null,
    blood: character.bloodMarks ?? [],
    lv: (levelSkills.get(character.name) ?? []).sort((a, b) =>
      (a.level ?? 999) - (b.level ?? 999) || a.name.localeCompare(b.name, 'zh')),
    skills,
    arts,
    magic,
  }
}

const header = `// 由 web/scripts/build-fefw-data.mjs 生成，请勿手改。
// 来源：https://cn.fire-emblem-fw.site/data/{characters,skills}.json
// 生成时间：${new Date().toISOString()}
// 角色数：${Object.keys(index).length}
window.FEFW_SKILL_INDEX = `
const body = JSON.stringify(index, null, 1)
const NL = String.fromCharCode(10)
const holdersLine = 'window.FEFW_SKILL_HOLDERS = ' + JSON.stringify(holders) + ';' + NL
writeFileSync(OUT, header + body + ';' + NL + holdersLine, 'utf8')
console.log(
  'wrote ' + OUT + ' (' + Object.keys(index).length + ' characters, ' +
    Object.keys(holders.arts).length + ' arts, ' + Object.keys(holders.magic).length + ' spells)',
)
