<template>
  <ul class="death-notice-list">
    <li v-for="(kill, idx) in kills" :key="kill.tick ?? idx" class="death-notice">
      <span class="name attacker">{{ kill.killer_name }}</span>
      <img v-if="weaponId(kill.weapon_name)" :src="weaponIcon(weaponId(kill.weapon_name))" class="weapon-icon" alt="weapon" />
      <span class="weapon-name">{{ weaponLabel(kill.weapon_name) }}</span>
      <img v-if="kill.is_headshot" :src="iconSrc('headshot')" class="suffix-icon" alt="headshot" />
      <img v-if="kill.is_wallbang" :src="iconSrc('penetrate')" class="suffix-icon" alt="penetrate" />
      <span class="name victim">{{ kill.victim_name }}</span>
    </li>
  </ul>
</template>

<script setup>
const props = defineProps({
  kills: {
    type: Array,
    default: () => [],
  },
});

const cs2WeaponMap = {
  ak47: "AK47",
  ammobox: "弹药箱",
  ammobox_threepack: "弹药箱三件套",
  armor: "护甲",
  armor_helmet: "护甲头盔",
  assaultsuit_helmet_only: "突击套装头盔",
  assaultsuit: "突击套装",
  aug: "AUG",
  awp: "AWP",
  axe: "斧头",
  bayonet: "刺刀",
  bizon: "PP野牛",
  breachcharge: "遥控炸弹",
  breachcharge_projectile: "遥控炸弹投射物",
  bumpmine: "弹射地雷",
  c4: "C4炸弹",
  clothing_hands: "服装手",
  controldrone: "无人机",
  customplayer: "自定义玩家",
  cz75a: "CZ75",
  deagle: "沙漠之鹰",
  decoy: "诱饵弹",
  defuser: "拆弹器",
  disconnect: "断开连接",
  diversion: "分散注意",
  dronegun: "无人机枪",
  elite: "精英",
  famas: "法玛斯",
  firebomb: "燃烧弹",
  fists: "拳头",
  fiveseven: "FN57",
  flair0: "天赋",
  flashbang: "闪光弹",
  flashbang_assist: "闪光弹助攻",
  frag_grenade: "破片手榴弹",
  g3sg1: "G3SG1",
  galilar: "加利尔",
  glock: "格洛克",
  grenadepack: "手榴弹包",
  grenadepack2: "手榴弹包2",
  hammer: "锤子",
  healthshot: "医疗针",
  heavy_armor: "重装甲",
  hegrenade: "高爆手榴弹",
  helmet: "头盔",
  hkp2000: "P2000",
  incgrenade: "燃烧手榴弹",
  inferno: "地狱火",
  kevlar: "凯夫拉",
  knife: "刀",
  knife_bowie: "鲍伊猎刀",
  knife_butterfly: "蝴蝶刀",
  knife_canis: "求生匕首",
  knife_cord: "系绳匕首",
  knife_css: "海豹短刀",
  knife_falchion: "弯刀",
  knife_flip: "折叠刀",
  knife_gut: "穿肠刀",
  knife_gypsy_jackknife: "折刀",
  knife_karambit: "爪子刀",
  knife_kukri: "廓尔喀弯刀",
  knife_m9_bayonet: "M9刺刀",
  knife_push: "暗影双匕",
  knife_skeleton: "骷髅匕首",
  knife_stiletto: "短剑",
  knife_survival_bowie: "求生鲍伊猎刀",
  knife_t: "T刀",
  knife_tactical: "猎杀者匕首",
  knife_twinblade: "双刃匕首",
  knife_ursus: "熊刀",
  knife_widowmaker: "锯齿爪刀",
  knifegg: "gg刀",
  m4a1: "M4A4",
  m4a1_silencer: "M4A1消音版",
  m4a1_silencer_off: "M4A1无消音器",
  m249: "M249",
  mac10: "MAC-10",
  mag7: "MAG-7",
  melee: "幽灵之刃",
  molotov: "燃烧弹",
  movelinear: "拳击",
  mp5sd: "MP5",
  mp7: "MP7",
  mp9: "MP9",
  negev: "内格夫",
  nova: "新星",
  p90: "P90",
  p250: "P250",
  p2000: "P2000",
  planted_c4_survival: "放置C4生存",
  planted_c4: "放置C4",
  prop_exploding_barrel: "爆炸桶",
  radarjammer: "雷达干扰器",
  revolver: "左轮手枪",
  sawedoff: "匪喷",
  scar20: "SCAR",
  sg556: "SG553",
  shield: "盾牌",
  smokegrenade: "烟雾弹",
  snowball: "雪球",
  spanner: "扳手",
  spray0: "喷漆",
  ssg08: "SSG08",
  stomp_damage: "踩踏伤害",
  tablet: "平板",
  tagrenade: "标记手榴弹",
  taser: "电击枪",
  tec9: "Tec-9",
  tripwirefire_projectile: "绊网火投射物",
  tripwirefire: "绊网火",
  ump45: "UMP45",
  usp_silencer: "USP消音版",
  usp_silencer_off: "USP无消音器",
  xm1014: "XM1014连喷",
  zone_repulsor: "区域排斥装置",
};

