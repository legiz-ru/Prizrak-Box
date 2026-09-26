<script setup lang="ts">
import {Profile} from "@/types/profile";
import createApi from "@/api";
import {pError, pLoad, pSuccess, pWarning} from "@/util/pLoad";
import {useProxiesStore} from "@/store/proxiesStore";
import {useMenuStore} from "@/store/menuStore";
import {useSettingStore} from "@/store/settingStore";
import {getTemplateTitle, isHttpOrHttps, prettyBytes} from "@/util/format";
import {useI18n} from "vue-i18n";
import {Browser, Clipboard, Events} from "@/runtime"
import {useWebStore} from "@/store/webStore";
import {WS} from "@/util/ws";
import {onBeforeRouteLeave} from "vue-router";
import AnnounceText from "@/components/home/AnnounceText.vue";
import AddProfileDialog from "@/components/profile/AddProfileDialog.vue";
import {confirm} from "@/components/ui";
import type {UiSelectOption} from "@/components/ui";
import {formatDateValue, formatExact, formatTrafficValue, hasValue, profileDisplayTitle, relativeDate} from "@/util/profileView";
import IconBell from "~icons/tabler/bell";
import IconSpeakerphone from "~icons/tabler/speakerphone";
import IconCopyCheck from "~icons/tabler/copy-check";
import IconDeviceTv from "~icons/tabler/device-tv";
import IconUserCog from "~icons/tabler/user-cog";
import IconPlus from "~icons/tabler/plus";
import {useHwidStatusStore} from "@/store/hwidStatusStore";
import {parseHwidFromError} from "@/api/profiles";

// i18n
const {t} = useI18n();

// 获取当前 Vue 实例的 proxy 对象
const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);

// 当前页面使用store
const menuStore = useMenuStore();
const proxiesStore = useProxiesStore();
const webStore = useWebStore();
const settingStore = useSettingStore();
const hwidStatusStore = useHwidStatusStore();

// 头部几个按钮操作
const addFormVisible = ref(false)
const isNowAdd = ref(false)
const addInitial = ref('')

async function add(form: { content: string; ageSecretKey: string }) {
  if (!form.content) {
    return
  }

  isNowAdd.value = true
  const p = new Profile()
  p.content = form.content
  if (form.ageSecretKey) {
    p.ageSecretKey = form.ageSecretKey
  }
  try {
    const pList = await api.addProfileFromInput(p)
    if (pList && pList.length > 0) {
      pList.forEach(item => profiles.push(item))
    }
    sendOrder(profiles)
    pSuccess(t('drag.success'))
    addFormVisible.value = false
  } catch (e) {
    const hwid = parseHwidFromError(e)
    if (hwid) {
      if (hwid.hwidNotSupported) {
        hwidStatusStore.showNotSupported();
      } else if (hwid.hwidMaxDevicesReached) {
        hwidStatusStore.showMaxDevicesReached(hwid.supportUrl);
      }
    } else if (e['message']) {
      pError(e['message'])
    }
  }
  isNowAdd.value = false
}

function handleAdd() {
  addInitial.value = ""
  addFormVisible.value = true
}

function handlePaste() {
  addInitial.value = Clipboard.Text()
  addFormVisible.value = true
}

function openFile() {
  webStore.dnd = true
}

const getProfileDisplayTitle = profileDisplayTitle

let profiles = reactive<any[]>([])

// webStore.profileList — единый источник/кэш списка профилей. Держим его в
// синхроне с тем, что реально отрендерено: при переключении вкладок (компонент
// каждый раз монтируется заново) onMounted мгновенно восстанавливает список из
// этого кэша, поэтому экран не «мигает» пустотой. Глубокий watch ловит и
// добавление/удаление/перестановку, и изменения внутри элементов (выбор профиля).
watch(profiles, () => {
  webStore.profileList = profiles.slice()
}, {deep: true})

const applyProfileList = (list: any[]) => {
  profiles.splice(0, profiles.length)
  if (Array.isArray(list) && list.length > 0) {
    list.forEach(item => profiles.push(item))
  }
  selectionOrder.value = []
  const seeded = seedSelectionOrder()
  if (seeded) {
    selectionOrder.value = seeded
  } else {
    selectionOrder.value = profiles.filter(profile => profile['selected']).map(profile => profile['id'])
  }
  ensurePrimaryFirst()
  applySelectionOrder()
  if (!multiProfileEnabled.value && !isSwitchingProfile.value) {
    const selectedProfiles = profiles.filter(profile => profile['selected'])
    if (selectedProfiles.length > 1) {
      const primary = profiles.find(profile => profile['primary'])
          ?? profiles.find(profile => profile['selected'])
      if (primary) {
        void switchProfile(primary, true, true)
      }
    }
  }
}

const handleProfilesEvent = (list: any[]) => {
  applyProfileList(Array.isArray(list) ? list : [])
}

async function getProfileList() {
  let list: any
  try {
    list = await api.getProfileList()
  } catch (e) {
    // Транзиентная ошибка запроса (сеть/таймаут/отмена при быстром
    // переключении вкладок) — сохраняем текущий список из кэша, не очищаем.
    console.error('[Profiles] getProfileList failed', e)
    return
  }

  // Перехватчик axios возвращает undefined для не-200 ответов (204/304 и т.п.).
  // Затирать список таким ответом нельзя — рендерим то, что уже есть в кэше.
  if (!Array.isArray(list)) {
    return
  }

  // Сюда попадает только реальный ответ сервера (в т.ч. честный пустой массив,
  // когда профилей действительно нет). Синхронизация с webStore.profileList
  // происходит автоматически через watch(profiles).
  applyProfileList(list)
  Events.Emit({
    name: "profiles",
    data: list
  })
}

