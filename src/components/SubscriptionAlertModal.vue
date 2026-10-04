<script setup lang="ts">
// Shown when the user clicks a native "subscription expiring/expired/traffic
// used" notification (see util/subscriptionAlerts.ts). Both shells funnel the
// click into the same window CustomEvent — see that file's comment for the
// Electron/Wails split — so this component only ever listens to one thing.
import {ref, onMounted, onBeforeUnmount} from 'vue';
import {useI18n} from 'vue-i18n';
import {Browser, Events} from '@/runtime';
import {useWebStore} from '@/store/webStore';
import createApi from '@/api';
import {
  SUBSCRIPTION_ALERT_CLICKED_EVENT,
  notifySubscriptionAlertClicked,
  describeAlert,
  type SubscriptionAlert,
  type SubscriptionAlertClickDetail,
} from '@/util/subscriptionAlerts';

const {t, locale} = useI18n();
const webStore = useWebStore();
const {proxy} = getCurrentInstance()!;
const api = createApi(proxy);

const visible = ref(false);
const profileName = ref('');
const message = ref('');
const expireDetail = ref('');
const renewUrl = ref('');
const logo = ref('');

async function resolveProfile(profileId: string): Promise<any | null> {
  const cached = webStore.profileList?.find((p: any) => p.id === profileId);
  if (cached) {
    return cached;
  }
  // The cache can be stale/empty right after launch — fall back to a fresh
  // fetch rather than showing an empty dialog.
  try {
    const list = await api.getProfileList();
    return list?.find((p: any) => p.id === profileId) ?? null;
  } catch {
    return null;
  }
}

async function handleClick(detail: SubscriptionAlertClickDetail) {
  if (!detail?.profileId) {
    return;
  }

  const profile = await resolveProfile(detail.profileId);
  const alert: SubscriptionAlert = {kind: detail.kind, days: detail.days, percent: detail.percent};

  profileName.value = profile?.title || profile?.headerTitle || '';
  // Computed from the profile as it is now: the notification may have sat
  // unopened for a day, and "4 days" must not outlive the 4 days.
  const texts = describeAlert(t, alert, profile, locale.value);
  message.value = texts.message;
  expireDetail.value = texts.detail ?? '';
  renewUrl.value = profile?.renewUrl || '';
  logo.value = typeof profile?.logo === 'string' ? profile.logo.trim() : '';
  visible.value = true;
}

function onWindowEvent(e: Event) {
  const detail = (e as CustomEvent<SubscriptionAlertClickDetail>).detail;
  if (detail) {
    void handleClick(detail);
  }
}

// Wails: the click round-trips through the Go NotificationService
// (OnNotificationResponse in src-wails/main.go -> "px:be:subscriptionAlertClicked"),
// re-dispatched here as the same window CustomEvent Electron's direct closure
// path already uses. Harmless no-op subscription under Electron — nothing
// ever emits this channel there.
function onBackendEvent(data: any) {
  if (data) {
    notifySubscriptionAlertClicked(data as SubscriptionAlertClickDetail);
  }
}

onMounted(() => {
  window.addEventListener(SUBSCRIPTION_ALERT_CLICKED_EVENT, onWindowEvent);
  Events.On('subscriptionAlertClicked', onBackendEvent);
});

onBeforeUnmount(() => {
  window.removeEventListener(SUBSCRIPTION_ALERT_CLICKED_EVENT, onWindowEvent);
  Events.Off('subscriptionAlertClicked', onBackendEvent);
});

function goRenew() {
  if (renewUrl.value) {
    try {
      Browser.OpenURL(renewUrl.value);
    } catch {
      window.open(renewUrl.value, '_blank', 'noopener');
    }
  }
  visible.value = false;
}
</script>

<template>
  <UiNotice v-model="visible" :title="profileName" :width="380" tone="warning">
    <template #top>
      <img v-if="logo" :src="logo" alt="" class="sub-alert-logo">
    </template>
    <div class="sub-alert-pill">
      <icon-tabler-bell width="17" height="17"/>
      <span>{{ message }}</span>
    </div>
    <div v-if="expireDetail" class="sub-alert-detail">{{ expireDetail }}</div>
    <template #actions>
      <button type="button" class="px-btn" @click="visible = false">{{ t('close') }}</button>
      <button v-if="renewUrl" type="button" class="px-btn px-btn--primary" @click="goRenew">{{ t('profiles.renew') }}</button>
    </template>
  </UiNotice>
</template>

<style scoped>
.sub-alert-logo {
  width: 30px;
  height: 30px;
  border-radius: 8px;
  object-fit: contain;
  margin-bottom: 4px;
}

.sub-alert-pill {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
  padding: 10px 14px;
  border-radius: 12px;
  background: color-mix(in srgb, var(--warning) 14%, transparent);
  color: var(--warning);
  font-size: 14px;
  font-weight: 600;
}

.sub-alert-detail {
  margin-top: 8px;
  text-align: center;
  font-size: 13px;
  color: var(--text-2);
}

.sub-alert-pill svg {
  flex-shrink: 0;
}
</style>
