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
import { useHwidStatusStore } from '@/store/hwidStatusStore';
import { shouldShowRenewButton } from '@/util/subscriptionAlerts';

const { proxy } = getCurrentInstance()!;
const api = createApi(proxy);
const { t } = useI18n();
const hwidStatusStore = useHwidStatusStore();

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

      if (refreshed?.hwidNotSupported) {
        hwidStatusStore.showNotSupported();
      } else if (refreshed?.hwidMaxDevicesReached) {
        const supportUrl = typeof refreshed.support === 'string' ? refreshed.support : '';
        hwidStatusStore.showMaxDevicesReached(supportUrl);
      }
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

// Открыть страницу продления подписки
function goRenew() {
  const url = activeProfile.value?.renewUrl;
  if (!hasValue(url)) {
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

// Кнопка "Продлить подписку" — независима от настройки "Subscription
// reminders" (это элемент интерфейса, не пуш) и от того, что уже было
// показано пуш-уведомлениями: просто отражает текущее состояние подписки.
const showRenewButton = computed(() => shouldShowRenewButton(activeProfile.value));
</script>

<template>
  <div v-if="activeProfile" class="px-card profile-card">
    <ProfileToolbar
      :profile="activeProfile"
      @refresh="refreshProfile"
    />

    <div class="px-divider"></div>

    <ProfileStats :profile="activeProfile" />

    <button
      v-if="hasValue(activeProfile?.announce)"
      type="button"
      class="announce"
      :class="{ 'is-clickable': hasValue(activeProfile?.announceUrl) }"
      :disabled="!hasValue(activeProfile?.announceUrl)"
      @click="openAnnounceUrl()"
    >
      <AnnounceText
        :text="activeProfile.announce"
        :url="activeProfile.announceUrl"
        :clickable="hasValue(activeProfile?.announceUrl)"
      />
    </button>

    <button v-if="showRenewButton" type="button" class="renew-button" @click="goRenew">
      <icon-tabler-credit-card width="15" height="15"/>
      <span>{{ t('profiles.renew') }}</span>
    </button>
  </div>

  <MyIp />
</template>

<style scoped>
.profile-card {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.announce {
  border: none;
  background: transparent;
  padding: 0 6px;
  font-size: 13px;
  color: var(--text-2);
  text-align: center;
  cursor: default;
  overflow-wrap: anywhere;
}

.announce.is-clickable {
  cursor: pointer;
}

.announce.is-clickable:hover {
  color: var(--text);
}

/* Readable over any background: same text colour as the card values, on a
   light accent tint. */
.renew-button {
  width: 100%;
  border: none;
  border-radius: 999px;
  background: color-mix(in srgb, var(--accent) 16%, transparent);
  color: var(--text);
  font-size: 13px;
  font-weight: 700;
  padding: 10px 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  cursor: pointer;
}

.renew-button:hover {
  background: color-mix(in srgb, var(--accent) 26%, transparent);
}
</style>