// 拖动相关
const canDrag = ref(false)

function mouseEnter() {
  canDrag.value = true
}

function mouseLeave() {
  canDrag.value = false
}

const applyPrimarySelection = (id?: string) => {
  if (!id) {
    for (let profile of profiles) {
      profile['primary'] = false
    }
    return
  }
  for (let profile of profiles) {
    profile['primary'] = profile['id'] === id
  }
}

// 切换订阅配置
const selectionOrder = ref<string[]>([])
const multiProfileInfoVisible = ref(false)
const multiProfileEnabled = computed({
  get: () => settingStore.multiProfileEnabled,
  set: (value: boolean) => settingStore.setMultiProfileEnabled(value),
})
const isSwitchingProfile = ref(false)

const ensurePrimaryFirst = () => {
  const primaryId = profiles.find(profile => profile['selected'] && profile['primary'])?.['id']
  if (!primaryId) {
    return
  }
  selectionOrder.value = [primaryId, ...selectionOrder.value.filter(id => id !== primaryId)]
}

const appendSelectionOrder = (id: string) => {
  if (!id || selectionOrder.value.includes(id)) {
    return
  }
  selectionOrder.value.push(id)
}

const removeSelectionOrder = (id: string) => {
  selectionOrder.value = selectionOrder.value.filter(item => item !== id)
}

const applySelectionOrder = () => {
  const orderMap = new Map(selectionOrder.value.map((id, index) => [id, index + 1]))
  for (const profile of profiles) {
    const order = orderMap.get(profile['id'])
    if (profile['selected'] && order) {
      profile['selectionOrder'] = order
    } else {
      profile['selectionOrder'] = undefined
    }
  }
}

const seedSelectionOrder = () => {
  const selectedProfiles = profiles.filter(profile => profile['selected'])
  const ordered = selectedProfiles
      .filter(profile => typeof profile['selectionOrder'] === 'number' && profile['selectionOrder'] > 0)
      .sort((a, b) => (a['selectionOrder'] as number) - (b['selectionOrder'] as number))

  if (ordered.length === 0) {
    return null
  }

  const ids = ordered.map(profile => profile['id'])
  const seen = new Set(ids)
  for (const profile of selectedProfiles) {
    if (!seen.has(profile['id'])) {
      ids.push(profile['id'])
      seen.add(profile['id'])
    }
  }

  return ids
}

const syncSelectionOrder = () => {
  const selectedIds = profiles.filter(profile => profile['selected']).map(profile => profile['id'])
  if (selectionOrder.value.length === 0) {
    const seeded = seedSelectionOrder()
    selectionOrder.value = seeded ?? [...selectedIds]
    ensurePrimaryFirst()
    applySelectionOrder()
    return
  }
  selectionOrder.value = selectionOrder.value.filter(id => selectedIds.includes(id))
  for (const id of selectedIds) {
    if (!selectionOrder.value.includes(id)) {
      selectionOrder.value.push(id)
    }
  }
  ensurePrimaryFirst()
  applySelectionOrder()
}

async function switchProfile(data: any, desired?: boolean, exclusive = false) {
  if (isSwitchingProfile.value) {
    return
  }
  let nextSelected = typeof desired === 'boolean' ? desired : !data['selected']
  if (!multiProfileEnabled.value && !exclusive) {
    exclusive = true
    nextSelected = true
  }
  const wasPrimary = !!data['primary']

  const selectedCount = profiles.filter(profile => profile['selected']).length
  const hasPrimarySelected = profiles.some(profile => profile['selected'] && profile['primary'])

  if (!exclusive && !nextSelected && selectedCount <= 1) {
    pWarning(t("select-profile-warning"))
    return
  }

  isSwitchingProfile.value = true
  try {
    await pLoad(t('profiles.switch.ing'), async () => {
      try {
        await api.switchProfile({
          id: data['id'],
          selected: nextSelected,
          exclusive,
        })
        proxiesStore.active = ""

        await api.waitRunning()

        if (exclusive) {
          for (let profile of profiles) {
            profile['selected'] = profile['id'] === data['id'] && nextSelected
          }
          selectionOrder.value = nextSelected ? [data['id']] : []
          applyPrimarySelection(nextSelected ? data['id'] : undefined)
        } else {
          data['selected'] = nextSelected
          if (nextSelected) {
            appendSelectionOrder(data['id'])
            if (selectedCount == 0 || !hasPrimarySelected) {
              applyPrimarySelection(data['id'])
            }
          } else {
            removeSelectionOrder(data['id'])
            if (wasPrimary) {
              applyPrimarySelection(selectionOrder.value[0])
            }
          }
        }
        ensurePrimaryFirst()
        applySelectionOrder()

        const activeProfile = profiles.find(profile => profile['primary'])
            ?? profiles.find(profile => profile['selected'])
        if (activeProfile) {
          webStore.fProfile = toRaw({
            ...activeProfile,
            exclusive,
          })
        }

        api.getRuleNum().then((res) => {
          menuStore.setRuleNum(res);
        });

        Events.Emit({
          name: "profiles",
          data: toRaw(profiles)
        })
        Events.Emit({
          name: "profileChanged",
          data: toRaw(webStore.fProfile)
        })
        window.dispatchEvent(new CustomEvent('profile-changed'))

        // 关闭之前的连接
        api.closeAllConnection()

        pSuccess(t('profiles.switch.success'))
      } catch (e) {
        if (e['message']) {
          pError(e['message'])
        }
      }
    })
  } finally {
    isSwitchingProfile.value = false
  }

}

