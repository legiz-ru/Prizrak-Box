<script setup lang="ts">

import MyPort from "@/components/setting/MyPort.vue";
import MyTun from "@/components/setting/MyTun.vue";
import {EditPen} from "@element-plus/icons-vue";
import {useWebStore} from "@/store/webStore";
import {copy} from "@/util/pLoad";
import {useI18n} from "vue-i18n";
import {useSettingStore} from "@/store/settingStore";
import createApi from "@/api";
import {changeMenu} from "@/util/menu";
import {useRouter} from "vue-router";
import {pUpdateMihomo} from "@/util/mihomo";
import {useMenuStore} from "@/store/menuStore";
import {Events} from "@/runtime";

// 获取当前 Vue 实例的 proxy 对象 和 api
const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);

// 使用 store
const webStore = useWebStore()
const menuStore = useMenuStore()
const settingStore = useSettingStore()
const {t} = useI18n()

// 使用路由
const router = useRouter()

// Initialization flag to prevent watcher from firing during initial load
let isInitialLoad = true;

// 数据监听
// dns
watch(() => settingStore.dns, (newValue) => {
  // 更新配置
  api.switchDNS({
    enable: newValue,
  });
});

// ipv6
watch(() => settingStore.ipv6, (newValue) => {
  // 更新配置
  api.updateConfigs({
    ipv6: newValue,
  }).then(() => {
    // 同步 mihomo 配置
    pUpdateMihomo(menuStore, settingStore, api)
  });
});

// hwid
watch(() => settingStore.hwid, async (newValue, oldValue) => {
  console.log(`[HWID Toggle] HWID setting changed from ${oldValue} to ${newValue}, isInitialLoad: ${isInitialLoad}`);
  
  // Skip the initial load
  if (isInitialLoad || oldValue === undefined) {
    console.log(`[HWID Toggle] Skipping - initial load or oldValue is undefined`);
    return;
  }
  
  // 更新后端配置 with retry and verification
  const maxRetries = 3;
  let retryCount = 0;
  
  while (retryCount < maxRetries) {
    try {
      console.log(`[HWID Toggle] Calling API to set HWID to: ${newValue} (attempt ${retryCount + 1}/${maxRetries})`);
      await api.prizrak.setHWIDSetting(newValue);
      console.log(`[HWID Toggle] API call successful, verifying backend state...`);
      
      // Verify the setting was actually saved
      const backendSettings = await api.prizrak.getSettings();
      console.log(`[HWID Toggle] Backend verification - HWID is now: ${backendSettings.hwid}`);
      
      if (backendSettings.hwid === newValue) {
        console.log(`[HWID Toggle] Verification successful: backend HWID matches UI setting (${newValue})`);
        break; // Success, exit retry loop
      } else {
        throw new Error(`Backend verification failed: expected ${newValue}, got ${backendSettings.hwid}`);
      }
    } catch (error) {
      retryCount++;
      console.error(`[HWID Toggle] Failed to update HWID setting (attempt ${retryCount}/${maxRetries}):`, error);
      
      if (retryCount >= maxRetries) {
        console.error(`[HWID Toggle] All retry attempts failed. Could not sync HWID setting to backend.`);
        // Revert the UI setting to match backend
        try {
          const backendSettings = await api.prizrak.getSettings();
          console.log(`[HWID Toggle] Reverting UI to backend state: ${backendSettings.hwid}`);
          settingStore.setHwid(backendSettings.hwid);
        } catch (revertError) {
          console.error(`[HWID Toggle] Could not revert UI setting:`, revertError);
        }
      } else {
        // Wait before retry
        await new Promise(resolve => setTimeout(resolve, 1000));
      }
    }
  }
});

// 开机自启
watch(() => settingStore.startup, (newValue) => {
  // 更新配置
  Events.Emit({name: "boot", data: newValue});
  // 同步 mihomo 配置
  pUpdateMihomo(menuStore, settingStore, api)
});

// 打开配置目录
function pxConfigDir() {
  // @ts-ignore
  api.configDir().then(res => window["pxConfigDir"](res))
}

// 检查更新
function checkUpdate() {
  const url = "https://github.com/legiz-ru/Prizrak-Box/releases"
  // @ts-ignore
  window["pxOpen"](url)
}

// 加载初始设置
onMounted(async () => {
  console.log('[Settings Load] Loading initial settings from backend...');
  try {
    const settings = await api.prizrak.getSettings();
    console.log('[Settings Load] Received settings from backend:', settings);
    
    if (settings && typeof settings.hwid !== 'undefined') {
      console.log(`[Settings Load] Setting HWID in store to: ${settings.hwid} (backend authoritative)`);
      settingStore.setHwid(settings.hwid);
      console.log(`[Settings Load] Current HWID in store after update: ${settingStore.hwid}`);
    } else {
      console.warn('[Settings Load] No HWID setting received from backend');
    }
  } catch (error) {
    console.error('[Settings Load] Failed to load settings:', error);
  } finally {
    // Mark initial load as complete after a small delay
    setTimeout(() => {
      isInitialLoad = false;
      console.log('[Settings Load] Initial load complete, watchers now active');
    }, 1000);
  }
});

