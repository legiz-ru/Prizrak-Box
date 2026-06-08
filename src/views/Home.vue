<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, watch, toRaw } from 'vue';
import { Events, Service } from '@/runtime';
import { useOnboardingStore } from '@/store/onboardingStore';
import createApi from '@/api';
import ServiceSetup from '@/components/home/ServiceSetup.vue';
import WelcomeScreen from '@/components/home/WelcomeScreen.vue';
import ActiveProfile from '@/components/home/ActiveProfile.vue';
import FirstProfileModal from '@/components/home/FirstProfileModal.vue';

const { proxy } = getCurrentInstance()!;
const api = createApi(proxy);

const onboardingStore = useOnboardingStore();

// Состояния
const profiles = ref<any[]>([]);
const showFirstProfileModal = ref(false);
const isCheckingService = ref(false);

// Текущее состояние интерфейса
const currentState = computed(() => {
  // Состояние 0: Экран установки сервиса
  if (onboardingStore.shouldShowServiceSetup) {
    return 'service-setup';
  }

  // Состояние 1: Экран приветствия (нет профилей)
  if (!profiles.value || profiles.value.length === 0) {
    return 'welcome';
  }

  // Состояние 2: Экран активного профиля
  return 'active-profile';
});

// Обработчик завершения установки сервиса
function handleServiceSetupComplete() {
  // После установки/пропуска сервиса переходим к следующему состоянию
  // Состояние автоматически обновится через computed
}

// Получение списка профилей
async function getProfileList() {
  try {
    const list = await api.getProfileList();

    if (list && list.length > 0) {
      profiles.value = list;
      // Отмечаем, что у пользователя есть профили
      onboardingStore.markHasProfiles();
    } else {
      profiles.value = [];
    }

    // Отправляем событие обновления профилей
    Events.Emit({
      name: "profiles",
      data: toRaw(profiles.value)
    });
  } catch (error) {
    console.error('Failed to get profile list:', error);
    profiles.value = [];
  }
}

// Обработчик события изменения профилей
const handleProfilesEvent = (list: any[]) => {
  // Проверяем, был ли это переход с пустого состояния
  const wasEmpty = !onboardingStore.hasEverHadProfiles;

  if (Array.isArray(list) && list.length > 0) {
    profiles.value = list;

    // Если это первый добавленный профиль, показываем модальное окно
    if (wasEmpty && onboardingStore.shouldShowFirstProfileInfo) {
      showFirstProfileModal.value = true;
    }

    // Отмечаем, что у пользователя есть профили
    onboardingStore.markHasProfiles();
  } else {
    profiles.value = [];
  }
};

// Обработчик импорта профиля через deeplink
function handleProfilesImported(event: Event) {
  const customEvent = event as CustomEvent;
  const detail = customEvent.detail;

  if (!detail || !Array.isArray(detail.profiles)) {
    return;
  }

  const wasEmpty = !onboardingStore.hasEverHadProfiles;
  let added = false;

  for (const item of detail.profiles) {
    if (!item) {
      continue;
    }
    const exists = profiles.value.some(profile => profile['id'] === item['id']);
    if (!exists) {
      profiles.value.push(item);
      added = true;
    }
  }

  if (added) {
    // Если это первый добавленный профиль, показываем модальное окно
    if (wasEmpty && onboardingStore.shouldShowFirstProfileInfo) {
      showFirstProfileModal.value = true;
    }

    // Отмечаем, что у пользователя есть профили
    onboardingStore.markHasProfiles();

    Events.Emit({
      name: "profiles",
      data: toRaw(profiles.value)
    });
  }
}

// Обработчик обновления профилей из Vue компонентов
function handleVueProfilesUpdate(event: Event) {
  const customEvent = event as CustomEvent;
  const detail = customEvent.detail;

  if (!detail || !Array.isArray(detail.profiles)) {
    return;
  }

  handleProfilesEvent(detail.profiles);
}

// Жизненный цикл
onMounted(async () => {
  // Проверяем статус службы, если нужно показать экран установки
  if (onboardingStore.shouldShowServiceSetup) {
    isCheckingService.value = true;
    try {
      const isRunning = await Service.IsRunning();
      if (isRunning) {
        // Служба уже запущена, помечаем как установленную
        onboardingStore.markServiceInstalled();
      }
    } catch (error) {
      // Ошибка проверки - показываем экран установки
    } finally {
      isCheckingService.value = false;
    }
  }

  // Загружаем список профилей
  await getProfileList();

  // Подписываемся на события IPC (Electron)
  Events.On("profiles", handleProfilesEvent);

  // Подписываемся на события внутри Vue (браузерные CustomEvents)
  window.addEventListener('vue-profiles-updated', handleVueProfilesUpdate as EventListener);
  window.addEventListener('deeplink-profile-imported', handleProfilesImported as EventListener);
});

onBeforeUnmount(() => {
  // Отписываемся от событий
  Events.Off("profiles", handleProfilesEvent);
  window.removeEventListener('vue-profiles-updated', handleVueProfilesUpdate as EventListener);
  window.removeEventListener('deeplink-profile-imported', handleProfilesImported as EventListener);
});
</script>

<template>
  <MyLayout>
    <template #bottom>
      <!-- Состояние 0: Экран установки сервиса -->
      <ServiceSetup
        v-if="currentState === 'service-setup'"
        @complete="handleServiceSetupComplete"
      />

      <!-- Состояние 1: Экран приветствия -->
      <WelcomeScreen
        v-else-if="currentState === 'welcome'"
      />

      <!-- Состояние 2: Экран активного профиля -->
      <ActiveProfile
        v-else-if="currentState === 'active-profile'"
        :profiles="profiles"
      />

      <!-- Модальное окно после добавления первого профиля -->
      <FirstProfileModal
        v-model:visible="showFirstProfileModal"
      />
    </template>
  </MyLayout>
</template>

<style scoped>
:deep(.bottom) {
  padding-bottom: 0;
  overflow-y: hidden;
  overflow-x: hidden;
}
</style>
