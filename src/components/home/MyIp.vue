<script setup lang="ts">


import {useHomeStore} from "@/store/homeStore";
import {useI18n} from "vue-i18n";
import createApi from "@/api";
import {pError} from "@/util/pLoad";
import {useMenuStore} from "@/store/menuStore";
import {useSettingStore} from "@/store/settingStore";
import {Browser} from "@/runtime";

// 获取当前 Vue 实例的 proxy 对象
const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);

const {t} = useI18n()
const homeStore = useHomeStore()
const menuStore = useMenuStore()
const settingStore = useSettingStore()

// 预计算常量，减少重复运算
const dayInMs = 1000 * 60 * 60 * 24;
const hourInMs = 1000 * 60 * 60;
const minuteInMs = 1000 * 60;

// 优化计时器更新函数
function updateTimer() {
  const elapsed = Date.now() - homeStore.startTime; // 使用 `Date.now()` 获取当前时间戳

  // 将时间差转换为天、时、分、秒
  const days = Math.floor(elapsed / dayInMs);
  const hours = Math.floor((elapsed % dayInMs) / hourInMs);
  const minutes = Math.floor((elapsed % hourInMs) / minuteInMs);
  const seconds = Math.floor((elapsed % minuteInMs) / 1000);

  let show = `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`

  if (days) {
    show = `${days} ${t('home.system.day')} ` + show
  }

  // 更新计时器显示
  time.value = show;
}

// 页面变量
const time = ref("");
const admin = ref("off");
const version = ref("");
const port = ref("");
const ipInfo = ref({
  query: '',
  regionName: '',
  country: '',
  city: '',
  isp: '',
  timezone: '',
  as: '',
})

const ipInfoLink = computed(() => {
  if (!ipInfo.value.query) {
    return ''
  }
  return `https://ipinfo.io/${encodeURIComponent(ipInfo.value.query)}`
})

const asnInfoLink = computed(() => {
  if (!ipInfo.value.as) {
    return ''
  }
  return `https://ipinfo.io/${encodeURIComponent(ipInfo.value.as)}`
})

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

function goIpInfo() {
  openExternalLink(ipInfoLink.value)
}

function goAsnInfo() {
  openExternalLink(asnInfoLink.value)
}


const ipLoading = ref(false);

// 获取 ip 信息
async function getIpInfo(hide: boolean = true) {
  ipLoading.value = true;
  try {
    await loadIpInfo(hide);
  } finally {
    ipLoading.value = false;
  }
}

async function loadIpInfo(hide: boolean) {
  ipInfo.value = homeStore.ip;
  let md6: string
  try {
    // 切换节点后才进行 ip 请求
    md6 = await api.getGroupMd5()
    md6 += menuStore.language
    if (homeStore.md6 === md6) {
      return
    }

    // 进行ip探测
    const url = "http://ip-api.com/json/?lang=" + t('lang');
    const data = await api.getWebTestIp({url});
    data['as'] = data['as'].split(" ")[0];

    // 绑定数据
    ipInfo.value = data;
    homeStore.setIp(data)

    // 存储更新标志
    homeStore.setMd6(md6)

  } catch (e) {
    await getIpInfoFallback(md6)
    if (hide) {
      // 隐藏错误提示
      return
    }
    // 显示错误提示
    if (e['message']) {
      pError(e['message'])
    }
  }
}

async function getIpInfoFallback(md6: string) {
  try {
    // 进行ip探测
    const url = "https://ipwhois.app/json/?lang=" + t('lang');
    const fullIpData = await api.getWebTestIp({url});

    // 绑定数据
    ipInfo.value = {
      query: fullIpData.ip,
      regionName: fullIpData.region,
      country: fullIpData.country,
      city: fullIpData.city,
      isp: fullIpData.isp,
      timezone: fullIpData.timezone,
      as: fullIpData.asn,
    }
    homeStore.setIp(ipInfo.value)

    // 存储更新标志
    homeStore.setMd6(md6)
  } catch (e) {
  }
}

let timer: number | undefined;
onBeforeUnmount(() => window.clearInterval(timer));

onMounted(async () => {
  // 每秒更新
  updateTimer();
  timer = window.setInterval(updateTimer, 1000);
  // 获取版本
  version.value = await api.getVersion()
  // 获取端口
  const configs = await api.getConfigs();
  port.value = configs['mixed-port'];
  // 获取ip
  await getIpInfo(true)

  // 检测是否运行在管理员模式下
  const res = await api.getAdmin();
  if (res.data) {
    admin.value = "on"
  } else {
    admin.value = "off"
  }
})

