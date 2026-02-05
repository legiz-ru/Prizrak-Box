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

// 头部几个按钮操作
const addFormVisible = ref(false)
const isNowAdd = ref(false)
const addForm = reactive({
  content: '',
})

async function add() {
  if (!addForm.content) {
    return
  }

  isNowAdd.value = true
  const p = new Profile()
  p.content = addForm.content
  try {
    const pList = await api.addProfileFromInput(p)
    if (pList && pList.length > 0) {
      pList.forEach(item => profiles.push(item))
    }
    sendOrder(profiles)
    addForm.content = ""
    addFormVisible.value = false
  } catch (e) {
    if (e['message']) {
      pError(e['message'])
    }
  }
  isNowAdd.value = false
}

function handleAdd() {
  addForm.content = ""
  addFormVisible.value = true
}

function handlePaste() {
  addForm.content = Clipboard.Text()
  addFormVisible.value = true
}

function openFile() {
  webStore.dnd = true
}

function hasValue(value: any) {
  return value !== undefined && value !== null && value !== ''
}

function formatTrafficValue(value: any) {
  if (!hasValue(value)) {
    return ''
  }
  const num = Number(value)
  if (Number.isFinite(num)) {
    return prettyBytes(num)
  }
  return String(value)
}

function formatDateValue(value: any) {
  if (!hasValue(value)) {
    return ''
  }

  if (typeof value === 'string') {
    const trimmed = value.trim()
    const match = trimmed.match(/^(\d{4})[-/.](\d{2})[-/.](\d{2})$/)
    if (match) {
      return `${match[3]}.${match[2]}.${match[1]}`
    }

    const parsed = Date.parse(trimmed)
    if (!Number.isNaN(parsed)) {
      const date = new Date(parsed)
      const day = String(date.getDate()).padStart(2, '0')
      const month = String(date.getMonth() + 1).padStart(2, '0')
      const year = date.getFullYear()
      return `${day}.${month}.${year}`
    }

    return trimmed
  }

  if (typeof value === 'number') {
    const timestamp = value > 1e12 ? value : value * 1000
    const date = new Date(timestamp)
    if (!Number.isNaN(date.getTime())) {
      const day = String(date.getDate()).padStart(2, '0')
      const month = String(date.getMonth() + 1).padStart(2, '0')
      const year = date.getFullYear()
      return `${day}.${month}.${year}`
    }
  }

  if (value instanceof Date && !Number.isNaN(value.getTime())) {
    const day = String(value.getDate()).padStart(2, '0')
    const month = String(value.getMonth() + 1).padStart(2, '0')
    const year = value.getFullYear()
    return `${day}.${month}.${year}`
  }

  return String(value)
}

// 列表显示
const flagEmojiRegex = /([\u{1F1E6}-\u{1F1FF}]{2}|\u{1F3F3}|\u{1F3F4}|\u{1F6A9})/u

function containsFlagEmoji(value: any) {
  if (typeof value !== 'string') {
    return false
  }
  return flagEmojiRegex.test(value)
}

function getProfileDisplayTitle(profile: any) {
  const title = typeof profile?.title === 'string' ? profile.title.trim() : ''
  const headerTitle = typeof profile?.headerTitle === 'string' ? profile.headerTitle.trim() : ''

  if (title) {
    if (!headerTitle) {
      return title
    }
    if (containsFlagEmoji(title) || !containsFlagEmoji(headerTitle)) {
      return title
    }
  }

  return headerTitle || title || ''
}

