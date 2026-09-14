<script setup lang="ts">

import createApi from "@/api";
import {pError, pSuccess} from "@/util/pLoad";
import {useI18n} from "vue-i18n";

// i18n
const {t} = useI18n();

// 获取当前 Vue 实例的 proxy 对象 和 api
const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);


const bypass = ref('')

onMounted(async () => {
  const ignore: string[] = await api.getIgnore()
  bypass.value = ignore.join("\n")
})

async function savaIgnore() {
  let value = bypass.value.trim();
  if (value === '') {
    return
  }

  const ignores = value.split("\n");

  try {
    await api.updateIgnore(ignores)
    pSuccess(t('rule.success'))
  } catch (e) {
    if (e['message']) {
      pError(e['message'])
    }
  }
}


</script>

<template>
  <div class="ignore">
    <el-space class="op">
      <button class="px-btn" @click="savaIgnore">{{ $t('save') }}</button>
      <el-divider direction="vertical" border-style="dashed"/>
      <el-text class="st">{{ $t('rule.ignore.tip') }}</el-text>
      <el-tooltip
          content="Домены из этого списка обходят системный прокси и подключаются напрямую (DIRECT). Указывайте по одному домену в строке, например: localhost, *.local, intranet.company"
          placement="top"
          :show-after="300"
      >
        <el-icon class="info-icon" size="16">
          <icon-tabler-info-circle/>
        </el-icon>
      </el-tooltip>
    </el-space>
    <div class="content">
      <textarea
          v-model="bypass"
          class="custom-textarea"
          :placeholder="$t('rule.ignore.place')"
      ></textarea>
    </div>
  </div>
</template>

<style scoped>
:deep(.bottom) {
  padding-bottom: 0;
  overflow-y: hidden;
  display: flex;
  flex-direction: column;
}

.ignore {
  width: 100%;
  margin-left: 0;
  margin-top: 5px;
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.op {
  margin-top: 6px;
}

.st {
  color: var(--text-color);
}

.info-icon {
  color: var(--text-color);
  opacity: 0.8;
  cursor: help;
}

.info-icon:hover {
  opacity: 1;
}

.content {
  /* Тот же зазор тулбар→контент, что у карточек и стола провайдеров (var(--px-space-5), 20px) — здесь раньше был свой 25px. */
  margin-top: var(--px-space-5);
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.custom-textarea {
  background-color: transparent;
  /* Тот же край, что у стола правил в "Now" и редактора шаблонов в "Group":
     1px var(--sub-card-border), а не собственная рамка 2px var(--text-color). */
  border: 1px solid var(--sub-card-border);
  color: var(--text-color);
  padding: var(--px-space-2) var(--px-space-2) var(--px-space-2) var(--px-space-4);
  border-radius: var(--px-r-lg);
  font-size: var(--px-fs-lead);
  resize: none;
  outline: none;
  width: 100%;
  box-sizing: border-box;
  flex: 1;
  min-height: 0;
}

.custom-textarea::placeholder {
  /* Было rgba(255,255,255,.6) буквально — на светлой подложке (светлый фон,
     тёмный текст) белый плейсхолдер не виден. --placeholder-color уже
     переключается с темой сам. */
  color: var(--placeholder-color);
}

/* :focus-visible уже стилизован глобально (global.css); свой :focus здесь
   красил тенью var(--right-box-shadow) — ещё один независимый источник тени
   ради одного поля, вместо общего кольца фокуса. */
</style>
