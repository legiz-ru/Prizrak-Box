<script setup lang="ts">
import { ref, computed, toRaw } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';
import { Browser, Clipboard, Events } from '@/runtime';
import { pWarning, pError, pSuccess } from '@/util/pLoad';
import { useWebStore } from '@/store/webStore';
import { Profile } from '@/types/profile';
import createApi from '@/api';
import { useHwidStatusStore } from '@/store/hwidStatusStore';
import { parseHwidFromError } from '@/api/profiles';
import { profileDisplayTitle as displayTitleOf } from '@/util/profileView';
import AddProfileDialog from '@/components/profile/AddProfileDialog.vue';
import UiDropdown from '@/components/ui/UiDropdown.vue';

const { t } = useI18n();
const router = useRouter();
const webStore = useWebStore();
const { proxy } = getCurrentInstance()!;
const api = createApi(proxy);
const hwidStatusStore = useHwidStatusStore();

interface Props {
  profile: any;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  (e: 'refresh'): void
}>();

const isRefreshing = ref(false);
const addFormVisible = ref(false);
const addInitial = ref('');
const isAdding = ref(false);

// Название профиля с поддержкой emoji
const profileDisplayTitle = computed(() => displayTitleOf(props.profile));

// Проверка наличия значений
function hasValue(value: any) {
  return value !== undefined && value !== null && value !== '';
}

// Открыть страницу продления подписки
function goRenew() {
  if (hasValue(props.profile?.renewUrl)) {
    Browser.OpenURL(props.profile.renewUrl);
  }
}

// Открыть главную страницу профиля
function goHome() {
  if (hasValue(props.profile?.home)) {
    Browser.OpenURL(props.profile.home);
  }
}

// Открыть страницу поддержки
function goSupport() {
  if (hasValue(props.profile?.support)) {
    Browser.OpenURL(props.profile.support);
  }
}

// Переключить на страницу профилей
function switchProfiles() {
  router.push('/profiles');
}

// Обновить профиль
function refreshProfile() {
  isRefreshing.value = true;
  emit('refresh');
  // Анимация остановится автоматически через CSS или при следующем обновлении
  setTimeout(() => {
    isRefreshing.value = false;
  }, 1000);
}

// Открыть диалог добавления профиля
function openAddProfileDialog() {
  addInitial.value = '';
  addFormVisible.value = true;
}

// Импорт из буфера обмена
function handlePaste() {
  const clipboardText = Clipboard.Text();
  if (!clipboardText || !clipboardText.trim()) {
    pWarning(t('onboarding.active-profile.clipboard-empty'));
    return;
  }
  addInitial.value = clipboardText;
  addFormVisible.value = true;
}

// Открыть выбор файла
function openFile() {
  webStore.dnd = true;
}

// Добавить профиль
async function addProfile(form: { content: string; ageSecretKey: string }) {
  if (!form.content || !form.content.trim()) {
    return;
  }

  isAdding.value = true;
  const p = new Profile();
  p.content = form.content;
  if (form.ageSecretKey) {
    p.ageSecretKey = form.ageSecretKey;
  }

  try {
    const newProfiles = await api.addProfileFromInput(p);

    // Если профили добавлены, активируем первый из них
    if (newProfiles && newProfiles.length > 0) {
      const firstProfile = newProfiles[0];

      // Переключаемся на новый профиль (эксклюзивно)
      await api.switchProfile({
        id: firstProfile.id,
        selected: true,
        exclusive: true,
      });

      // Ждём, пока прокси запустится
      await api.waitRunning();

      Events.Emit({
        name: "profileChanged",
        data: {
          profile: toRaw(firstProfile),
          exclusive: true,
        }
      });
      window.dispatchEvent(new CustomEvent('profile-changed'));
    }

    // Получаем обновленный список профилей
    const fullList = await api.getProfileList();

    // Отправляем событие обновления профилей (через IPC в Electron)
    // Используем toRaw для избежания ошибки клонирования
    Events.Emit({
      name: "profiles",
      data: toRaw(fullList)
    });

    // Также отправляем событие внутри Vue для немедленного обновления
    window.dispatchEvent(new CustomEvent('vue-profiles-updated', {
      detail: { profiles: toRaw(fullList) }
    }));

    pSuccess(t('drag.success'));
    addFormVisible.value = false;
  } catch (e) {
    const hwid = parseHwidFromError(e);
    if (hwid) {
      if (hwid.hwidNotSupported) {
        hwidStatusStore.showNotSupported();
      } else if (hwid.hwidMaxDevicesReached) {
        hwidStatusStore.showMaxDevicesReached(hwid.supportUrl);
      }
    } else if (e['message']) {
      pError(e['message']);
    }
  } finally {
    isAdding.value = false;
  }
}