let profiles = reactive<any[]>([])

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
  if (multiProfileEnabled.value && !settingStore.multiProfileHintShown) {
    const selectedProfiles = profiles.filter(profile => profile['selected'])
    if (selectedProfiles.length > 1) {
      multiProfileInfoVisible.value = true
    }
  }
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
  const list = await api.getProfileList()
  if (list && list.length != 0) {
    applyProfileList(list)
    Events.Emit({
      name: "profiles",
      data: list
    })
  } else {
    applyProfileList([])
  }
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
        if (multiProfileEnabled.value && !settingStore.multiProfileHintShown) {
          const selectedProfiles = profiles.filter(profile => profile['selected'])
          if (selectedProfiles.length > 1) {
            multiProfileInfoVisible.value = true
          }
        }

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

  const selectedCount = profiles.filter(profile => profile['selected']).length
  if (selectedCount > 1 && !settingStore.multiProfileHintShown) {
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
let editForm = reactive<any>({})
let editFormD = {}

function updateProfile(data: any) {
  editFormD = data
  editForm = reactive<any>({})
  Object.assign(editForm, data)
  editFormVisible.value = true
}

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
  await api.updateProfile(editForm)
  isNowEdit.value = false
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
      pWarning('Удаление прошло, выберите новый активный профиль')
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
    applyProfileList(list ?? []);
    sendOrder(profiles);
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
let tList = reactive([]);

// vue 周期相关
onMounted(async () => {
  const urlTraffic = webStore.wsUrl + "/profile/order?token=" + webStore.secret;
  wsOrder = new WS(urlTraffic);

  await getProfileList()
  tList = await api.getTemplateList();
  tList.unshift({
    title: 'm0',
    id: 'm0'
  });

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
  <MyLayout>
    <template #top>
        <el-space class="space">
          <div class="title">
            {{ $t('profiles.title') }}
          </div>
          <div class="profile-option">
            <el-tooltip
                :content="multiProfileEnabled ? t('profiles.multi-select.disable') : t('profiles.multi-select.enable')"
                placement="top">
              <el-icon
                  @click="toggleMultiProfile"
                  class="profile-option-btn">
                <icon-mdi-checkbox-multiple-marked v-if="multiProfileEnabled"/>
                <icon-mdi-checkbox-multiple-blank-outline v-else/>
              </el-icon>
            </el-tooltip>
            <el-tooltip
                :content="t('profiles.multi-select.info')"
                placement="top">
              <el-icon
                  @click="showMultiProfileInfo"
                  class="profile-option-btn">
                <icon-mdi-information-outline/>
              </el-icon>
            </el-tooltip>
            <el-tooltip
                :content="$t('profiles.add')"
                placement="top">
              <el-icon
                  @click="handleAdd"
                class="profile-option-btn">
              <icon-mdi-plus-thick/>
            </el-icon>
          </el-tooltip>

          <el-tooltip
              :content="$t('profiles.paste')"
              placement="top">
            <el-icon
                @click="handlePaste"
                class="profile-option-btn">
              <icon-mdi-content-paste/>
            </el-icon>
          </el-tooltip>

          <el-tooltip
              :content="$t('profiles.open')"
              placement="top">
            <el-icon
                @click="openFile"
                class="profile-option-btn">
              <icon-mdi-folder-open/>
            </el-icon>
          </el-tooltip>
        </div>
      </el-space>

    </template>

    <template #bottom>
      <VDContainer
          :data="profiles"
          @getData="sendOrder"
          :gap="15"
          :draggable="canDrag"
          style="margin-left: 10px;width: 95%;"
      >
        <template v-slot:VDC="{data,index}">
          <div
              :class="data.selected?'sub-card sub-card-select':'sub-card'"
              @click="switchProfile(data, true, true)"
          >
            <div class="row card-header">
              <el-icon
                  @mouseenter.stop="mouseEnter"
                  @mouseleave.stop="mouseLeave"
                  @click.stop
                  size="22"
                  class="drag">
                <icon-mdi-drag/>
              </el-icon>
              <div class="profile-name" :title="getProfileDisplayTitle(data)">
                <span class="profile-name-text">{{ getProfileDisplayTitle(data) }}</span>
              </div>
              <div class="header-action">
                <el-tooltip
                    v-if="data.type == 1"
                    :content="$t('refresh')"
                    placement="top">
                  <el-icon size="22"
                           class="ops"
                           @click.stop="refresh(data)">
                    <icon-mdi-refresh/>
                  </el-icon>
                </el-tooltip>
              </div>
            </div>
            <div class="stats">
              <div class="stat-row" v-if="hasValue(data.used)">
                <el-icon size="18" class="stat-icon">
                  <icon-mdi-chart-timeline-variant/>
                </el-icon>
                <span class="stat-label">{{ $t('profiles.use') }}</span>
                <span class="stat-value">{{ formatTrafficValue(data.used) }}</span>
              </div>
              <div class="stat-row" v-if="hasValue(data.available)">
                <el-icon size="18" class="stat-icon">
                  <icon-mdi-database-check/>
                </el-icon>
                <span class="stat-label">{{ $t('profiles.available') }}</span>
                <span class="stat-value">{{ formatTrafficValue(data.available) }}</span>
              </div>
              <div class="stat-row" v-if="hasValue(data.expire)">
                <el-icon size="18" class="stat-icon">
                  <icon-mdi-calendar-alert/>
                </el-icon>
                <span class="stat-label">{{ $t('profiles.expire') }}</span>
                <span class="stat-value">{{ formatDateValue(data.expire) }}</span>
              </div>
              <div class="stat-row" v-if="hasValue(data.update)">
                <el-icon size="18" class="stat-icon">
                  <icon-mdi-update/>
                </el-icon>
                <span class="stat-label">{{ $t('profiles.update') }}</span>
                <span class="stat-value">{{ formatDateValue(data.update) }}</span>
              </div>
            </div>
              <div class="bottom-row" :class="{ 'multi-disabled': !multiProfileEnabled }">
              <div class="profile-select" v-if="multiProfileEnabled">
                <button
                    type="button"
                    class="profile-select-btn"
                    :class="{ 'is-selected': data.selected }"
                    @click.stop="switchProfile(data, !data.selected)"
                >
                  <el-icon class="profile-select-icon" size="18">
                    <icon-mdi-check-circle v-if="data.selected"/>
                    <icon-mdi-circle-outline v-else/>
                  </el-icon>
                  <el-icon v-if="data.selected" class="profile-select-order-icon" size="18">
                    <icon-mdi-numeric-1-circle v-if="data.selectionOrder === 1"/>
                    <icon-mdi-numeric-2-circle v-else-if="data.selectionOrder === 2"/>
                    <icon-mdi-numeric-3-circle v-else-if="data.selectionOrder === 3"/>
                    <icon-mdi-numeric-4-circle v-else-if="data.selectionOrder === 4"/>
                    <icon-mdi-numeric-5-circle v-else-if="data.selectionOrder === 5"/>
                    <icon-mdi-numeric-6-circle v-else-if="data.selectionOrder === 6"/>
                    <icon-mdi-numeric-7-circle v-else-if="data.selectionOrder === 7"/>
                    <icon-mdi-numeric-8-circle v-else-if="data.selectionOrder === 8"/>
                    <icon-mdi-numeric-9-circle v-else-if="data.selectionOrder === 9"/>
                    <icon-mdi-numeric-10-circle v-else/>
                  </el-icon>
                </button>
              </div>
              <div class="bottom-actions">
                <el-tooltip
                    v-if="data.announce"
                    :content="$t('profiles.announce')"
                    placement="top">
                  <el-icon
                      class="ops"
                      @click.stop="showAnnounce(data)"
                      size="20">
                    <icon-mdi-bullhorn-variant-outline/>
                  </el-icon>
                </el-tooltip>
                <el-tooltip
                    v-if="data.support"
                    :content="$t('profiles.support')"
                    placement="top">
                  <el-icon
                      class="ops"
                      @click.stop="goSupport(data)"
                      size="20">
                    <icon-mdi-face-agent/>
                  </el-icon>
                </el-tooltip>
                <el-tooltip
                    v-if="data.home"
                    :content="$t('profiles.home')"
                    placement="top">
                  <el-icon
                      class="ops"
                      @click.stop="goHome(data)"
                      size="20">
                    <icon-mdi-home-import-outline/>
                  </el-icon>
                </el-tooltip>
                <el-tooltip
                    :content="$t('edit')"
                    placement="top">
                  <el-icon
                      class="ops"
                      @click.stop="updateProfile(data)"
                      size="20">
                    <icon-mdi-square-edit-outline/>
                  </el-icon>
                </el-tooltip>
                <el-tooltip
                    :content="$t('delete')"
                    placement="top">
                  <el-icon
                      class="ops"
                      @click.stop="deleteProfile(data,index)"
                      size="20">
                    <icon-mdi-trash-can/>
                  </el-icon>
                </el-tooltip>
              </div>
            </div>
          </div>
        </template>
      </VDContainer>

    </template>
  </MyLayout>

  <el-dialog v-model="addFormVisible"
             :title="t('profiles.add')"
             width="520"
             draggable
             center
  >
    <el-form :model="addForm">
      <el-form-item>
        <el-input
            :rows="3"
            type="textarea"
            autocapitalize="off"
            autocomplete="off"
            spellcheck="false"
            :placeholder="t('profiles.placeholder')"
            v-model="addForm.content"
        />
      </el-form-item>
    </el-form>
    <template #footer>
      <div class="dialog-footer">
        <el-button @click="addFormVisible = false">
          {{ t('cancel') }}
        </el-button>
        <el-button
            :loading="isNowAdd"
            type="primary"
            @click="add">
          {{ t('confirm') }}
        </el-button>
      </div>
    </template>
  </el-dialog>

  <el-dialog v-model="editFormVisible"
             :title="t('edit')"
             width="520"
             draggable
             center
  >
    <el-form
        :model="editForm"
        label-position="top"
    >
      <el-form-item
          :label="t('profiles.edit.title')"
          label-width="120">
        <el-input
            v-model="editForm.title"
            clearable
            autocapitalize="off"
            autocomplete="off"
            spellcheck="false"/>
      </el-form-item>
      <el-form-item
          v-if="editForm.type == 1"
          :label="t('profiles.edit.url')"
          label-width="120">
        <el-input
            v-model="editForm.content"
            clearable
            autocapitalize="off"
            autocomplete="off"
            spellcheck="false"/>
      </el-form-item>
      <el-form-item
          v-if="editForm.type == 1"
          :label="t('profiles.edit.update')"
          label-width="120">
        <el-input
            v-model="editForm.interval"
            clearable
            autocapitalize="off"
            autocomplete="off"
            spellcheck="false">
        </el-input>
      </el-form-item>
      <el-form-item
          :label="t('profiles.edit.template')"
          label-width="120">
        <el-select
            v-model="editForm.template"
            placeholder=""
            clearable
        >
          <el-option
              v-for="item in tList"
              :key="item.id"
              :label="getTemplateTitle(t,item.title)"
              :value="item.id"
          />
        </el-select>
      </el-form-item>

    </el-form>
    <template #footer>
      <div class="dialog-footer">
        <el-button @click="editFormVisible = false">
          {{ t('cancel') }}
        </el-button>
        <el-button
            type="primary"
            :loading="isNowEdit"
            @click="saveUpdateProfile"
        >
          {{ t('confirm') }}
        </el-button>
      </div>
    </template>
  </el-dialog>

  <el-dialog
      v-model="multiProfileInfoVisible"
      :title="t('profiles.multi-select.title')"
      width="520"
      draggable
      center
      :close-on-click-modal="false"
      :close-on-press-escape="false"
      :show-close="false"
    >
      <div class="multi-profile-info">
        <p>{{ t('profiles.multi-select.description') }}</p>
        <p>{{ t('profiles.multi-select.description-secondary') }}</p>
        <p>{{ t('profiles.multi-select.description-tertiary') }}</p>
        <p>{{ t('profiles.multi-select.description-warning') }}</p>
        <p class="multi-profile-question">{{ t('profiles.multi-select.question') }}</p>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="declineMultiProfileInfo">
            {{ t('profiles.multi-select.decline') }}
          </el-button>
          <el-button type="primary" @click="confirmMultiProfileInfo">
            {{ t('profiles.multi-select.accept') }}
          </el-button>
        </div>
      </template>
    </el-dialog>

  <!-- Announce Dialog -->
  <el-dialog
      v-model="announceDialogVisible"
      :title="t('profiles.announce')"
      width="520"
      draggable
      center
  >
    <div class="announce-dialog-content">
      <AnnounceText
          :text="announceDialogData.text"
          :url="announceDialogData.url"
      />
    </div>
    <template #footer>
      <div class="dialog-footer">
        <el-button @click="announceDialogVisible = false">
          {{ t('close') }}
        </el-button>
        <el-button
            v-if="announceDialogData.url"
            type="primary"
            @click="goAnnounceUrl"
        >
          {{ t('profiles.announce-url') }}
        </el-button>
      </div>
    </template>
  </el-dialog>

</template>

<style scoped>
.space {
  margin-top: 15px;
}

.title {
  font-size: 32px;
  font-weight: bold;
  margin-left: 10px;
}

.profile-option {
  margin-left: 10px;
  font-size: 30px;
  padding-top: 10px;
}

.profile-option-btn {
  margin-right: 15px;
}

.profile-option-btn:hover {
  cursor: pointer;
  color: var(--hr-color);
}

.multi-profile-info {
  display: flex;
  flex-direction: column;
  gap: 8px;
  color: var(--el-text-color-regular);
  line-height: 1.4;
}

.multi-profile-question {
  font-weight: 600;
}

:deep(.vdc-item-container) {
  width: calc(33% - 10px);
  max-width: 245px;
}

.sub-card {
  padding: 5px 8px 5px 5px;
  border: 2px solid var(--sub-card-border);
  border-radius: 8px;
  background: var(--sub-card-bg);
  color: var(--text-color);
  box-shadow: var(--left-nav-shadow);
  margin-top: 5px;
}

.sub-card:hover, .sub-card-select {
  background-color: var(--left-item-selected-bg);
  border: 2px solid var(--text-color);
  cursor: pointer;
}

.sub-card-select:hover {
  cursor: default;
}

.sub-card .row {
  display: flex;
  justify-content: space-between;
}

.sub-card .row .drag:hover {
  cursor: grab;
}

.ops:hover {
  cursor: pointer;
}

.card-header {
  align-items: center;
  gap: 8px;
  padding: 4px 6px 0 6px;
}

.profile-name {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-width: 0;
  font-weight: 600;
}

.profile-name-text {
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  min-width: 0;
  font-variant-emoji: emoji;
  font-family: 'Twemoji', "Nunito", 'Microsoft YaHei', '微软雅黑', 'Helvetica Neue', Helvetica, 'PingFang SC', 'Hiragino Sans GB',
  Arial, -apple-system, BlinkMacSystemFont, "Segoe UI", "Roboto",
  "Oxygen", "Ubuntu", "Cantarell", "Fira Sans", "Droid Sans", "Helvetica Neue",
  sans-serif;
}

.header-action {
  min-width: 24px;
  display: flex;
  justify-content: center;
  align-items: center;
}

.stats {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 6px 6px 0 6px;
  font-size: 13px;
  color: var(--text-color);
  min-height: 90px;
}

.stat-row {
  display: flex;
  align-items: center;
  gap: 6px;
}

.stat-label {
  flex: 1;
  color: var(--text-color);
}

.stat-value {
  font-weight: 500;
}

.bottom-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 10px;
  margin-bottom: 4px;
  color: var(--text-color);
}

.bottom-row.multi-disabled {
  justify-content: flex-end;
}

.profile-select {
  display: flex;
  align-items: center;
}

.profile-select-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 2px 4px;
  border: none;
  background: transparent;
  color: var(--text-color);
  border-radius: 999px;
  transition: background 0.15s ease, box-shadow 0.15s ease;
}

.profile-select-btn:hover {
  cursor: pointer;
  background: rgba(255, 255, 255, 0.08);
}

.profile-select-icon {
  color: var(--text-color);
  opacity: 0.85;
}

.profile-select-order-icon {
  color: var(--text-color);
  opacity: 0.9;
}

.profile-select-btn.is-selected .profile-select-icon,
.profile-select-btn.is-selected .profile-select-order-icon {
  opacity: 1;
}

.bottom-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
.stat-icon {
  color: var(--text-color);
}

.announce-dialog-content {
  padding: 20px;
  font-size: 14px;
  color: var(--el-text-color-primary);
  text-align: center;
  line-height: 1.6;
  word-wrap: break-word;
  white-space: pre-wrap;
}
</style>
