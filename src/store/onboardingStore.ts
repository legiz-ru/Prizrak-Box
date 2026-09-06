import { defineStore } from 'pinia';
import { defaultPersist } from "@/types/persist";

export const useOnboardingStore = defineStore('onboarding', {
    state: () => ({
        // Флаг показа информационного модального окна
        firstProfileInfoShown: false,    // Показано ли модальное окно после первого профиля

        // Флаг наличия профилей (persistent)
        hasEverHadProfiles: false,       // Были ли когда-либо профили у пользователя
    }),

    getters: {
        // Нужно ли показывать информационное модальное окно
        shouldShowFirstProfileInfo(): boolean {
            return !this.firstProfileInfoShown;
        },
    },

    actions: {
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
            this.firstProfileInfoShown = false;
            this.hasEverHadProfiles = false;
        },
    },

    persist: defaultPersist,
});
