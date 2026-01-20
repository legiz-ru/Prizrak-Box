<script setup lang="ts">
import { ref, computed, toRaw, withDefaults } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';
import { Browser, Clipboard, Events } from '@/runtime';
import { pWarning, pError, pSuccess } from '@/util/pLoad';
import { useWebStore } from '@/store/webStore';
import { Profile } from '@/types/profile';
import createApi from '@/api';

const { t } = useI18n();
const router = useRouter();
const webStore = useWebStore();
const { proxy } = getCurrentInstance()!;
const api = createApi(proxy);

interface Props {
  profile: any;
  embedded?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  embedded: false,
});

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
    if (e['message']) {
      pError(e['message']);
    }
  } finally {
    isAdding.value = false;
  }
}

// Обработка команды из dropdown
function handleDropdownCommand(command: string) {
  if (command === 'add') {
    openAddProfileDialog();
  } else if (command === 'paste') {
    handlePaste();
  } else if (command === 'file') {
    openFile();
  }
}
</script>

<template>
  <div class="profile-toolbar" :class="{ 'profile-toolbar--embedded': embedded }">
    <div class="toolbar-content">
      <div class="toolbar-section toolbar-left">
        <!-- Иконка домашней страницы -->
        <el-tooltip
          v-if="hasValue(profile?.home)"
          :content="t('profiles.home')"
          placement="top"
        >
          <el-icon class="toolbar-icon" @click="goHome" size="20">
            <icon-mdi-home-import-outline />
          </el-icon>
        </el-tooltip>

        <!-- Иконка поддержки -->
        <el-tooltip
          v-if="hasValue(profile?.support)"
          :content="t('profiles.support')"
          placement="top"
        >
          <el-icon class="toolbar-icon" @click="goSupport" size="20">
            <icon-mdi-face-agent />
          </el-icon>
        </el-tooltip>
      </div>

      <div class="toolbar-section toolbar-center">
        <!-- Текст "Текущий профиль" -->
        <span class="current-profile-label">{{ t('onboarding.active-profile.current-profile') }}</span>

        <!-- Название профиля -->
        <span class="profile-name" :title="profileDisplayTitle">{{ profileDisplayTitle }}</span>
      </div>

      <div class="toolbar-section toolbar-right">
        <!-- Переключить профили -->
        <el-tooltip
          :content="t('onboarding.active-profile.switch-profiles')"
          placement="top"
        >
          <el-icon class="toolbar-icon" @click="switchProfiles" size="20">
            <icon-mdi-swap-horizontal />
          </el-icon>
        </el-tooltip>

        <!-- Обновить профиль -->
        <el-tooltip
          :content="t('onboarding.active-profile.refresh-profile')"
          placement="top"
        >
          <el-icon
            class="toolbar-icon"
            :class="{ 'rotating': isRefreshing }"
            @click="refreshProfile"
            size="20"
          >
            <icon-mdi-refresh />
          </el-icon>
        </el-tooltip>

        <!-- Добавить профиль -->
        <el-tooltip
          :content="t('profiles.add')"
          placement="top"
        >
          <el-dropdown trigger="click" @command="handleDropdownCommand">
            <el-icon class="toolbar-icon" size="20">
              <icon-mdi-plus-thick />
            </el-icon>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="add">
                  <el-icon><icon-mdi-pencil /></el-icon>
                  {{ t('profiles.add') }}
                </el-dropdown-item>
                <el-dropdown-item command="paste">
                  <el-icon><icon-mdi-content-paste /></el-icon>
                  {{ t('profiles.paste') }}
                </el-dropdown-item>
                <el-dropdown-item command="file">
                  <el-icon><icon-mdi-folder-open /></el-icon>
                  {{ t('profiles.open') }}
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </el-tooltip>
      </div>
    </div>
  </div>

  <!-- Модальное окно добавления профиля -->
  <el-dialog
    v-model="addFormVisible"
    :title="t('profiles.add')"
    width="520"
    draggable
    center
  >
    <el-form :model="addForm">
      <el-form-item>
        <el-input
          :rows="3"
          type="textarea"
          autocapitalize="off"
          autocomplete="off"
          spellcheck="false"
          :placeholder="t('profiles.placeholder')"
          v-model="addForm.content"
        />
      </el-form-item>
    </el-form>
    <template #footer>
      <div class="dialog-footer">
        <el-button @click="addFormVisible = false">
          {{ t('cancel') }}
        </el-button>
        <el-button
          type="primary"
          @click="addProfile"
          :loading="isAdding"
        >
          {{ t('confirm') }}
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<style scoped>
.profile-toolbar {
  width: 95%;
  margin: 0 auto;
  padding: 12px 20px;
  box-sizing: border-box;
  border-radius: 8px;
  background: var(--sub-card-bg);
  box-shadow: var(--right-box-shadow);
}

.profile-toolbar--embedded {
  width: 100%;
  margin: 0;
  padding: 0 30px;
  box-sizing: border-box;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
}

.profile-toolbar--embedded .toolbar-content {
  justify-content: center;
  gap: 16px;
}

.profile-toolbar--embedded .toolbar-section {
  justify-content: center;
  gap: 4px;
  flex: 0 1 auto;
}

.profile-toolbar--embedded .toolbar-left {
  justify-content: center;
}

.profile-toolbar--embedded .toolbar-right {
  justify-content: center;
}

.profile-toolbar--embedded .toolbar-center {
  flex: 0 1 auto;
  max-width: none;
  min-width: 0;
  text-align: center;
}

.profile-toolbar--embedded .toolbar-icon {
  width: 28px;
  height: 28px;
}

.toolbar-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 20px;
}

.toolbar-section {
  display: flex;
  align-items: center;
  gap: 6px;
}

.toolbar-left {
  flex: 0 0 auto;
  justify-content: flex-start;
  min-width: 0;
}

.toolbar-center {
  flex: 1 1 auto;
  justify-content: center;
  max-width: 60%;
  min-width: 0;
}

.toolbar-right {
  flex: 0 0 auto;
  justify-content: flex-end;
  min-width: 0;
}

/* Убираем переопределение для embedded, оставляем базовый flex */

.toolbar-icon {
  width: 32px;
  height: 32px;
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
  font-size: 14px;
  color: var(--text-color);
  opacity: 0.6;
  margin-left: 8px;
}

.profile-toolbar--embedded .current-profile-label {
  font-size: 12px;
  margin-left: 2px;
}

.profile-name {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-color);
  max-width: 300px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-variant-emoji: emoji;
  font-family: 'Twemoji', "Nunito", 'Microsoft YaHei', '微软雅黑', 'Helvetica Neue', Helvetica, 'PingFang SC', 'Hiragino Sans GB',
  Arial, -apple-system, BlinkMacSystemFont, "Segoe UI", "Roboto",
  "Oxygen", "Ubuntu", "Cantarell", "Fira Sans", "Droid Sans", "Helvetica Neue",
  sans-serif;
  min-width: 0;
}

.profile-toolbar--embedded .profile-name {
  font-size: 15px;
  max-width: none;
  flex-shrink: 1;
}
</style>
