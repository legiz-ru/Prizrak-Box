import { defineStore } from 'pinia';
import { defaultPersist } from "@/types/persist";

export const useOnboardingStore = defineStore('onboarding', {
    state: () => ({
        // Флаги состояния установки сервиса
        serviceSetupShown: false,        // Показан ли экран установки сервиса
        serviceSkipped: false,           // Пропущена ли установка сервиса
        serviceInstalled: false,         // Установлен ли сервис успешно

        // Флаг показа информационного модального окна
        firstProfileInfoShown: false,    // Показано ли модальное окно после первого профиля

        // Флаг наличия профилей (persistent)
        hasEverHadProfiles: false,       // Были ли когда-либо профили у пользователя
    }),

    getters: {
        // Нужно ли показывать экран установки сервиса
        // Показывается только если ни один из трёх флагов не установлен
        shouldShowServiceSetup(): boolean {
            return !this.serviceSetupShown && !this.serviceSkipped && !this.serviceInstalled;
        },

        // Нужно ли показывать информационное модальное окно
        shouldShowFirstProfileInfo(): boolean {
            return !this.firstProfileInfoShown;
        },
    },

    actions: {
        // Отметить, что экран установки был показан
        markServiceSetupShown() {
            this.serviceSetupShown = true;
        },

        // Отметить, что установка была пропущена
        markServiceSkipped() {
            this.serviceSkipped = true;
            this.serviceSetupShown = true;
        },

        // Отметить, что сервис был успешно установлен
        markServiceInstalled() {
            this.serviceInstalled = true;
            this.serviceSetupShown = true;
        },

        // Отметить, что информационное модальное окно было показано
        markFirstProfileInfoShown() {
            this.firstProfileInfoShown = true;
        },

        // Отметить, что у пользователя есть профили
        markHasProfiles() {
            this.hasEverHadProfiles = true;
        },

        // Сброс всех флагов (для тестирования)
        resetAll() {
            this.serviceSetupShown = false;
            this.serviceSkipped = false;
            this.serviceInstalled = false;
            this.firstProfileInfoShown = false;
            this.hasEverHadProfiles = false;
        },
    },

    persist: defaultPersist,
});
