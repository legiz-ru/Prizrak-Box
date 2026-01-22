<script setup lang="ts">
import { ref, computed, onMounted, toRaw } from 'vue';
import { useI18n } from 'vue-i18n';
import { Events, Browser } from '@/runtime';
import createApi from '@/api';
import { pLoad, pSuccess, pError } from '@/util/pLoad';
import ProfileToolbar from './ProfileToolbar.vue';
import ProfileStats from './ProfileStats.vue';
import AnnounceText from './AnnounceText.vue';
import MyIp from './MyIp.vue';

const { proxy } = getCurrentInstance()!;
const api = createApi(proxy);
const { t } = useI18n();

interface Props {
  profiles: any[];
}

const props = defineProps<Props>();

// Определение активного профиля
const activeProfile = computed(() => {
  if (!props.profiles || props.profiles.length === 0) {
    return null;
  }

  // Приоритет: primary > selected > первый в списке
  const primary = props.profiles.find(p => p.primary);
  if (primary) return primary;

  const selected = props.profiles.find(p => p.selected);
  if (selected) return selected;

  return props.profiles[0];
});

// Обновление профиля
async function refreshProfile() {
  if (!activeProfile.value) return;

  await pLoad(t('profiles.refresh.ing'), async () => {
    try {
      const refreshed = await api.refreshProfile(activeProfile.value);
      Object.assign(activeProfile.value, refreshed);

      // Получаем обновленный список профилей для синхронизации
      const fullList = await api.getProfileList();

      // Используем toRaw для избежания ошибки клонирования
      Events.Emit({
        name: "profiles",
        data: toRaw(fullList)
      });

      // Также отправляем событие внутри Vue для немедленного обновления
      window.dispatchEvent(new CustomEvent('vue-profiles-updated', {
        detail: { profiles: toRaw(fullList) }
      }));

      pSuccess(t('profiles.refresh.success'));
    } catch (e) {
      if (e['message']) {
        pError(e['message']);
      }
    }
  });
}

// Открыть announce URL
function openAnnounceUrl() {
  if (!activeProfile.value?.announceUrl) {
    return;
  }

  const url = activeProfile.value.announceUrl.trim();
  if (!url) {
    return;
  }

  try {
    Browser.OpenURL(url);
  } catch (error) {
    if (typeof window !== 'undefined') {
      window.open(url, '_blank', 'noopener');
    }
  }
}

// Проверка наличия значения
function hasValue(value: any) {
  return value !== undefined && value !== null && value !== '';
}
</script>

<template>
  <div class="active-profile-container">
    <div class="home-cards">
      <div v-if="activeProfile" class="profile-card">
        <ProfileToolbar
          :profile="activeProfile"
          :embedded="true"
          @refresh="refreshProfile"
        />

        <ProfileStats
          :profile="activeProfile"
          :embedded="true"
        />

        <!-- Announce -->
        <div
          v-if="hasValue(activeProfile?.announce)"
          class="announce-container"
          :class="{ 'announce-clickable': hasValue(activeProfile?.announceUrl) }"
          @click="hasValue(activeProfile?.announceUrl) && openAnnounceUrl()"
        >
          <AnnounceText
            :text="activeProfile.announce"
            :url="activeProfile.announceUrl"
            :clickable="hasValue(activeProfile?.announceUrl)"
          />
        </div>
      </div>

      <!-- Нижняя панель IP и Система -->
      <MyIp class="home-ip" />
    </div>
  </div>
</template>

<style scoped>
.active-profile-container {
  width: 100%;
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding-top: 15px;
  padding-bottom: 0;
  position: relative;
  gap: 16px;
  --home-card-width: 95%;
  box-sizing: border-box;
  overflow-x: hidden;
}

.home-cards {
  margin-left: 10px;
  margin-right: 10px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  flex: 1 1 auto;
  min-height: 0;
}

.profile-card {
  width: 100%;
  padding: 12px 0;
  border-radius: 8px;
  background: var(--sub-card-bg);
  border: 1px solid var(--sub-card-border);
  box-shadow: var(--right-box-shadow);
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.announce-container {
  width: 100%;
  padding: 12px 30px;
  font-size: 14px;
  color: var(--text-color);
  text-align: center;
  word-wrap: break-word;
}

.announce-clickable {
  cursor: pointer;
}

.announce-clickable:hover {
  opacity: 0.8;
}
</style>