const confirmMultiProfileInfo = () => {
  multiProfileInfoVisible.value = false
  settingStore.setMultiProfileHintShown(true)
}

const showMultiProfileInfo = () => {
  multiProfileInfoVisible.value = true
}

const disableMultiProfile = async () => {
  multiProfileEnabled.value = false
  const primary = profiles.find(profile => profile['primary'])
      ?? profiles.find(profile => profile['selected'])
  const selectedCount = profiles.filter(profile => profile['selected']).length
  if (primary && selectedCount > 1) {
    await switchProfile(primary, true, true)
  }
}

const declineMultiProfileInfo = async () => {
  multiProfileInfoVisible.value = false
  await disableMultiProfile()
}

const toggleMultiProfile = async () => {
  const next = !multiProfileEnabled.value
  multiProfileEnabled.value = next
  if (!next) {
    await disableMultiProfile()
    return
  }

  // Accepted product change: the warning shows the first time multi-select is
  // turned on (until answered "Yes"), and on demand via the (i) button.
  if (!settingStore.multiProfileHintShown) {
    multiProfileInfoVisible.value = true
  }
}


watch(() => webStore.fProfile, async (data: any) => {
  if (!data || !data['id']) {
    return
  }

  const exclusive = !!data['exclusive']
  const desired = typeof data['selected'] === 'boolean' ? data['selected'] : true

  if (exclusive) {
    for (let profile of profiles) {
      profile['selected'] = profile['id'] === data['id'] && desired
    }
    selectionOrder.value = desired ? [data['id']] : []
    applyPrimarySelection(desired ? data['id'] : undefined)
    ensurePrimaryFirst()
    applySelectionOrder()
    return
  }

  let wasPrimary = false
  for (let profile of profiles) {
    if (profile['id'] === data['id']) {
      wasPrimary = !!profile['primary']
      profile['selected'] = desired
      break
    }
  }

  if (desired) {
    appendSelectionOrder(data['id'])
    const isPrimary = !!data['primary']
    const hasPrimarySelected = profiles.some(profile => profile['selected'] && profile['primary'])
    if (isPrimary || !hasPrimarySelected) {
      applyPrimarySelection(data['id'])
    }
  } else {
    removeSelectionOrder(data['id'])
    if (wasPrimary) {
      applyPrimarySelection(selectionOrder.value[0])
    }
  }
  ensurePrimaryFirst()
  applySelectionOrder()
})


// 更新订阅
async function refresh(data: any) {
  await pLoad(t('profiles.refresh.ing'), async () => {
    try {
      const re = await api.refreshProfile(data)
      Object.assign(data, re);
      webStore.fProfile = toRaw({...data});

      Events.Emit({
        name: "profiles",
        data: toRaw(profiles)
      })
      pSuccess(t('profiles.refresh.success'))

      if (re?.hwidNotSupported) {
        hwidStatusStore.showNotSupported();
      } else if (re?.hwidMaxDevicesReached) {
        const supportUrl = typeof re.support === 'string' ? re.support : '';
        hwidStatusStore.showMaxDevicesReached(supportUrl);
      }
    } catch (e) {
      if (e['message']) {
        pError(e['message'])
      }
    }
  })
}

// 几个按钮操作
// 到主页
function openExternalLink(raw: any) {
  if (typeof raw !== 'string') {
    return
  }

  const url = raw.trim()
  if (!url) {
    return
  }

  try {
    Browser.OpenURL(url)
  } catch (error) {
    if (typeof window !== 'undefined') {
      window.open(url, '_blank', 'noopener')
    }
  }
}

function goHome(data: any) {
  openExternalLink(data.home)
}

function goSupport(data: any) {
  openExternalLink(data.support)
}

function goRenew(data: any) {
  openExternalLink(data.renewUrl)
}

// TV send dialog
const tvDialogVisible = ref(false)
const tvDialogProfile = ref<any>(null)
const tvIsSending = ref(false)
const tvForm = reactive({ ip: '', port: '' })

function openTvDialog(data: any) {
  tvDialogProfile.value = data
  tvDialogVisible.value = true
}

const tvWarning = computed(() => t('profiles.tv-dialog.warning').replace(/^\s*⚠️\s*/u, ''))

async function submitToTv() {
  if (!tvForm.ip.trim() || !tvForm.port.trim()) {
    pError(t('profiles.tv-dialog.ip') + ' / ' + t('profiles.tv-dialog.port'))
    return
  }
  tvIsSending.value = true
  try {
    const url = `http://${tvForm.ip}:${tvForm.port}/Prizrak-BoxTVimport/submit`
    const controller = new AbortController()
    const timeout = setTimeout(() => controller.abort(), 4000)
    let ok = false
    try {
      const response = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content: tvDialogProfile.value?.content }),
        signal: controller.signal,
      })
      clearTimeout(timeout)
      const result = await response.json()
      ok = result?.status === 'ok'
    } catch {
      clearTimeout(timeout)
    }
    if (ok) {
      pSuccess(t('profiles.tv-dialog.success'))
      tvDialogVisible.value = false
    } else {
      pError(t('profiles.tv-dialog.error'))
    }
  } finally {
    tvIsSending.value = false
  }
}

// Announce dialog
const announceDialogVisible = ref(false)
let announceDialogData = reactive<any>({
  text: '',
  url: ''
})

function showAnnounce(data: any) {
  announceDialogData = reactive<any>({})
  Object.assign(announceDialogData, {
    text: data.announce || '',
    url: data.announceUrl || ''
  })
  announceDialogVisible.value = true
}

