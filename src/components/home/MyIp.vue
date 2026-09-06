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


// 获取 ip 信息
async function getIpInfo(hide: boolean = true) {
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

onMounted(async () => {
  // 每秒更新
  setInterval(updateTimer, 1000);
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
  <div class="ip-system-wrapper">
    <div class="spark">
      <div class="spark-col">
        <div class="box box1">
          <div class="title title-left">
            {{ $t('home.ip.title') }}
            <el-tooltip
                :content="$t('home.ip.service-tip')"
                placement="top">
              <el-icon size="18" class="ipServiceInfo">
                <icon-mdi-information-outline/>
              </el-icon>
            </el-tooltip>
            <el-tooltip
                :content="$t('refresh')"
                placement="top">
              <el-icon size="18"
                       @click="getIpInfo(false)"
                       class="refreshIp">
                <icon-mdi-refresh/>
              </el-icon>
            </el-tooltip>
          </div>
          <hr/>
          <ul class="info-list info-list-left">
            <li class="info-item info-item--link">
              <strong>{{ $t('home.ip.real') }} : </strong>
              <span
                  class="info-item-value"
                  :class="{'info-item-value--link': Boolean(ipInfoLink)}"
              >
                {{ ipInfo['query'] }}
                <a
                    v-if="ipInfoLink"
                    :href="ipInfoLink"
                    class="info-link"
                    :aria-label="$t('home.ip.real')"
                    role="link"
                    @click.prevent.stop="goIpInfo()"
                    @keydown.enter.prevent.stop="goIpInfo()"
                    @keydown.space.prevent.stop="goIpInfo()"
                    tabindex="0"
                >
                  <icon-mdi-open-in-new/>
                </a>
              </span>
            </li>
            <li class="info-item"><strong>{{ $t('home.ip.city') }} : </strong>
              {{ ipInfo['city'] }}
            </li>
            <li class="info-item"><strong>{{ $t('home.ip.country') }} : </strong>
              {{ ipInfo['country'] }}
            </li>
            <li class="info-item"><strong>{{ $t('home.ip.isp') }} : </strong>
              {{ ipInfo['isp'] }}
            </li>
            <li class="info-item info-item--link">
              <strong>{{ $t('home.ip.asn') }} : </strong>
              <span
                  class="info-item-value"
                  :class="{'info-item-value--link': Boolean(asnInfoLink)}"
              >
                {{ ipInfo['as'] }}
                <a
                    v-if="asnInfoLink"
                    :href="asnInfoLink"
                    class="info-link"
                    :aria-label="$t('home.ip.asn')"
                    role="link"
                    @click.prevent.stop="goAsnInfo()"
                    @keydown.enter.prevent.stop="goAsnInfo()"
                    @keydown.space.prevent.stop="goAsnInfo()"
                    tabindex="0"
                >
                  <icon-mdi-open-in-new/>
                </a>
              </span>
            </li>
            <li class="info-item"><strong>{{ $t('home.ip.time-zone') }} : </strong>
              {{ ipInfo['timezone'] }}
            </li>
          </ul>
        </div>
      </div>

      <div class="spark-col">
        <div class="box box2">
          <div class="title title-right">
            {{ $t('home.system.title') }}
          </div>
          <hr/>
          <ul class="info-list info-list-right">
            <li class="info-item"><strong>{{ $t('home.system.os') }} : </strong> {{ homeStore.os }}</li>
            <li class="info-item"><strong>{{ $t('home.system.runtime') }} : </strong>
              {{ time }}
            </li>
            <li class="info-item"><strong>{{ $t('home.system.startup') }} : </strong> {{ settingStore.startup ? $t('on') : $t('off') }}</li>
            <li class="info-item"><strong>{{ $t('home.system.admin') }} : </strong> {{ $t(admin) }}</li>
            <li class="info-item"><strong>{{ $t('home.system.port') }} : </strong>
              {{ port }}
            </li>
            <li class="info-item"><strong>{{ $t('home.system.version') }} : </strong>
              {{ version }}
            </li>
          </ul>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.ip-system-wrapper {
  width: 100%;
  display: flex;
  margin-top: auto;
  overflow: hidden;
  box-sizing: border-box;
}

.spark {
  width: 100%;
  max-width: 100%;
  margin-top: 0;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 15px;
  padding: 0;
}

.spark-col {
  min-width: 0;
}

.box {
  padding: 12px 20px;
  border-radius: 20px;
  background: var(--sub-card-bg);
  border: 1px solid var(--sub-card-border);
  box-shadow: var(--right-box-shadow);
  text-align: left;
  box-sizing: border-box;
}

.box hr {
  border: none;
  height: 1px;
  background-color: var(--hr-color);
  margin: 10px 0;
}

.title {
  position: relative;
  font-weight: 500;
  font-size: 16px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.title-left {
  text-align: left;
  justify-content: flex-start;
}

.title-right {
  text-align: right;
  justify-content: flex-end;
}

.info-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.info-list-left {
  text-align: left;
}

.info-list-right {
  text-align: right;
}

.info-list li {
  font-size: 18px;
  margin: 8px 0;
  line-height: 20px;
}

.info-item--link {
  position: static;
  padding-right: 0;
}

.info-item-value {
  position: relative;
  display: inline-block;
}

.info-item-value--link .info-link {
  position: absolute;
  top: 50%;
  left: calc(100% + 6px);
  transform: translateY(-50%);
}

.refreshIp {
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.refreshIp:hover {
  cursor: pointer;
  opacity: 0.8;
}

.ipServiceInfo {
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.ipServiceInfo:hover {
  cursor: pointer;
  opacity: 0.8;
}

.info-link {
  margin-left: 6px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 1em;
  height: 1em;
  color: inherit;
  cursor: pointer;
  text-decoration: none;
  vertical-align: text-bottom;
  line-height: 1;
}

.info-link :deep(svg) {
  display: block;
  width: 0.9em;
  height: 0.9em;
}

.info-link:focus-visible {
  outline: 2px solid var(--el-color-primary);
  outline-offset: 2px;
}
</style>
