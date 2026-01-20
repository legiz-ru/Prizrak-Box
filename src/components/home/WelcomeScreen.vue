<script setup lang="ts">
import { ref, toRaw, onMounted, onBeforeUnmount } from 'vue';
import { useI18n } from 'vue-i18n';
import { Events, Clipboard } from '@/runtime';
import { Profile } from '@/types/profile';
import createApi from '@/api';
import { pError, pSuccess } from '@/util/pLoad';
import { useWebStore } from '@/store/webStore';

const { t } = useI18n();
const { proxy } = getCurrentInstance()!;
const api = createApi(proxy);
const webStore = useWebStore();

const addFormVisible = ref(false);
const addForm = ref({
  content: '',
});
const isAdding = ref(false);

// Открыть модальное окно добавления профиля (ручной ввод)
function openAddProfileDialog() {
  addForm.value.content = '';
  addFormVisible.value = true;
}

// Импорт из буфера обмена
function handlePaste() {
  const text = Clipboard.Text();
  if (!text || !text.trim()) {
    pError(t('onboarding.active-profile.clipboard-empty'));
    return;
  }
  addForm.value.content = text;
  addFormVisible.value = true;
}

// Открыть file picker
function openFile() {
  webStore.dnd = true;
}

// Добавить профиль из модального окна
async function addProfile() {
  if (!addForm.value.content.trim()) {
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

      // Обновляем профиль для получения полной информации (логотип, имя и т.д.)
      if (firstProfile.type === 1) {
        try {
          await api.refreshProfile(firstProfile);
        } catch (e) {
          // Игнорируем ошибку обновления, профиль уже добавлен
          console.warn('Failed to refresh profile:', e);
        }
      }
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

// Обработка drag & drop
function handleDragOver(e: DragEvent) {
  e.preventDefault();
  e.stopPropagation();
  if (e.dataTransfer) {
    e.dataTransfer.dropEffect = 'copy';
  }
}

async function handleDrop(e: DragEvent) {
  e.preventDefault();
  e.stopPropagation();

  const files = e.dataTransfer?.files;
  if (!files || files.length === 0) {
    return;
  }

  if (files.length > 1) {
    pError(t('drag.size'));
    return;
  }

  const file = files[0];
  const reader = new FileReader();

  reader.onload = async (event) => {
    const content = event.target?.result as string;
    if (!content) {
      pError(t('drag.error'));
      return;
    }

    addForm.value.content = content;
    await addProfile();
  };

  reader.onerror = () => {
    pError(t('drag.error'));
  };

  reader.readAsText(file);
}
</script>

<template>
  <div
    class="welcome-container"
    @dragover="handleDragOver"
    @drop="handleDrop"
  >
    <div class="welcome-content">
      <h1 class="welcome-title">{{ t('onboarding.welcome.title') }}</h1>
      <p class="welcome-subtitle">{{ t('onboarding.welcome.subtitle') }}</p>

      <div class="add-profile-button-container">
        <!-- Dropdown меню -->
        <el-dropdown trigger="click" @command="(cmd) => {
          if (cmd === 'add') openAddProfileDialog();
          else if (cmd === 'paste') handlePaste();
          else if (cmd === 'file') openFile();
        }">
          <button class="add-profile-button" :aria-label="t('onboarding.welcome.add-profile')">
            <el-icon :size="40">
              <icon-mdi-plus-thick />
            </el-icon>
          </button>
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

        <div class="add-profile-label">{{ t('onboarding.welcome.add-profile') }}</div>
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
.welcome-container {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  width: 100%;
}

.welcome-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 40px 20px;
}

.welcome-title {
  font-size: 32px;
  font-weight: 600;
  margin-bottom: 10px;
  color: var(--text-color);
}

.welcome-subtitle {
  font-size: 18px;
  margin-bottom: 50px;
  color: var(--text-color);
  opacity: 0.7;
}

.add-profile-button-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 15px;
  margin-bottom: 20px;
}

.add-profile-button {
  width: 120px;
  height: 120px;
  border-radius: 50%;
  border: none;
  background: var(--left-nav-btn-bg);
  color: var(--text-color);
  cursor: pointer;
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: var(--left-nav-shadow);
}

.add-profile-button:hover {
  transform: scale(1.05);
  background: var(--left-nav-btn-active-bg);
  box-shadow: var(--left-nav-hover-shadow);
}

.add-profile-button:active {
  transform: scale(0.98);
}

.add-profile-label {
  font-size: 16px;
  color: var(--text-color);
  font-weight: 500;
}

:deep(.el-dropdown-menu__item:hover) {
  background-color: var(--left-item-selected-bg);
  color: var(--text-color);
}
</style>
