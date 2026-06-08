<script setup lang="ts">
// 获取当前 Vue 实例的 proxy 对象
import {useProxiesStore} from "@/store/proxiesStore";
import {useMenuStore} from "@/store/menuStore";
import createApi from "@/api";
import {Events} from "@/runtime";
import {useI18n} from "vue-i18n";
import {pError, pLoad, pSuccess} from "@/util/pLoad";
import {useSettingStore} from "@/store/settingStore";
import {resetProxyOriginCache} from "@/api/proxies";
import {useWebStore} from "@/store/webStore";

// i18n
const {t} = useI18n();

// 获取当前 Vue 实例的 proxy 对象
const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);

// 当前页面使用store
const menuStore = useMenuStore();
const proxiesStore = useProxiesStore();
const settingStore = useSettingStore();
const webStore = useWebStore();

// 模式切换
Events.On("switchMode", (ev: any) => {
  console.log("switchMode=====", ev);
  menuStore.rule = ev;
});
watch(
    () => menuStore.rule,
    (newVal) => {
      Events.Emit({name: "mode", data: newVal});
    }
);

// Watch for proxy changes and update tray menu with debounce
let updateTrayTimeout: any = null;
watch(
    () => proxiesStore.now,
    () => {
      // Debounce to avoid too frequent updates
      clearTimeout(updateTrayTimeout);
      updateTrayTimeout = setTimeout(() => {
        updateProxyGroupsInTray();
      }, 500);
    }
);


const normalizeSwitchProfilePayload = (ev: any) => {
  if (ev && ev.profile) {
    return {
      profile: ev.profile,
      selected: ev.selected,
      exclusive: ev.exclusive,
    };
  }

  return {
    profile: ev,
    selected: true,
    exclusive: true,
  };
};

let isSwitchingProfiles = false;

// 配置切换
Events.On("switchProfiles", async (ev: any) => {
  if (isSwitchingProfiles) {
    return;
  }
  const {profile, selected, exclusive} = normalizeSwitchProfilePayload(ev);
  const allowMulti = settingStore.multiProfileEnabled;
  const nextSelected = allowMulti ? (typeof selected === 'boolean' ? selected : true) : true;
  const isExclusive = allowMulti ? (typeof exclusive === 'boolean' ? exclusive : true) : true;

  isSwitchingProfiles = true;
  try {
    await pLoad(t('profiles.switch.ing'), async () => {
      try {
        await api.switchProfile({
          id: profile?.id,
          selected: nextSelected,
          exclusive: isExclusive,
        })
        proxiesStore.active = ""

        await api.waitRunning()

        api.getRuleNum().then((res) => {
          menuStore.setRuleNum(res);
        });

        const list = await api.getProfileList();
        if (list && list.length != 0) {
          Events.Emit({
            name: "profiles",
            data: list
          })
        }

        const refreshed = list?.find((item: any) => item?.id === profile?.id);
        const primaryProfile = list?.find((item: any) => item?.primary);
        const selectedProfile = list?.find((item: any) => item?.selected);
        const activeProfile = nextSelected
            ? (primaryProfile ?? refreshed ?? {...profile, selected: nextSelected, primary: true})
            : (primaryProfile ?? selectedProfile);
        if (activeProfile) {
          webStore.fProfile = {
            ...activeProfile,
            exclusive: isExclusive,
          }
        }

        resetProxyOriginCache();

        // Update proxy groups after profile switch
        updateProxyGroupsInTray();
        Events.Emit({
          name: "profileChanged",
          data: {
            profile: activeProfile,
            exclusive: isExclusive,
          }
        })
        window.dispatchEvent(new CustomEvent('profile-changed'))

        pSuccess(t('profiles.switch.success'))
      } catch (e) {
        if (e['message']) {
          pError(e['message'])
        }
      }
    })
  } finally {
    isSwitchingProfiles = false;
  }
});

