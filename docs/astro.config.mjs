import starlight from '@astrojs/starlight';
import { defineConfig } from 'astro/config';
import starlightThemeNord from 'starlight-theme-nord';

export default defineConfig({
  integrations: [
    starlight({
      plugins: [starlightThemeNord()],
      title: 'Prizrak Box',
      defaultLocale: 'root',
      locales: {
        root: {
          label: 'English',
          lang: 'en',
        },
        ru: {
          label: 'Русский',
          lang: 'ru',
        },
      },
      logo: {
        src: './src/assets/logo.svg',
        replacesTitle: false,
      },
      social: [
        { icon: 'github', label: 'GitHub', href: 'https://github.com/legiz-ru/Prizrak-Box' },
        { icon: 'telegram', label: 'Telegram', href: 'https://t.me/prizrak_box' },
      ],
      sidebar: [
        { label: 'Welcome', translations: { ru: 'Добро пожаловать' }, link: '/' },
        { label: 'Install App', translations: { ru: 'Установка' }, link: '/install/' },
        { label: 'About App', translations: { ru: 'О приложении' }, link: '/about/' },
        { label: 'Android TV', link: '/android-tv/' },
        { label: 'Deep Linking', translations: { ru: 'Диплинки' }, link: '/deep-linking/' },
        { label: 'AGE Encryption', translations: { ru: 'Шифрование AGE' }, link: '/age-encryption/' },
        { label: 'Links', translations: { ru: 'Ссылки' }, link: '/links/' },
        { label: 'Important Note', translations: { ru: 'Важно' }, link: '/important-note/' },
        {
          label: 'For Devs',
          translations: { ru: 'Для разработчиков' },
          items: [
            {
              label: 'Supported Headers',
              translations: { ru: 'Поддерживаемые заголовки' },
              link: '/for-devs/supported-headers/',
            },
          ],
        },
        {
          label: 'FAQ Video by CrazyOpS',
          link: 'https://t.me/crazy_day_admin/168',
          attrs: { target: '_blank', rel: 'noopener noreferrer' },
        },
      ],
      editLink: {
        baseUrl: 'https://github.com/legiz-ru/Prizrak-Box/edit/main/docs/',
      },
      lastUpdated: true,
    }),
  ],
});