</script>

<template>
  <div class="home-bottom">
    <section class="px-card info-card" :aria-label="$t('home.ip.title')">
      <div class="info-card__head">
        <span class="info-card__title">{{ $t('home.ip.title') }}</span>
        <span class="px-info" v-tip="$t('home.ip.service-tip')" tabindex="0" :aria-label="$t('home.ip.service-tip')">
          <icon-tabler-info-circle width="14" height="14"/>
        </span>
        <UiIconButton class="info-card__refresh" :label="$t('refresh')" :size="22" :loading="ipLoading" @click="getIpInfo(false)">
          <UiSpinner v-if="ipLoading" :size="12"/>
          <icon-tabler-refresh v-else width="14" height="14"/>
        </UiIconButton>
      </div>
      <div class="px-divider info-card__divider"></div>
      <ul class="info-list">
        <li>
          <strong>{{ $t('home.ip.real') }}:</strong>
          <span class="ellipsis tabular">{{ ipInfo['query'] }}</span>
          <button v-if="ipInfoLink" type="button" class="info-link" :aria-label="'ipinfo.io'" v-tip="'ipinfo.io'" @click="goIpInfo()">
            <icon-tabler-external-link width="13" height="13"/>
          </button>
        </li>
        <li><strong>{{ $t('home.ip.city') }}:</strong> <span class="ellipsis">{{ ipInfo['city'] }}</span></li>
        <li><strong>{{ $t('home.ip.country') }}:</strong> <span class="ellipsis">{{ ipInfo['country'] }}</span></li>
        <li><strong>{{ $t('home.ip.isp') }}:</strong> <span class="ellipsis" v-tip="ipInfo['isp']">{{ ipInfo['isp'] }}</span></li>
        <li>
          <strong>{{ $t('home.ip.asn') }}:</strong>
          <span class="ellipsis">{{ ipInfo['as'] }}</span>
          <button v-if="asnInfoLink" type="button" class="info-link" :aria-label="'ipinfo.io'" v-tip="'ipinfo.io'" @click="goAsnInfo()">
            <icon-tabler-external-link width="13" height="13"/>
          </button>
        </li>
        <li><strong>{{ $t('home.ip.time-zone') }}:</strong> <span class="ellipsis">{{ ipInfo['timezone'] }}</span></li>
      </ul>
    </section>

    <section class="px-card info-card" :aria-label="$t('home.system.title')">
      <div class="info-card__title info-card__title--right">{{ $t('home.system.title') }}</div>
      <div class="px-divider info-card__divider"></div>
      <ul class="info-list info-list--right">
        <li><strong>{{ $t('home.system.os') }}:</strong> <span class="ellipsis">{{ homeStore.os }}</span></li>
        <li><strong>{{ $t('home.system.runtime') }}:</strong> <span class="tabular">{{ time }}</span></li>
        <li><strong>{{ $t('home.system.startup') }}:</strong> {{ settingStore.startup ? $t('on') : $t('off') }}</li>
        <li><strong>{{ $t('home.system.admin') }}:</strong> {{ $t(admin) }}</li>
        <li><strong>{{ $t('home.system.port') }}:</strong> <span class="tabular">{{ port }}</span></li>
        <li><strong>{{ $t('home.system.version') }}:</strong> <span class="ellipsis">{{ version }}</span></li>
      </ul>
    </section>
  </div>
</template>

<style scoped>
.home-bottom {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
  margin-top: auto;
}

.info-card {
  min-width: 0;
}

.info-card__head {
  display: flex;
  align-items: center;
  gap: 8px;
}

.info-card__title {
  font-size: 14px;
  font-weight: 700;
}

.info-card__title--right {
  text-align: right;
}

.info-card__refresh {
  margin-left: auto;
}

.info-card__divider {
  margin: 10px 0;
}

.info-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 7px;
  font-size: 13px;
  color: var(--text-2);
}

.info-list li {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
}

.info-list strong {
  color: var(--text);
  font-weight: 600;
  flex-shrink: 0;
}

.info-list--right li {
  justify-content: flex-end;
  text-align: right;
}

.info-link {
  display: inline-flex;
  padding: 2px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--text-3);
  cursor: pointer;
  flex-shrink: 0;
}

.info-link:hover {
  color: var(--accent);
  background: var(--hover-bg);
}
</style>
