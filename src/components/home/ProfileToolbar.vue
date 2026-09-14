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
import AddProfileMenu from './AddProfileMenu.vue';

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
const addForm = ref({
  content: ''
});
const isAdding = ref(false);

// Название профиля с поддержкой emoji
const profileDisplayTitle = computed(() => {
  const flagEmojiRegex = /([\u{1F1E6}-\u{1F1FF}]{2}|\u{1F3F3}|\u{1F3F4}|\u{1F6A9})/u;

  const containsFlagEmoji = (value: any) => {
    if (typeof value !== 'string') {
      return false;
    }
    return flagEmojiRegex.test(value);
  };

  const title = typeof props.profile?.title === 'string' ? props.profile.title.trim() : '';
  const headerTitle = typeof props.profile?.headerTitle === 'string' ? props.profile.headerTitle.trim() : '';

  if (title) {
    if (!headerTitle) {
      return title;
    }
    if (containsFlagEmoji(title) || !containsFlagEmoji(headerTitle)) {
      return title;
    }
  }

  return headerTitle || title || '';
});

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
  addForm.value.content = '';
  addFormVisible.value = true;
}

// Импорт из буфера обмена
function handlePaste() {
  const clipboardText = Clipboard.Text();
  if (!clipboardText || !clipboardText.trim()) {
    pWarning(t('onboarding.active-profile.clipboard-empty'));
    return;
  }
  addForm.value.content = clipboardText;
  addFormVisible.value = true;
}

// Открыть выбор файла
function openFile() {
  webStore.dnd = true;
}

// Добавить профиль
async function addProfile() {
  if (!addForm.value.content || !addForm.value.content.trim()) {
    return;
  }

  isAdding.value = true;
  const p = new Profile();
  p.content = addForm.value.content;

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
    addForm.value.content = '';
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
    <div class="toolbar-content">
      <div class="toolbar-section toolbar-left">
        <!-- Иконка продления подписки -->
        <n-tooltip v-if="hasValue(profile?.renewUrl)" trigger="hover" placement="top">
          <template #trigger>
            <n-icon class="toolbar-icon" @click="goRenew" size="20">
              <icon-mdi-credit-card-outline />
            </n-icon>
          </template>
          {{ t('profiles.renew') }}
        </n-tooltip>

        <!-- Иконка домашней страницы -->
        <n-tooltip v-if="hasValue(profile?.home)" trigger="hover" placement="top">
          <template #trigger>
            <n-icon class="toolbar-icon" @click="goHome" size="20">
              <icon-mdi-home-import-outline />
            </n-icon>
          </template>
          {{ t('profiles.home') }}
        </n-tooltip>

        <!-- Иконка поддержки -->
        <n-tooltip v-if="hasValue(profile?.support)" trigger="hover" placement="top">
          <template #trigger>
            <n-icon class="toolbar-icon" @click="goSupport" size="20">
              <icon-mdi-face-agent />
            </n-icon>
          </template>
          {{ t('profiles.support') }}
        </n-tooltip>
      </div>

      <div class="toolbar-section toolbar-center">
        <!-- Текст "Текущий профиль" -->
        <span class="current-profile-label">{{ t('onboarding.active-profile.current-profile') }}</span>

        <!-- Название профиля -->
        <span class="profile-name" :title="profileDisplayTitle">{{ profileDisplayTitle }}</span>
      </div>

      <div class="toolbar-section toolbar-right">
        <!-- Переключить профили -->
        <n-tooltip trigger="hover" placement="top">
          <template #trigger>
            <n-icon class="toolbar-icon" @click="switchProfiles" size="20">
              <icon-mdi-swap-horizontal />
            </n-icon>
          </template>
          {{ t('onboarding.active-profile.switch-profiles') }}
        </n-tooltip>

        <!-- Обновить профиль -->
        <n-tooltip trigger="hover" placement="top">
          <template #trigger>
            <n-icon
              class="toolbar-icon"
              :class="{ 'rotating': isRefreshing }"
              @click="refreshProfile"
              size="20"
            >
              <icon-mdi-refresh />
            </n-icon>
          </template>
          {{ t('onboarding.active-profile.refresh-profile') }}
        </n-tooltip>

        <!-- Добавить профиль -->
        <n-tooltip trigger="hover" placement="top">
          <template #trigger>
            <AddProfileMenu @add="openAddProfileDialog" @paste="handlePaste" @file="openFile">
              <n-icon class="toolbar-icon" size="20">
                <icon-mdi-plus-thick />
              </n-icon>
            </AddProfileMenu>
          </template>
          {{ t('profiles.add') }}
        </n-tooltip>
      </div>
    </div>
  </div>

  <!-- Модальное окно добавления профиля -->
  <n-modal
    v-model:show="addFormVisible"
    preset="card"
    :title="t('profiles.add')"
    :bordered="false"
    style="width: 520px"
  >
    <n-form :model="addForm">
      <n-form-item :show-label="false">
        <n-input
          :rows="3"
          type="textarea"
          autocapitalize="off"
          autocomplete="off"
          spellcheck="false"
          :placeholder="t('profiles.placeholder')"
          v-model:value="addForm.content"
        />
      </n-form-item>
    </n-form>
    <template #footer>
      <div class="dialog-footer">
        <n-button @click="addFormVisible = false">
          {{ t('cancel') }}
        </n-button>
        <n-button
          type="primary"
          @click="addProfile"
          :loading="isAdding"
        >
          {{ t('confirm') }}
        </n-button>
      </div>
    </template>
  </n-modal>
</template>

<style scoped>
.profile-toolbar {
  width: 100%;
  padding: 0 30px;
  box-sizing: border-box;
}

.toolbar-content {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 16px;
}

.toolbar-section {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  flex: 0 1 auto;
  min-width: 0;
}

.toolbar-center {
  text-align: center;
}

.toolbar-icon {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  cursor: pointer;
  transition: background 0.2s ease;
  color: var(--text-color);
}

.toolbar-icon:hover {
  background: var(--hr-color);
}

.toolbar-icon.rotating {
  animation: rotate 1s linear infinite;
}

@keyframes rotate {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.current-profile-label {
  font-size: 12px;
  color: var(--text-color);
  opacity: 0.6;
  margin-left: 2px;
}

.profile-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-color);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
  flex-shrink: 1;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