const labelToId = Object.entries(cs2WeaponMap).reduce((acc, [id, label]) => {
  const key = normalizeKey(label);
  acc[key] = id;
  return acc;
}, {});

const weaponAliases = [
  { match: /ak-?47/, id: "ak47" },
  { match: /awp/, id: "awp" },
  { match: /m4a4/, id: "m4a1" },
  { match: /m4a1[-_ ]s|m4a1s/, id: "m4a1_silencer" },
  { match: /m4a1/, id: "m4a1_silencer" },
  { match: /usp-?s/, id: "usp_silencer" },
  { match: /usp/, id: "usp_silencer_off" },
  { match: /glock/, id: "glock" },
  { match: /deagle|desert\s*eagle/, id: "deagle" },
  { match: /p250/, id: "p250" },
  { match: /p2000/, id: "p2000" },
  { match: /famas/, id: "famas" },
  { match: /galil|galilar/, id: "galilar" },
  { match: /sg\s*553|sg556/, id: "sg556" },
  { match: /aug/, id: "aug" },
  { match: /ssg\s*08/, id: "ssg08" },
  { match: /g3sg1/, id: "g3sg1" },
  { match: /scar\s*20|scar20/, id: "scar20" },
  { match: /mp9/, id: "mp9" },
  { match: /mp7/, id: "mp7" },
  { match: /mac-?10/, id: "mac10" },
  { match: /ump45/, id: "ump45" },
  { match: /p90/, id: "p90" },
  { match: /bizon|pp-?bizon/, id: "bizon" },
  { match: /mp5/, id: "mp5sd" },
  { match: /nova/, id: "nova" },
  { match: /mag-?7/, id: "mag7" },
  { match: /xm1014/, id: "xm1014" },
  { match: /m249/, id: "m249" },
  { match: /negev/, id: "negev" },
  { match: /he\s*grenade|hegrenade/, id: "hegrenade" },
  { match: /flashbang/, id: "flashbang" },
  { match: /smoke/, id: "smokegrenade" },
  { match: /molotov/, id: "molotov" },
  { match: /incendiary|incgrenade/, id: "incgrenade" },
  { match: /decoy/, id: "decoy" },
  { match: /knife/, id: "knife" },
];

function normalizeKey(value) {
  return String(value || "")
    .toLowerCase()
    .replace(/^weapon_/, "")
    .replace(/[\s\-]/g, "")
    .replace(/[^a-z0-9_]/g, "");
}

function resolveWeaponId(name) {
  if (!name) return "";
  const raw = String(name);
  const lower = raw.toLowerCase();
  for (const rule of weaponAliases) {
    if (rule.match.test(lower)) return rule.id;
  }
  const cleaned = normalizeKey(raw);
  if (cs2WeaponMap[cleaned]) return cleaned;
  const labelKey = normalizeKey(raw);
  return labelToId[labelKey] || cleaned;
}

function weaponId(name) {
  return resolveWeaponId(name);
}

function weaponLabel(name) {
  const id = resolveWeaponId(name);
  return cs2WeaponMap[id] || String(name || "weapon").replace(/^weapon_/, "");
}

function weaponIcon(id) {
  return `/cs2/weapon/${id}.svg`;
}

function iconSrc(name) {
  return `/cs2/deathnotice/${name}.svg`;
}
</script>

<style scoped>
.death-notice-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.death-notice {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  border-radius: 4px;
  background: rgba(0, 0, 0, 0.65);
  color: #fff;
  font-weight: 700;
  font-size: 13px;
  line-height: 1;
  width: fit-content;
}

.name {
  max-width: 220px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.attacker {
  color: #6f9ce6;
}

.victim {
  color: #eabe54;
}

.weapon-icon {
  height: 20px;
  width: auto;
  filter: brightness(0) invert(1);
}

.weapon-name {
  font-size: 12px;
  font-weight: 600;
  color: #e6e6e6;
  margin-right: 2px;
}

.suffix-icon {
  height: 18px;
  width: auto;
}

.weapon-fallback {
  font-size: 12px;
  padding: 2px 6px;
  border-radius: 3px;
  background: rgba(255, 255, 255, 0.12);
}
</style>
