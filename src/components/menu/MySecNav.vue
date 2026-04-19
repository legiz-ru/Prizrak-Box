<script setup lang="ts">
import {changeMenu} from "@/util/menu";
import {useMenuStore} from "@/store/menuStore";
import {useRouter} from "vue-router";
import {WS} from "@/util/ws";
import {useWebStore} from "@/store/webStore";
import {formatDate} from "@/util/format";
import createApi from "@/api";
import {logLevel} from "@/composables/logLevel";

// 获取当前 Vue 实例的 proxy 对象 和 api
const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);

// 获取Store
const menuStore = useMenuStore();
const webStore = useWebStore();

// 获取路由
const router = useRouter();

const conn = ref(0);

// 连接数
function onConn(ev: MessageEvent) {
  const parsedData = JSON.parse(ev.data);
  if (parsedData["connections"]) {
    conn.value = parsedData["connections"].length;
  } else {
    conn.value = 0;
  }
}

// 日志
function onLog(ev: MessageEvent) {
  const parsedData = JSON.parse(ev.data);
  webStore.addLog({
    time: formatDate(new Date()),
    type: parsedData["type"].toUpperCase(),
    payload: parsedData["payload"]
  });
}

function aliveTest(conn: WS, cb: Function) {
  setInterval(() => {
    try {
      if (conn.ws.readyState === WebSocket.OPEN) {
        conn.ws.send("ping");
      } else {
        console.log("WebSocket 连接可能已断开");
        if (cb) {
          cb()
        }
      }
    } catch (error) {
      console.error("发送失败，WebSocket 可能已断开:", error);
      if (cb) {
        cb()
      }
    }
  }, 10000);
}

let wsConn: WS | null = null;
let logConn: WS | null = null;

function buildLogUrl() {
  const level = logLevel.value;
  const levelParam = level ? `&level=${encodeURIComponent(level)}` : '';
  return webStore.wsUrl + "/logs?token=" + webStore.secret + levelParam;
}

function connectLog(clearOnReconnect = false) {
  if (logConn) {
    logConn.close();
    logConn = null;
  }
  if (clearOnReconnect) {
    webStore.clearLogs();
  }
  const url = buildLogUrl();
  logConn = new WS(url, null, onLog);
  aliveTest(logConn, () => {
    logConn?.close();
    logConn = null;
    connectLog(false);
  });
}

// Reconnect with the new level whenever it changes (user-triggered)
watch(logLevel, (newVal, oldVal) => {
  if (newVal !== oldVal) {
    connectLog(true);
  }
});

onMounted(() => {
  const urlTraffic = webStore.wsUrl + "/connections?token=" + webStore.secret;
  wsConn = new WS(urlTraffic, null, onConn);
  aliveTest(wsConn, () => {
    wsConn?.close();
    wsConn = null;
    wsConn = new WS(urlTraffic, null, onConn);
  })

  connectLog(false);

  api.getRuleNum().then((res) => {
    menuStore.setRuleNum(res);
  });
});

</script>

<template>
  <div class="nav">
    <div
        :class="menuStore.menu == 'Rule' ? 'nav-btn nav-btn-select' : 'nav-btn'"
        @click="changeMenu('Rule', router)"
    >
      <el-text class="nav-text">
        <el-icon>
          <icon-mdi-source-branch/>
        </el-icon>
        <span class="nav-info"
        >{{ $t("sec-nav.rule") }} · {{ menuStore.ruleNum }}</span
        >
      </el-text>
    </div>

    <div
        :class="
        menuStore.menu == 'Connection' ? 'nav-btn nav-btn-select' : 'nav-btn'
      "
        @click="changeMenu('Connection', router)"
    >
      <el-text class="nav-text">
        <el-icon>
          <icon-mdi-lan-connect/>
        </el-icon>
        <span class="nav-info">{{ $t("sec-nav.conn") }} · {{ conn }}</span>
      </el-text>
    </div>

    <div
        :class="menuStore.menu == 'Log' ? 'nav-btn nav-btn-select' : 'nav-btn'"
        @click="changeMenu('Log', router)"
    >
      <el-text class="nav-text">
        <el-icon>
          <icon-mdi-text-box-outline/>
        </el-icon>
        <span class="nav-info">{{ $t("sec-nav.log") }}</span>
      </el-text>
    </div>
  </div>
</template>

<style scoped>
.nav {
  margin-top: 18px;
  margin-left: 22px;
  width: 185px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.nav-btn {
  display: flex;
  align-items: center;
  padding: 10px 14px;
  border-radius: 999px;
  cursor: pointer;
  background-color: var(--left-nav-btn-bg);
  box-shadow: var(--left-nav-shadow);
  transition: background-color 0.2s ease, box-shadow 0.2s ease;
}

.nav-btn:hover {
  background-color: var(--left-nav-btn-hover-bg);
  box-shadow: var(--left-nav-hover-shadow);
}

.nav-btn-select,
.nav-btn-select:hover {
  background-color: var(--left-item-selected-bg);
  box-shadow: var(--left-nav-hover-shadow);
}

.nav-text {
  color: var(--text-color);
  font-size: 18px;
  display: flex;
  align-items: center;
}

.nav-info {
  font-size: 14px;
  margin-left: 12px;
}
</style>