function goAnnounceUrl() {
  if (announceDialogData.url) {
    openExternalLink(announceDialogData.url)
  }
}

// 修改配置
const editFormVisible = ref(false)
const editHasAgeKey = ref(false)
const editShowAgeKey = ref(false)
let editForm = reactive<any>({})
let editFormD = {}

// Напоминания о подписке — колокольчик показывается только если продавец
// прислал хотя бы один из notify-expire-days/notify-traffic-percent (см.
// src-go/internal/resolve.go ParseHeaders)
const subscriptionAlertInfoVisible = ref(false)

function showSubscriptionAlertInfo() {
  subscriptionAlertInfoVisible.value = true
}

const subscriptionAlertInfoLines = computed(() => {
  const lines = [t('subscriptionAlert.infoBody')]
  if (editForm.notifyExpireDays?.length) {
    const days = [...editForm.notifyExpireDays].sort((a: number, b: number) => a - b).join(', ')
    lines.push(t('subscriptionAlert.infoExpire', {days}))
  }
  if (editForm.notifyTrafficPercent?.length) {
    const percent = [...editForm.notifyTrafficPercent].sort((a: number, b: number) => a - b).join(', ')
    lines.push(t('subscriptionAlert.infoTraffic', {percent}))
  }
  if (!settingStore.notifySubscriptionAlerts) {
    lines.push(t('subscriptionAlert.infoDisabledLocally'))
  }
  return lines
})

function updateProfile(data: any) {
  editFormD = data
  editForm = reactive<any>({})
  Object.assign(editForm, data)
  if (editForm.pxdTemplateUrl) {
    editForm.template = 'pxd_subscription'
  }
  editHasAgeKey.value = !!(editForm.ageSecretKey && editForm.ageSecretKey.trim())
  editShowAgeKey.value = false
  editFormVisible.value = true
}

const intervalLocked = computed(() => !!editForm.intervalFromHeader)
const intervalInvalid = computed(() => editForm.type == 1 && !intervalLocked.value && !validateField(editForm.interval))

function stepInterval(delta: number) {
  if (intervalLocked.value) return
  const current = parseInt(String(editForm.interval ?? ''), 10)
  const base = Number.isFinite(current) ? current : (delta > 0 ? 0 : 2)
  editForm.interval = String(Math.min(127, Math.max(1, base + delta)))
}

function onIntervalInput(e: Event) {
  if (intervalLocked.value) return
  editForm.interval = (e.target as HTMLInputElement).value.replace(/[^0-9]/g, '').slice(0, 3)
}

const templateOptions = computed<UiSelectOption<string>[]>(() => {
  const list: UiSelectOption<string>[] = []
  if (editForm.pxdTemplateUrl) {
    list.push({value: 'pxd_subscription', label: t('profiles.edit.pxd-subscription')})
  }
  tList.value.forEach((item: any) => list.push({value: item.id, label: getTemplateTitle(t, item.title)}))
  return list
})

function validateField(value: any) {
  // 如果为空，则通过校验
  if (value === "" || value === null || value === undefined) {
    return true;
  }

  // 如果不为空，验证是否是大于0且小于等于128的整数
  const regex = /^[1-9][0-9]?$|^1[0-2][0-8]$/;
  return regex.test(value.toString());
}

const isNowEdit = ref(false)

async function saveUpdateProfile() {

  switch (editForm.type) {
    case 2:
      if (!editForm.title) {
        pError(t('profiles.edit.title-tip'))
        return
      }
      break
    case 1:
      if (!editForm.title) {
        pError(t('profiles.edit.title-tip'))
        return
      }

      if (!editForm.content) {
        pError(t('profiles.edit.url-tip'))
        return
      }

      if (!isHttpOrHttps(editForm.content)) {
        pError(t('profiles.edit.url-error'))
        return
      }

      if (!validateField(editForm.interval)) {
        pError(t('profiles.edit.update-tip'))
        return
      }
  }

  isNowEdit.value = true
  try {
    await api.updateProfile(editForm)
  } finally {
    isNowEdit.value = false
  }
  // 更新当前页面的值
  Object.assign(editFormD, editForm)
  editFormVisible.value = false
  pSuccess(t('profiles.edit.success'))

  Events.Emit({
    name: "profiles",
    data: toRaw(profiles)
  })

  api.getRuleNum().then((res) => {
    menuStore.setRuleNum(res);
  });
}

// Accepted product change: deleting asks for confirmation first.
async function askDeleteProfile(data: any, index: any) {
  const ok = await confirm({
    title: t('confirm.delete-profile.title'),
    text: t('confirm.delete-profile.text', {name: getProfileDisplayTitle(data)}),
    okLabel: t('confirm.delete-profile.ok'),
  })
  if (ok) {
    await deleteProfile(data, index)
  }
}

// 删除配置
async function deleteProfile(data: any, index: any) {
  const isSelected = Boolean(data['selected']);
  try {
    await api.deleteProfile(data)
    profiles.splice(index, 1)
    Events.Emit({
      name: "profiles",
      data: toRaw(profiles)
    })
    if (profiles.length === 0) {
      webStore.fProfile = {}
      proxiesStore.setActive('')
      proxiesStore.setNow('')
      proxiesStore.replaceGroupExpansions({})
      Events.Emit({
        name: "profileChanged",
        data: {}
      })
      window.dispatchEvent(new CustomEvent('profile-changed'))
    }
    if (isSelected) {
      pWarning(t('profiles.deleted-select-new'))
    }
  } catch (e) {
    if (e['message']) {
      pError(e['message'])
    }
  }
}

// webSocket相关操作
let wsOrder: WS