// Switch proxy in group from tray
Events.On("switchProxyInGroup", async (ev: any) => {
  const {group, proxy} = ev;

  try {
    // Use correct API format: {name: proxyName}
    await api.setProxy(group, {name: proxy});

    // Update store to trigger watchers in other components
    proxiesStore.setActive(group);
    proxiesStore.setNow(proxy);

    // Poll API to verify the change was applied
    let retries = 0;
    const maxRetries = 5;
    const checkInterval = 200;

    const waitForUpdate = () => {
      setTimeout(async () => {
        try {
          const proxies = await api.getProxies(group, false, false);
          const current = proxies.find((p: any) => p?.now);

          if (current?.name === proxy || retries >= maxRetries) {
            // Proxy switched successfully or max retries reached
            updateProxyGroupsInTray();
          } else {
            // Not yet switched, retry
            retries++;
            waitForUpdate();
          }
        } catch (e) {
          // On error, just update menu anyway
          updateProxyGroupsInTray();
        }
      }, checkInterval);
    };

    waitForUpdate();
  } catch (e) {
    if (e['message']) {
      pError(e['message'])
    }
  }
});

// Function to update proxy groups in tray
const updateProxyGroupsInTray = async () => {
  try {
    const groups = await api.getGroups();
    if (!groups || groups.length === 0) {
      return;
    }

    const proxyGroupsData = await Promise.all(
      groups.map(async (groupItem: any) => {
        const groupName = typeof groupItem === 'string' ? groupItem : groupItem.name;
        if (!groupName) {
          return null;
        }

        try {
          const proxies = await api.getProxies(groupName, false, false);
          // Only include groups that have proxies
          if (proxies && proxies.length > 0) {
            return {
              name: groupName,
              proxies: proxies.map((p: any) => ({
                name: p.name,
                now: p.now || false,
                type: p.type
              }))
            };
          }
        } catch (e) {
          // Ignore errors for individual groups
        }
        return null;
      })
    );

    const validGroups = proxyGroupsData.filter((g) => g !== null);

    if (validGroups.length > 0) {
      Events.Emit({
        name: "proxyGroups",
        data: validGroups
      });
    }
  } catch (e) {
    console.error('Failed to update proxy groups in tray:', e);
  }
};

// 快捷键注册
watch(() => settingStore.sc_switch, (val) => {
  if (val) {
    Events.Emit({name: 'shortcut:register', data: {name: 'showOrHide', key: settingStore.sc_switch_key}});
  } else {
    Events.Emit({name: 'shortcut:unregister-all', data: null});
  }
});

watch(() => settingStore.sc_switch_key, (newKey, oldKey) => {
  if (settingStore.sc_switch) {
    Events.Emit({name: 'shortcut:register', data: {name: 'showOrHide', key: newKey, old: oldKey}});
  }
});

Events.On('shortcut:result', (result: boolean) => {
  console.log('Shortcut registration result:', result);
});

// "Start minimized to tray": keep the shell's persisted flag in sync. The Wails
// shell reads it at startup to decide whether to show the window; harmless under
// Electron (no handler — Electron reads the setting from its own store).
watch(() => settingStore.startMinimized, (val) => {
  Events.Emit({name: 'startMinimized', data: !!val});
});

onMounted(async () => {
  // Publish the current settings the shell needs at startup (hotkey + start
  // minimized), so the persisted flags match the UI even without a change.
  Events.Emit({name: 'startMinimized', data: !!settingStore.startMinimized});
  if (settingStore.sc_switch) {
    Events.Emit({name: 'shortcut:register', data: {name: 'showOrHide', key: settingStore.sc_switch_key}});
  }
  // 获取初始数据
  const res = await api.getMihomo()
  menuStore.setRule(res.mode)
  menuStore.setProxy(res.proxy)

  settingStore.setPort(res.port)
  settingStore.setBindAddress(res.bindAddress)
  settingStore.setStack(res.stack)
  settingStore.setDns(res.dns)
  settingStore.setIpv6(res.ipv6)


  // 发送代理模式数据
  Events.Emit({name: "mode", data: menuStore.rule});

  // 发送订阅配置数据
  api.getProfileList().then((list) => {
    if (list && list.length != 0) {
      Events.Emit({
        name: "profiles",
        data: list
      })
    }
  })

  // 发送系统代理数据
  Events.Emit({
    name: "proxy",
    data: menuStore.proxy
  })

  // Send proxy groups data to tray
  await updateProxyGroupsInTray();
})

</script>

<template></template>
