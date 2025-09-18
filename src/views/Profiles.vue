<script setup lang="ts">
import {Profile} from "@/types/profile";
import createApi from "@/api";
import {pError, pLoad, pSuccess, pWarning} from "@/util/pLoad";
import {useProxiesStore} from "@/store/proxiesStore";
import {useMenuStore} from "@/store/menuStore";
import {getTemplateTitle, isHttpOrHttps, prettyBytes} from "@/util/format";
import {useI18n} from "vue-i18n";
import {Browser, Clipboard, Events} from "@/runtime"
import {useWebStore} from "@/store/webStore";
import {WS} from "@/util/ws";
import {onBeforeRouteLeave} from "vue-router";

// i18n
const {t} = useI18n();

// 获取当前 Vue 实例的 proxy 对象
const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);

// 当前页面使用store
const menuStore = useMenuStore();
const proxiesStore = useProxiesStore();
const webStore = useWebStore();

const hasUsageValue = (value: unknown) => value !== undefined && value !== null && value !== ''

const formatBytes = (value: unknown) => {
  if (!hasUsageValue(value)) {
    return ''
  }
  const number = typeof value === 'number' ? value : Number(value)
  if (!Number.isFinite(number)) {
    return ''
  }
  return prettyBytes(number)
}

const formatDate = (value: unknown) => {
  if (!hasUsageValue(value)) {
    return ''
  }

  if (typeof value === 'string') {
    const match = value.match(/^(\d{4})-(\d{2})-(\d{2})$/)
    if (match) {
      const [, year, month, day] = match
      return `${day}.${month}.${year}`
    }
  }

  const date = new Date(value as any)
  if (Number.isNaN(date.getTime())) {
    return `${value}`
  }

  const day = String(date.getDate()).padStart(2, '0')
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const year = date.getFullYear()

  return `${day}.${month}.${year}`
}

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

// 列表显示
let profiles = reactive<any[]>([])