function num2SafeNumber(data: any, key: string) {
  if (data[key] !== undefined && data[key] !== null) {
    let num = Number(data[key]);

    if (!Number.isFinite(num)) {
      console.warn(`Invalid number for key "${key}":`, data[key]);
      return;
    }

    if (num > Number.MAX_SAFE_INTEGER) {
      data[key] = Number.MAX_SAFE_INTEGER;
    } else if (num < Number.MIN_SAFE_INTEGER) {
      data[key] = Number.MIN_SAFE_INTEGER;
    } else {
      data[key] = num;
    }
  }
}

function sendOrder(data: any) {
  if (wsOrder) {
    Events.Emit({
      name: "profiles",
      data: toRaw(data)
    })
    for (let i = 0; i < data.length; i++) {
      num2SafeNumber(data[i], 'available')
      num2SafeNumber(data[i], 'used')
      num2SafeNumber(data[i], 'total')
    }
    wsOrder.send(JSON.stringify(data))
  }
}

async function handleProfilesImported(event: Event) {
  const customEvent = event as CustomEvent;
  const detail = customEvent.detail;
  if (!detail || !Array.isArray(detail.profiles)) {
    return;
  }

  try {
    const list = await api.getProfileList();
    if (Array.isArray(list)) {
      applyProfileList(list);
      sendOrder(profiles);
    }
  } catch (error) {
    console.error('Failed to refresh profiles after deeplink import', error);
  }
}

// 路由切换前关闭 WebSocket
onBeforeRouteLeave(() => {
  wsOrder.close();
});
onBeforeUnmount(() => {
  wsOrder.close();
  window.removeEventListener('deeplink-profile-imported', handleProfilesImported as EventListener);
  Events.Off("profiles", handleProfilesEvent)
})

// Template列表
const tList = ref<any[]>([]);

// Skeleton cards until the first list arrives (unless the cache already has one).
const firstLoad = ref(true);

// vue 周期相关
onMounted(async () => {
  const urlTraffic = webStore.wsUrl + "/profile/order?token=" + webStore.secret;
  wsOrder = new WS(urlTraffic);

  // Show cached list immediately (populated by App.vue on startup) so the
  // view renders without waiting for the API round-trip.
  if (webStore.profileList.length > 0) {
    applyProfileList(webStore.profileList);
  }

  try {
    await getProfileList()
  } finally {
    firstLoad.value = false
  }
  try {
    const templates = await api.getTemplateList();
    tList.value = [{title: 'm0', id: 'm0'}, ...(Array.isArray(templates) ? templates : [])];
  } catch (e) {
    tList.value = [{title: 'm0', id: 'm0'}];
  }

  window.addEventListener('deeplink-profile-imported', handleProfilesImported as EventListener);
  Events.On("profiles", handleProfilesEvent)
})

watch(() => webStore.dProfile, async (pList) => {
  if (pList && pList.length > 0) {
    pList.forEach(item => profiles.push(item))
  }
})

</script>

