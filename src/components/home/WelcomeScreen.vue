<script setup lang="ts">
import { ref, toRaw, onMounted, onBeforeUnmount } from 'vue';
import { useI18n } from 'vue-i18n';
import { Events, Clipboard } from '@/runtime';
import { Profile } from '@/types/profile';
import createApi from '@/api';
import { pError, pSuccess } from '@/util/pLoad';
import { useWebStore } from '@/store/webStore';
import AddProfileDialog from '@/components/profile/AddProfileDialog.vue';
import UiDropdown from '@/components/ui/UiDropdown.vue';

const { t } = useI18n();
const { proxy } = getCurrentInstance()!;
const api = createApi(proxy);
const webStore = useWebStore();

const addFormVisible = ref(false);
const addInitial = ref('');
const isAdding = ref(false);

// Открыть модальное окно добавления профиля (ручной ввод)
function openAddProfileDialog() {
  addInitial.value = '';
  addFormVisible.value = true;
}

// Импорт из буфера обмена
function handlePaste() {
  const text = Clipboard.Text();
  if (!text || !text.trim()) {
    pError(t('onboarding.active-profile.clipboard-empty'));
    return;
  }
  addInitial.value = text;
  addFormVisible.value = true;
}

// Открыть file picker
function openFile() {
  webStore.dnd = true;
}

// Добавить профиль из модального окна
async function addProfile(form: { content: string; ageSecretKey?: string }) {
  if (!form.content.trim()) {
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

    let firstProfileId = null;

    // Если профили добавлены, активируем первый из них
    if (newProfiles && newProfiles.length > 0) {
      const firstProfile = newProfiles[0];
      firstProfileId = firstProfile.id;

      // Обновляем профиль для получения полной информации (логотип, имя и т.д.)
      if (firstProfile.type === 1) {
        try {
          await api.refreshProfile(firstProfile);
        } catch (e) {
          // Игнорируем ошибку обновления, профиль уже добавлен
          console.warn('Failed to refresh profile:', e);
        }
      }

      // Переключаемся на новый профиль (эксклюзивно)
      await api.switchProfile({
        id: firstProfile.id,
        selected: true,
        exclusive: true,
      });

      // Ждём, пока прокси запустится.
      // pxd-template профили могут загружаться >30с при первой активации
      // (бэкенд скачивает proxy-providers без работающего прокси).
      // Таймаут не критичен — горутина продолжает работу, прокси появятся
      // через несколько секунд после открытия вкладки Прокси.
      try {
        await api.waitRunning();
      } catch (e) {
        console.warn('[WelcomeScreen] waitRunning timeout, backend still processing:', (e as any)?.message);
      }
    }

    // Получаем обновленный список профилей ПОСЛЕ refresh
    const fullList = await api.getProfileList();

    // Находим активный профиль в списке
    let activeProfile = fullList?.find((item: any) => item?.primary)
      ?? fullList?.find((item: any) => item?.selected)
      ?? fullList?.[0];

    // Если профиль все еще без логотипа, делаем refresh еще раз для профиля из списка
    if (firstProfileId && activeProfile?.id === firstProfileId && activeProfile?.type === 1) {
      if (!activeProfile?.logo && !(activeProfile as any)?.icon) {
        try {
          const refreshed = await api.refreshProfile(activeProfile);
          Object.assign(activeProfile, refreshed);
        } catch (e) {
          console.warn('Failed to re-refresh profile:', e);
        }
      }
    }

    // Обновляем webStore.fProfile для корректного отображения в UI
    if (activeProfile) {
      webStore.fProfile = toRaw({
        ...activeProfile,
        exclusive: true,
      });
    }

    // Отправляем события ПОСЛЕ того как activeProfile точно обновлен
    if (activeProfile) {
      Events.Emit({
        name: "profileChanged",
        data: {
          profile: toRaw(activeProfile),
          exclusive: true,
        }
      });
      window.dispatchEvent(new CustomEvent('profile-changed'));
    }

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

    await addProfile({content});
  };

  reader.onerror = () => {
    pError(t('drag.error'));
  };

  reader.readAsText(file);
}
</script>

<template>
  <div
    class="welcome"
    @dragover="handleDragOver"
    @drop="handleDrop"
  >
    <div class="welcome-content">
      <h1 class="welcome-title">{{ t('onboarding.welcome.title') }}</h1>
      <p class="welcome-subtitle">{{ t('onboarding.welcome.subtitle') }}</p>

      <UiDropdown role="menu" :min-width="240" class="welcome-dd">
        <template #trigger="{ toggle, attrs }">
          <button type="button"
                  v-bind="attrs"
                  class="welcome-add"
                  :aria-label="t('onboarding.welcome.add-profile')"
                  @click="toggle">
            <icon-tabler-plus width="44" height="44" stroke-width="2.6"/>
          </button>
          <div class="welcome-add-label">{{ t('onboarding.welcome.add-profile') }}</div>
        </template>
        <template #default="{ close }">
          <button type="button" role="menuitem" data-dd-item class="px-dd-item welcome-item" @click="close(); openAddProfileDialog()">
            <icon-tabler-pencil width="16" height="16"/>{{ t('profiles.add') }}
          </button>
          <button type="button" role="menuitem" data-dd-item class="px-dd-item welcome-item" @click="close(); handlePaste()">
            <icon-tabler-clipboard width="16" height="16"/>{{ t('profiles.paste') }}
          </button>
          <button type="button" role="menuitem" data-dd-item class="px-dd-item welcome-item" @click="close(); openFile()">
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
.welcome {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
}

.welcome-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}

.welcome-title {
  font-size: 24px;
  font-weight: 700;
  letter-spacing: -.01em;
  margin: 0 0 10px;
}

.welcome-subtitle {
  font-size: 16px;
  color: var(--text-2);
  margin: 0 0 48px;
  text-wrap: pretty;
}

.welcome-dd {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
}

.welcome-dd :deep(.px-dd-panel) {
  top: 128px !important;
  left: 50% !important;
  transform: translateX(-50%);
}

.welcome-add {
  width: 120px;
  height: 120px;
  border-radius: 50%;
  border: none;
  background: var(--accent);
  color: var(--on-accent);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  box-shadow: 0 14px 36px color-mix(in srgb, var(--accent) 45%, transparent);
  transition: transform .2s;
}

.welcome-add:hover {
  transform: scale(1.05);
}

.welcome-add:active {
  transform: scale(.97);
}

.welcome-add-label {
  font-size: 16px;
  font-weight: 600;
}

.welcome-item {
  font-weight: 600;
  padding: 9px 12px;
  gap: 10px;
}

.welcome-item svg {
  color: var(--text-2);
  flex-shrink: 0;
}
</style>