</script>

<template>
  <el-row :gutter="20" class="spark"
          style="margin-left: 0;
          margin-top: 2px;
          margin-right: 0;">
    <el-col :span="24">
      <div class="box box1">
        <div class="title">
          Mihomo
        </div>
        <hr/>
        <ul class="info-list">
          <li>
            <MyPort></MyPort>
          </li>
          <li>
            <MyBind></MyBind>
          </li>
          <li>
            <MyTun></MyTun>
          </li>
          <li>
            <strong>
              {{ $t('setting.mihomo.dns') }} :
            </strong>
            <el-icon
                @click="changeMenu('Setting/Dns',router)"
                class="btn">
              <EditPen/>
            </el-icon>
            <el-switch
                v-model="settingStore.dns"
                class="set-switch"
                style="margin-left: 28px"
            />
          </li>
          <li>
            <strong>IPV6 :</strong>
            <el-switch
                v-model="settingStore.ipv6"
                class="set-switch"
            />
          </li>
          <li style="height: 30px">
            <strong>Api :</strong>
            {{ webStore.baseUrl }}
            <el-button
                @click="copy(webStore.baseUrl,t)">
              {{ $t('copy.title') }}
            </el-button>
          </li>
          <li style="height: 30px">
            <strong>Secret:</strong>
            {{ webStore.secret }}
            <el-button
                @click="copy(webStore.secret,t)">
              {{ $t('copy.title') }}
            </el-button>
          </li>
        </ul>
      </div>
    </el-col>
  </el-row>

  <el-row :gutter="20" class="spark"
          style="margin-left: 0;
          margin-top: 30px;
          margin-right: 0;">
    <el-col :span="24">
      <div class="box box2">
        <div class="title">
          Prizrak-Box
        </div>
        <hr/>
        <ul class="info-list">
          <li>
            <strong>HWID :</strong>
            <el-switch
                v-model="settingStore.hwid"
                class="set-switch"
            />
          </li>
          <li>
            <strong>{{ $t('setting.px.startup') }} :</strong>
            <el-switch
                v-model="settingStore.startup"
                class="set-switch"
            />
          </li>
          <li>
            <strong>{{ $t('setting.px.auth') }} :</strong>
            <el-switch
                v-model="settingStore.auth"
                class="set-switch"
            />
          </li>
          <li style="height: 30px">
            <strong>{{ $t('setting.px.dir') }} :</strong>
            <el-button @click="pxConfigDir" style="margin-left: 10px">
              {{ $t('setting.px.open') }}
            </el-button>
            <!--            <el-button>{{ $t('setting.px.export') }}</el-button>-->
            <!--            <el-button>{{ $t('setting.px.import') }}</el-button>-->
          </li>
          <li style="height: 30px">
            <strong>{{ $t('setting.px.update') }} :</strong>
            <el-button @click="checkUpdate" style="margin-left: 10px">{{ $t('setting.px.check') }}</el-button>
          </li>
        </ul>
      </div>
    </el-col>
  </el-row>


</template>

<style scoped>
.spark {
  max-width: 95%;
}

.box {
  padding: 10px;
  border-radius: 8px;
  text-align: left;
}

.box hr {
  border: none;
  height: 1px;
  background-color: var(--hr-color);
  margin: 10px 0;
}

.info-list {
  list-style: none;
  padding: 0;
}

.info-list li {
  font-size: 18px;
  margin: 8px 0;
}

.box1 {
  box-shadow: var(--right-box-shadow);
}

.box2 {
  box-shadow: var(--right-box-shadow);
}

.set-switch {
  margin-left: 10px;
  --el-switch-border-color: var(--text-color);
  --el-switch-on-color: var(--left-item-selected-bg);
  --el-switch-off-color: transparent;
}

:deep(.el-switch__core) {
  width: 46px;
  height: 26px;
  border-radius: 12px;
  border: 2px solid var(--text-color);
}

:deep(.el-switch__core .el-switch__action) {
  margin-left: 2px;
  background-color: var(--text-color);
}

:deep(.el-switch.is-checked .el-switch__core .el-switch__action) {
  left: calc(100% - 21px);
}

:deep(.el-button) {
  padding: 2px 10px;
  --el-button-bg-color: transparent;
  --el-button-text-color: var(--text-color);
  --el-button-hover-text-color: var(--left-item-selected-bg);
  --el-button-hover-bg-color: var(--text-color)
}

.btn {
  font-size: 18px;
  position: absolute;
  margin-top: 6px;
}

.btn:hover {
  cursor: pointer;
  color: var(--hr-color);
}


</style>