<template>
  <div class="px-page">
    <div class="px-page-head profiles-head">
      <h1 class="px-page-title">{{ $t('profiles.title') }}</h1>
      <div class="profiles-tools">
        <UiIconButton :label="multiProfileEnabled ? t('profiles.multi-select.disable') : t('profiles.multi-select.enable')"
                      :active="multiProfileEnabled"
                      :pressed="multiProfileEnabled"
                      @click="toggleMultiProfile">
          <icon-tabler-checkbox v-if="multiProfileEnabled" width="17" height="17"/>
          <icon-tabler-square v-else width="17" height="17"/>
        </UiIconButton>
        <UiIconButton :label="t('profiles.multi-select.info')" @click="showMultiProfileInfo">
          <icon-tabler-info-circle width="17" height="17"/>
        </UiIconButton>
        <span class="px-vdivider"></span>
        <UiIconButton :label="$t('profiles.add')" @click="handleAdd">
          <icon-tabler-plus width="17" height="17"/>
        </UiIconButton>
        <UiIconButton :label="$t('profiles.paste')" @click="handlePaste">
          <icon-tabler-clipboard width="17" height="17"/>
        </UiIconButton>
        <UiIconButton :label="$t('profiles.open')" @click="openFile">
          <icon-tabler-folder-open width="17" height="17"/>
        </UiIconButton>
      </div>
    </div>

    <div class="px-page-body">
      <div v-if="firstLoad && profiles.length === 0" class="profiles-grid">
        <UiSkeleton :count="3" :min-height="200"/>
      </div>
      <UiEmpty v-else-if="profiles.length === 0"
               :icon="IconUserCog"
               :title="t('empty.profiles.title')"
               :text="t('empty.profiles.text')"
               :action-label="t('profiles.add')"
               :action-icon="IconPlus"
               @action="handleAdd"/>
      <VDContainer
          v-else
          class="profiles-vdc"
          :data="profiles"
          @getData="sendOrder"
          :gap="14"
          :draggable="canDrag"
      >
        <template v-slot:VDC="{data,index}">
          <div
              class="profile-card"
              :class="{ 'is-selected': data.selected }"
              role="button"
              tabindex="0"
              :aria-pressed="data.selected ? 'true' : 'false'"
              :aria-label="getProfileDisplayTitle(data)"
              @click="switchProfile(data, true, true)"
          >
            <div class="card-head">
              <span class="card-grip"
                    aria-hidden="true"
                    @mouseenter.stop="mouseEnter"
                    @mouseleave.stop="mouseLeave"
                    @click.stop>
                <icon-tabler-grip-vertical width="16" height="16"/>
              </span>
              <span class="card-title ellipsis" v-tip="getProfileDisplayTitle(data)">{{ getProfileDisplayTitle(data) }}</span>
              <UiIconButton v-if="data.type == 1" :size="26" :label="$t('refresh')" @click.stop="refresh(data)">
                <icon-tabler-refresh width="15" height="15"/>
              </UiIconButton>
            </div>
            <div class="px-divider card-divider"></div>
            <div class="card-stats">
              <div v-if="hasValue(data.used)" class="stat-row">
                <span class="stat-label"><icon-tabler-activity width="14" height="14"/>{{ $t('profiles.use') }}</span>
                <span class="stat-value tabular">{{ formatTrafficValue(data.used) }}</span>
              </div>
              <div v-if="hasValue(data.available)" class="stat-row">
                <span class="stat-label"><icon-tabler-database width="14" height="14"/>{{ $t('profiles.available') }}</span>
                <span class="stat-value tabular">{{ formatTrafficValue(data.available) }}</span>
              </div>
              <div v-if="hasValue(data.expire)" class="stat-row">
                <span class="stat-label"><icon-tabler-calendar-due width="14" height="14"/>{{ $t('profiles.expire') }}</span>
                <span class="stat-value">{{ formatDateValue(data.expire) }}</span>
              </div>
              <div v-if="hasValue(data.update)" class="stat-row">
                <span class="stat-label"><icon-tabler-clock-check width="14" height="14"/>{{ $t('profiles.update') }}</span>
                <span class="stat-value" v-tip="formatExact(data.update)">{{ relativeDate(t, data.update) }}</span>
              </div>
            </div>
            <div class="card-bottom">
              <button v-if="multiProfileEnabled"
                      type="button"
                      class="card-select"
                      :class="{ 'is-selected': data.selected }"
                      :aria-pressed="data.selected ? 'true' : 'false'"
                      :aria-label="t('profiles.multi-select.pick')"
                      v-tip="t('profiles.multi-select.pick')"
                      @click.stop="switchProfile(data, !data.selected)">
                <icon-tabler-circle-check v-if="data.selected" width="18" height="18"/>
                <icon-tabler-circle v-else width="18" height="18"/>
                <span v-if="data.selected && data.selectionOrder" class="card-order tabular">{{ data.selectionOrder }}</span>
              </button>
              <span class="card-spacer"></span>
              <div class="card-actions">
                <UiIconButton v-if="data.content && isHttpOrHttps(data.content)" :size="26" :label="$t('profiles.tv-send')" @click.stop="openTvDialog(data)">
                  <icon-tabler-device-tv width="14" height="14"/>
                </UiIconButton>
                <UiIconButton v-if="data.announce" :size="26" :label="$t('profiles.announce')" @click.stop="showAnnounce(data)">
                  <icon-tabler-speakerphone width="14" height="14"/>
                </UiIconButton>
                <UiIconButton v-if="data.renewUrl" :size="26" :label="$t('profiles.renew')" @click.stop="goRenew(data)">
                  <icon-tabler-credit-card width="14" height="14"/>
                </UiIconButton>
                <UiIconButton v-if="data.support" :size="26" :label="$t('profiles.support')" @click.stop="goSupport(data)">
                  <icon-tabler-headphones width="14" height="14"/>
                </UiIconButton>
                <UiIconButton v-if="data.home" :size="26" :label="$t('profiles.home')" @click.stop="goHome(data)">
                  <icon-tabler-home-link width="14" height="14"/>
                </UiIconButton>
                <UiIconButton :size="26" :label="$t('edit')" @click.stop="updateProfile(data)">
                  <icon-tabler-edit width="14" height="14"/>
                </UiIconButton>
                <UiIconButton :size="26" danger :label="$t('delete')" @click.stop="askDeleteProfile(data, index)">
                  <icon-tabler-trash width="14" height="14"/>
                </UiIconButton>
              </div>
            </div>
          </div>
        </template>
      </VDContainer>
    </div>
  </div>

  <AddProfileDialog v-model="addFormVisible"
                    :initial-content="addInitial"
                    :loading="isNowAdd"
                    @submit="add"/>

  <!-- Edit -->
  <UiModal v-model="editFormVisible" :title="t('edit')" divided>
    <button v-if="editForm.renewUrl" type="button" class="px-btn px-btn--soft px-btn--block" @click="goRenew(editForm)">
      <icon-tabler-credit-card width="15" height="15"/>
      {{ t('profiles.renew') }}
    </button>
    <label class="px-field">
      <span class="px-field__label">{{ t('profiles.edit.title') }}</span>
      <input v-model="editForm.title" class="px-input" autocapitalize="off" autocomplete="off" spellcheck="false">
    </label>
    <template v-if="editForm.type == 1">
      <label class="px-field">
        <span class="px-field__label">{{ t('profiles.edit.url') }}</span>
        <input v-model="editForm.content" class="px-input" autocapitalize="off" autocomplete="off" spellcheck="false">
      </label>
      <div class="px-field">
        <span class="px-field__label" id="edit-interval-label">
          {{ t('profiles.edit.update') }}
          <span v-if="intervalLocked" class="px-info" v-tip="t('profiles.edit.interval-locked')">
            <icon-tabler-lock width="13" height="13"/>
          </span>
        </span>
        <div class="stepper" :class="{ 'is-locked': intervalLocked }" role="group" aria-labelledby="edit-interval-label">
          <button type="button" class="stepper__btn" :disabled="intervalLocked" :aria-label="t('ui.decrease')" v-tip="t('ui.decrease')" @click="stepInterval(-1)">
            <icon-tabler-minus width="14" height="14"/>
          </button>
          <input :value="editForm.interval"
                 class="stepper__input tabular"
                 inputmode="numeric"
                 :disabled="intervalLocked"
                 aria-labelledby="edit-interval-label"
                 @input="onIntervalInput">
          <span class="stepper__unit">{{ t('ui.hours-short') }}</span>
          <button type="button" class="stepper__btn" :disabled="intervalLocked" :aria-label="t('ui.increase')" v-tip="t('ui.increase')" @click="stepInterval(1)">
            <icon-tabler-plus width="14" height="14"/>
          </button>
        </div>
        <span v-if="intervalLocked" class="px-field__hint">{{ t('profiles.edit.interval-locked') }}</span>
        <span v-else-if="intervalInvalid" class="px-field__error" role="alert">{{ t('profiles.edit.update-tip') }}</span>
      </div>
    </template>
    <div class="px-field">
      <span class="px-field__label">{{ t('profiles.edit.template') }}</span>
      <UiSelect v-model="editForm.template"
                :options="templateOptions"
                placement="top"
                :aria-label="t('profiles.edit.template')"
                :disabled="!!editForm.pxdTemplateUrl"/>
    </div>
    <label v-if="editShowAgeKey" class="px-field">
      <span class="px-field__label">age-secret-key</span>
      <input v-model="editForm.ageSecretKey"
             class="px-input"
             autocapitalize="off"
             autocomplete="off"
             spellcheck="false"
             :placeholder="t('age.profile.keyPlaceholder')">
    </label>
    <template #footer-left>
      <span v-if="editForm.hwidActive" class="edit-indicator edit-indicator--accent" v-tip="t('hwid.active.tooltip')" tabindex="0" :aria-label="t('hwid.active.tooltip')">
        <icon-tabler-shield-check width="18" height="18"/>
      </span>
      <UiIconButton v-if="editForm.notifyExpireDays?.length || editForm.notifyTrafficPercent?.length"
                    class="edit-bell"
                    :size="30"
                    :label="t('subscriptionAlert.bellTooltip')"
                    @click="showSubscriptionAlertInfo">
        <icon-tabler-bell width="18" height="18"/>
      </UiIconButton>
      <UiIconButton v-if="editForm.type == 1 || editHasAgeKey"
                    :size="30"
                    :active="editShowAgeKey"
                    :pressed="editShowAgeKey"
                    :label="editHasAgeKey ? t('age.profile.replaceHint') : t('age.profile.toggleOff')"
                    @click="editShowAgeKey = !editShowAgeKey">
        <icon-tabler-key width="18" height="18"/>
      </UiIconButton>
    </template>
    <template #footer>
      <button type="button" class="px-btn" @click="editFormVisible = false">{{ t('cancel') }}</button>
      <button type="button" class="px-btn px-btn--primary" :disabled="isNowEdit" @click="saveUpdateProfile">
        <UiSpinner v-if="isNowEdit"/>
        {{ t('save') }}
      </button>
    </template>
  </UiModal>

  <!-- Subscription reminders configured by the seller (bell in the editor) -->
  <UiModal v-model="subscriptionAlertInfoVisible"
           :title="t('subscriptionAlert.infoTitle')"
           :icon="IconBell"
           tone="warning"
           :width="460"
           :z-index="70">
    <template v-for="(line, i) in subscriptionAlertInfoLines" :key="line">
      <span v-if="i === 0" class="info-line">{{ line }}</span>
      <span v-else-if="line === t('subscriptionAlert.infoDisabledLocally')" class="info-line info-line--warn">{{ line }}</span>
      <span v-else class="info-line info-line--row tabular">{{ line }}</span>
    </template>
    <template #footer>
      <button type="button" class="px-btn px-btn--primary" @click="subscriptionAlertInfoVisible = false">{{ t('close') }}</button>
    </template>
  </UiModal>

  <!-- Multi-profile warning: must be answered, Esc/overlay do nothing -->
  <UiModal v-model="multiProfileInfoVisible"
           :title="t('profiles.multi-select.title')"
           :icon="IconCopyCheck"
           tone="warning"
           :width="520"
           :close-on-esc="false"
           :close-on-overlay="false"
           :show-close="false">
    <span class="info-line">{{ t('profiles.multi-select.description') }}</span>
    <span class="info-line">{{ t('profiles.multi-select.description-secondary') }}</span>
    <span class="info-line">{{ t('profiles.multi-select.description-tertiary') }}</span>
    <div class="px-alert px-alert--warning">
      <icon-tabler-alert-triangle width="16" height="16"/>
      <span style="font-weight:600">{{ t('profiles.multi-select.description-warning') }}</span>
    </div>
    <span class="multi-question">{{ t('profiles.multi-select.question') }}</span>
    <template #footer>
      <button type="button" class="px-btn" @click="declineMultiProfileInfo">{{ t('profiles.multi-select.decline') }}</button>
      <button type="button" class="px-btn px-btn--primary" @click="confirmMultiProfileInfo">{{ t('profiles.multi-select.accept') }}</button>
    </template>
  </UiModal>

  <!-- TV -->
  <UiModal v-model="tvDialogVisible" :title="t('profiles.tv-dialog.title')" :width="380" divided>
    <div class="px-alert px-alert--warning">
      <icon-tabler-alert-triangle width="17" height="17"/>
      <span>{{ tvWarning }}</span>
    </div>
    <label class="px-field">
      <span class="px-field__label">{{ t('profiles.tv-dialog.ip') }}</span>
      <input v-model="tvForm.ip" class="px-input" placeholder="192.168.1.100" autocomplete="off" spellcheck="false">
    </label>
    <label class="px-field">
      <span class="px-field__label">{{ t('profiles.tv-dialog.port') }}</span>
      <input v-model="tvForm.port" class="px-input" placeholder="8080" inputmode="numeric" autocomplete="off" spellcheck="false">
    </label>
    <template #footer>
      <button type="button" class="px-btn" @click="tvDialogVisible = false">{{ t('cancel') }}</button>
      <button type="button" class="px-btn px-btn--primary" :disabled="tvIsSending" @click="submitToTv">
        <UiSpinner v-if="tvIsSending"/>
        <icon-tabler-device-tv v-else width="15" height="15"/>
        {{ t('profiles.tv-dialog.submit') }}
      </button>
    </template>
  </UiModal>

  <!-- Announce -->
  <UiModal v-model="announceDialogVisible" :title="t('profiles.announce')" :icon="IconSpeakerphone" :width="520">
    <div class="announce-box">
      <AnnounceText :text="announceDialogData.text" :url="announceDialogData.url"/>
    </div>
    <template #footer>
      <button type="button" class="px-btn" @click="announceDialogVisible = false">{{ t('close') }}</button>
      <button v-if="announceDialogData.url" type="button" class="px-btn px-btn--primary" @click="goAnnounceUrl">
        <icon-tabler-external-link width="15" height="15"/>
        {{ t('profiles.announce-url') }}
      </button>
    </template>
  </UiModal>