async function getProfileList() {
  if (profiles.length != 0) {
    profiles.splice(0, profiles.length)
  }
  const list = await api.getProfileList()
  if (list && list.length != 0) {
    list.forEach(item => {
      profiles.push(item)
    })

    Events.Emit({
      name: "profiles",
      data: list
    })

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

// 切换订阅配置
async function switchProfile(data: any) {
  if (data['selected']) {
    return
  }

  await pLoad(t('profiles.switch.ing'), async () => {
    try {
      await api.switchProfile(data)
      proxiesStore.active = ""

      await api.waitRunning()

      for (let profile of profiles) {
        if (profile['selected']) {
          profile['selected'] = false
        }
      }
      data['selected'] = true

      api.getRuleNum().then((res) => {
        menuStore.setRuleNum(res);
      });

      Events.Emit({
        name: "profiles",
        data: toRaw(profiles)
      })

      // 关闭之前的连接
      api.closeAllConnection()

      pSuccess(t('profiles.switch.success'))
    } catch (e) {
      if (e['message']) {
        pError(e['message'])
      }
    }
  })

}


watch(() => webStore.fProfile, async (data: any) => {
  for (let profile of profiles) {
    if (profile['selected']) {
      profile['selected'] = false
    }
    if (profile['id'] == data['id']) {
      data = profile
    }
  }

  data['selected'] = true
})


// 更新订阅
async function refresh(data: any) {
  await pLoad(t('profiles.refresh.ing'), async () => {
    try {
      const re = await api.refreshProfile(data)
      Object.assign(data, re);
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
function goHome(data: any) {
  Browser.OpenURL(data.home)
}

function goSupport(data: any) {
  Browser.OpenURL(data.support)
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
  if (data['selected']) {
    pWarning(t('profiles.del-tip'))
    return
  }

  try {
    await api.deleteProfile(data)
    profiles.splice(index, 1)
    Events.Emit({
      name: "profiles",
      data: toRaw(profiles)
    })
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

function handleProfilesImported(event: Event) {
  const customEvent = event as CustomEvent;
  const detail = customEvent.detail;
  if (!detail || !Array.isArray(detail.profiles)) {
    return;
  }

  let added = false;
  for (const item of detail.profiles) {
    if (!item) {
      continue;
    }
    const exists = profiles.some(profile => profile['id'] === item['id']);
    if (!exists) {
      profiles.push(item);
      added = true;
    }
  }

  if (added) {
    sendOrder(profiles);
  }
}

// 路由切换前关闭 WebSocket
onBeforeRouteLeave(() => {
  wsOrder.close();
});
onBeforeUnmount(() => {
  wsOrder.close();
  window.removeEventListener('deeplink-profile-imported', handleProfilesImported as EventListener);
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
                @click="switchProfile(data)"
            >
              <div class="system-info">
                <div class="profile-title">
                  <el-icon
                      @mouseenter.stop="mouseEnter"
                      @mouseleave.stop="mouseLeave"
                      size="22"
                      class="drag">
                    <icon-mdi-drag/>
                  </el-icon>
                  <span class="profile-title-text" :title="data.title">
                    {{ data.title }}
                  </span>
                  <el-tooltip
                      v-if="data.type == 1"
                      :content="$t('refresh')"
                      placement="top">
                    <el-icon size="22"
                             class="ops row-refresh"
                             @click.stop="refresh(data)">
                      <icon-mdi-refresh/>
                    </el-icon>
                  </el-tooltip>
                </div>
                <div
                    v-if="hasUsageValue(data.available) || hasUsageValue(data.used) || hasUsageValue(data.expire) || hasUsageValue(data.update)"
                    class="profile-stats"
                >
                  <div class="profile-stat" v-if="hasUsageValue(data.available)">
                    <div class="profile-stat-label">
                      <icon-mdi-database-check/>
                      <span>{{ $t('profiles.available') }}</span>
                    </div>
                    <div class="profile-stat-value">
                      {{ formatBytes(data.available) }}
                    </div>
                  </div>
                  <div class="profile-stat" v-if="hasUsageValue(data.used)">
                    <div class="profile-stat-label">
                      <icon-mdi-arrow-up-bold-box-outline/>
                      <span>{{ $t('profiles.use') }}</span>
                    </div>
                    <div class="profile-stat-value">
                      {{ formatBytes(data.used) }}
                    </div>
                  </div>
                  <div class="profile-stat" v-if="hasUsageValue(data.expire)">
                    <div class="profile-stat-label">
                      <icon-mdi-timer-outline/>
                      <span>{{ $t('profiles.expire') }}</span>
                    </div>
                    <div class="profile-stat-value">
                      {{ formatDate(data.expire) }}
                    </div>
                  </div>
                  <div class="profile-stat" v-if="hasUsageValue(data.update)">
                    <div class="profile-stat-label">
                      <icon-mdi-update/>
                      <span>{{ $t('profiles.update') }}</span>
                    </div>
                    <div class="profile-stat-value">
                      {{ formatDate(data.update) }}
                    </div>
                  </div>
                </div>
                <div class="bottom-row">
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
  display: flex;
  flex-direction: column;
  min-height: 190px;
}

.sub-card:hover, .sub-card-select {
  background-color: var(--left-item-selected-bg);
  border: 2px solid var(--text-color);
  cursor: pointer;
}

.sub-card-select:hover {
  cursor: default;
}


.row-refresh {
  margin-left: auto;
}

.system-info {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 6px 12px 10px 12px;
  color: var(--text-color);
  flex: 1;
}

.profile-title {
  display: flex;
  align-items: center;
  gap: 10px;
  font-weight: 600;
  font-size: 15px;
  color: var(--text-color);
}

.profile-title-text {
  flex: 1;
  text-align: center;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.profile-title .drag:hover {
  cursor: grab;
}

.ops:hover {
  cursor: pointer;
}

.profile-stats {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  grid-auto-flow: row dense;
}

.profile-stat {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 6px 8px;
  border-radius: 6px;
  background-color: rgba(255, 255, 255, 0.05);
  background-color: color-mix(in srgb, var(--text-color) 8%, transparent);
}

.profile-stat-label {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--placeholder-color);
  font-size: 12px;
}

.profile-stat-value {
  color: var(--text-color);
  font-weight: 600;
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.bottom-row {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: auto;
  padding-top: 4px;
  color: var(--text-color);
  align-items: center;
}


</style>