</script>

<template>
  <div class="profile-toolbar">
    <div class="toolbar-side">
      <UiIconButton v-if="hasValue(profile?.renewUrl)" :size="28" :label="t('profiles.renew')" @click="goRenew">
        <icon-tabler-credit-card width="15" height="15"/>
      </UiIconButton>
      <UiIconButton v-if="hasValue(profile?.home)" :size="28" :label="t('profiles.home')" @click="goHome">
        <icon-tabler-home-shield width="15" height="15"/>
      </UiIconButton>
      <UiIconButton v-if="hasValue(profile?.support)" :size="28" :label="t('profiles.support')" @click="goSupport">
        <icon-tabler-headphones width="15" height="15"/>
      </UiIconButton>
    </div>

    <span class="profile-name ellipsis" v-tip="profileDisplayTitle">{{ profileDisplayTitle }}</span>

    <div class="toolbar-side toolbar-side--right">
      <UiIconButton :size="28" :label="t('onboarding.active-profile.switch-profiles')" @click="switchProfiles">
        <icon-tabler-arrows-left-right width="15" height="15"/>
      </UiIconButton>
      <UiIconButton :size="28" :label="t('onboarding.active-profile.refresh-profile')" :loading="isRefreshing" @click="refreshProfile">
        <UiSpinner v-if="isRefreshing" :size="13"/>
        <icon-tabler-refresh v-else width="15" height="15"/>
      </UiIconButton>
      <UiDropdown align="right" role="menu" :min-width="240">
        <template #trigger="{ toggle, attrs }">
          <UiIconButton v-bind="attrs" :size="28" :label="t('profiles.add')" @click="toggle">
            <icon-tabler-plus width="15" height="15"/>
          </UiIconButton>
        </template>
        <template #default="{ close }">
          <button type="button" role="menuitem" data-dd-item class="px-dd-item add-item" @click="close(); openAddProfileDialog()">
            <icon-tabler-pencil width="16" height="16"/>{{ t('profiles.add') }}
          </button>
          <button type="button" role="menuitem" data-dd-item class="px-dd-item add-item" @click="close(); handlePaste()">
            <icon-tabler-clipboard width="16" height="16"/>{{ t('profiles.paste') }}
          </button>
          <button type="button" role="menuitem" data-dd-item class="px-dd-item add-item" @click="close(); openFile()">
            <icon-tabler-folder-open width="16" height="16"/>{{ t('profiles.open') }}
          </button>
        </template>
      </UiDropdown>
    </div>
  </div>

  <AddProfileDialog v-model="addFormVisible"
                    :initial-content="addInitial"
                    :loading="isAdding"
                    @submit="addProfile"/>
</template>

<style scoped>
.profile-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

/* Both sides reserve room for three buttons so the name stays centred. */
.toolbar-side {
  display: flex;
  gap: 4px;
  flex: 0 0 92px;
}

.toolbar-side--right {
  justify-content: flex-end;
}

.profile-name {
  font-size: 14px;
  font-weight: 700;
  text-align: center;
  flex: 1;
}

.add-item {
  font-weight: 600;
  padding: 9px 12px;
  gap: 10px;
}

.add-item svg {
  color: var(--text-2);
  flex-shrink: 0;
}
</style>