</template>

<style scoped>
.profiles-head {
  justify-content: space-between;
}

.profiles-tools {
  display: flex;
  align-items: center;
  gap: 6px;
}

.profiles-grid,
.profiles-vdc :deep(.vdc-trans-group-container) {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 14px;
}

.profiles-vdc :deep(.vdc-item-container) {
  min-width: 0;
}

.profile-card {
  height: 100%;
  display: flex;
  flex-direction: column;
  border: 2px solid var(--border);
  border-radius: 12px;
  padding: 14px 16px;
  cursor: pointer;
  background: var(--input-bg);
  color: var(--text);
  transition: border-color .15s, background .15s;
}

.profile-card:hover {
  border-color: color-mix(in srgb, var(--accent) 55%, var(--border));
}

.profile-card.is-selected {
  border-color: var(--accent);
  background: color-mix(in srgb, var(--accent) 10%, var(--input-bg));
}

.card-head {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

/* The refresh button keeps a 26px hit area without making the head taller
   than the 16px title line (the mockup draws a bare icon there). */
.card-head :deep(.px-icon-btn) {
  margin: -5px -5px -5px 0;
}

.card-grip {
  display: flex;
  color: var(--text-3);
  cursor: grab;
  flex-shrink: 0;
}

.card-title {
  flex: 1;
  font-size: 14px;
  font-weight: 700;
}

.card-divider {
  margin: 10px 0;
}

/* Rows disappear when the value is missing, but the block keeps the height of
   four rows so every card in the grid lines up. */
.card-stats {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 12px;
  color: var(--text-2);
  min-height: 92px;
}

.stat-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-width: 0;
}

