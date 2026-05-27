import { createRouter, createWebHashHistory } from "vue-router";
import type { RouteRecordRaw } from "vue-router";

const routes: RouteRecordRaw[] = [
  {
    path: "/",
    redirect: "/import",
  },
  {
    path: "/import",
    component: () => import("@/features/import/pages/ImportPage.vue"),
    children: [
      {
        path: "",
        name: "import-actions",
        component: () => import("@/features/import/pages/ImportActions.vue"),
      },
      {
        path: "wanmei",
        name: "import-wanmei",
        component: () => import("@/features/import/pages/WanmeiImport.vue"),
      },
      {
        path: "5e",
        name: "import-5e",
        component: () => import("@/features/import/pages/FiveEImport.vue"),
      },
    ],
  },
  {
    path: "/clips",
    name: "clips",
    component: () => import("@/features/clips/pages/ClipsPage.vue"),
  },
  {
    path: "/produce",
    name: "produce",
    component: () => import("@/features/produce/pages/ProducePage.vue"),
  },
  {
    path: "/edit",
    name: "edit",
    component: () => import("@/features/edit/pages/EditPage.vue"),
  },
  {
    path: "/settings",
    name: "settings",
    component: () => import("@/features/settings/pages/SettingsPage.vue"),
  },
];

const router = createRouter({
  history: createWebHashHistory(),
  routes,
});

export default router;
