<template>
  <div class="death-notice" :class="{ compact }">
    <span :class="['name', sideClass(kill.killer_side)]">{{ kill.killer_name }}</span>
    <img v-if="weaponID" :src="weaponIcon(weaponID)" class="weapon-icon" alt="weapon" />
    <span class="weapon-name">{{ weaponLabel(kill.weapon_name) }}</span>
    <img v-if="kill.is_headshot" :src="iconSrc('headshot')" class="suffix-icon" alt="headshot" />
    <img v-if="kill.is_wallbang" :src="iconSrc('penetrate')" class="suffix-icon" alt="penetrate" />
    <span :class="['name', sideClass(kill.victim_side)]">{{ kill.victim_name }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { DemoClipKill } from "@/shared/types";

const props = withDefaults(
  defineProps<{
    kill: DemoClipKill;
    compact?: boolean;
  }>(),
  {
    compact: false,
  },
);

const weaponDisplay: Record<string, string> = {
  ak47: "AK-47",
  aug: "AUG",
  awp: "AWP",
  bizon: "PP-Bizon",
  cz75a: "CZ75-Auto",
  deagle: "Desert Eagle",
  elite: "Dual Berettas",
  famas: "FAMAS",
  fiveseven: "Five-SeveN",
  galilar: "Galil AR",
  g3sg1: "G3SG1",
  glock: "Glock-18",
  hegrenade: "HE Grenade",
  hkp2000: "P2000",
  incgrenade: "Incendiary",
  knife: "Knife",
  m249: "M249",
  m4a1: "M4A4",
  m4a1_silencer: "M4A1-S",
  mac10: "MAC-10",
  mag7: "MAG-7",
  molotov: "Molotov",
  mp5sd: "MP5-SD",
  mp7: "MP7",
  mp9: "MP9",
  negev: "Negev",
  nova: "Nova",
  p250: "P250",
  p90: "P90",
  revolver: "R8 Revolver",
  sawedoff: "Sawed-Off",
  scar20: "SCAR-20",
  sg556: "SG 553",
  smokegrenade: "Smoke Grenade",
  ssg08: "SSG 08",
  taser: "Zeus x27",
  tec9: "Tec-9",
  ump45: "UMP-45",
  usp_silencer: "USP-S",
  xm1014: "XM1014",
};

const weaponAliases: Array<{ match: RegExp; id: string }> = [
  { match: /ak-?47/, id: "ak47" },
  { match: /awp/, id: "awp" },
  { match: /m4a4/, id: "m4a1" },
  { match: /m4a1[-_ ]s|m4a1s/, id: "m4a1_silencer" },
  { match: /m4a1/, id: "m4a1_silencer" },
  { match: /usp-?s/, id: "usp_silencer" },
  { match: /usp/, id: "usp_silencer" },
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

function normalizeKey(value: unknown): string {
  return String(value || "")
    .toLowerCase()
    .replace(/^weapon_/, "")
    .replace(/[\s-]/g, "")
    .replace(/[^a-z0-9_]/g, "");
}

function resolveWeaponID(name: string): string {
  if (!name) return "";
  const lower = name.toLowerCase();
  for (const rule of weaponAliases) {
    if (rule.match.test(lower)) {
      return rule.id;
    }
  }
  return normalizeKey(name);
}

const weaponID = computed(() => resolveWeaponID(props.kill.weapon_name));

function weaponLabel(name: string): string {
  const id = resolveWeaponID(name);
  if (weaponDisplay[id]) return weaponDisplay[id];
  return String(name || "weapon").replace(/^weapon_/, "");
}

function weaponIcon(id: string): string {
  return `/cs2/weapon/${id}.svg`;
}

function iconSrc(name: string): string {
  return `/cs2/deathnotice/${name}.svg`;
}

function sideClass(side: string): string {
  const normalized = (side || "").toLowerCase();
  if (normalized === "ct") return "side-ct";
  if (normalized === "t") return "side-t";
  return "side-t";
}
</script>

<style scoped>
.death-notice {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  overflow: hidden;
  padding: 6px 10px;
  border-radius: 6px;
  background: rgba(0, 0, 0, 0.55);
  color: #fff;
  font-size: 13px;
  font-weight: 700;
  line-height: 1;
}

.death-notice.compact {
  padding: 4px 8px;
  font-size: 12px;
}

.name {
  min-width: 0;
  max-width: 180px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.side-ct {
  color: #6f9ce6;
}

.side-t {
  color: #eabe54;
}

.weapon-icon {
  height: 18px;
  width: auto;
  flex: 0 0 auto;
  filter: brightness(0) invert(1);
}

.weapon-name {
  min-width: 0;
  max-width: 140px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  font-weight: 600;
  color: #e6e6e6;
}

.suffix-icon {
  height: 16px;
  width: auto;
  flex: 0 0 auto;
}
</style>