.stat-label {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.stat-label svg {
  flex-shrink: 0;
}

.stat-value {
  color: var(--text);
  font-weight: 600;
  white-space: nowrap;
  flex-shrink: 0;
}

.card-bottom {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 10px;
}

.card-spacer {
  flex: 1;
}

.card-actions {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.card-select {
  display: flex;
  align-items: center;
  gap: 2px;
  border: none;
  background: transparent;
  color: var(--text-3);
  cursor: pointer;
  padding: 2px;
  border-radius: 999px;
}

.card-select.is-selected {
  color: var(--accent);
}

.card-order {
  font-size: 11px;
  font-weight: 700;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  border-radius: 999px;
  background: var(--accent);
  color: var(--on-accent);
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.stepper {
  display: flex;
  align-items: center;
  gap: 4px;
  width: 176px;
  height: 38px;
  padding: 4px;
  border-radius: 8px;
  border: 1px solid var(--border);
  background: var(--input-bg);
}

.stepper:focus-within {
  border-color: var(--accent);
}

.stepper.is-locked {
  opacity: .55;
}

.stepper__btn {
  width: 28px;
  height: 28px;
  border-radius: 8px;
  border: none;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: var(--text-2);
  background: var(--panel-soft);
  cursor: pointer;
  padding: 0;
}

.stepper__btn:hover:not(:disabled) {
  background: var(--hover-bg);
  color: var(--text);
}

.stepper__btn:disabled {
  cursor: not-allowed;
}

.stepper__input {
  flex: 1;
  min-width: 0;
  width: 40px;
  border: none;
  background: transparent;
  color: var(--text);
  font-size: 14px;
  font-weight: 600;
  text-align: center;
  outline: none;
}

.stepper__unit {
  font-size: 12px;
  color: var(--text-3);
}

.edit-indicator {
  display: flex;
  padding: 6px;
}

.edit-indicator--accent {
  color: var(--accent);
}

.edit-bell {
  color: var(--warning);
}

.info-line {
  font-size: 13px;
  color: var(--text-2);
  line-height: 1.6;
  text-wrap: pretty;
}

.info-line--row {
  color: var(--text);
  padding: 8px 12px;
  border-radius: 8px;
  background: var(--panel-soft);
}

.info-line--warn {
  color: var(--warning);
  font-weight: 600;
}

.multi-question {
  font-size: 14px;
  font-weight: 700;
}

.announce-box {
  padding: 16px;
  border-radius: 12px;
  background: var(--panel-soft);
  font-size: 14px;
  line-height: 1.6;
  color: var(--text);
}
</style>
