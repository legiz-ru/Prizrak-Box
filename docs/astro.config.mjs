import starlight from '@astrojs/starlight';
import { defineConfig } from 'astro/config';

export default defineConfig({
  site: 'https://docs.prizrak.app',
  integrations: [
    starlight({
      title: 'Prizrak-Box',
      // The app's design tokens (a copy of src/styles/tokens.css) and their
      // mapping onto Starlight; see the comments at the top of each file.
      customCss: ['./src/styles/tokens.css', './src/styles/theme.css'],
      expressiveCode: {
        themes: ['github-dark', 'github-light'],
        useStarlightUiThemeColors: true,
        styleOverrides: {
          borderRadius: '8px',
          borderColor: 'var(--border)',
          codeBackground: 'var(--input-bg)',
          frames: {
            shadowColor: 'transparent',
            editorTabBarBackground: 'var(--panel-soft)',
            editorActiveTabBackground: 'var(--input-bg)',
            terminalTitlebarBackground: 'var(--panel-soft)',
            terminalBackground: 'var(--input-bg)',
          },
        },
      },
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
        src: './src/assets/appicon.png',
        alt: 'Prizrak-Box',
      },
      social: [
        { icon: 'github', label: 'GitHub', href: 'https://github.com/legiz-ru/Prizrak-Box' },
        { icon: 'telegram', label: 'Telegram', href: 'https://t.me/legiz_trashbag' },
      ],
      sidebar: [
        { label: 'Welcome', translations: { ru: 'Добро пожаловать' }, link: '/' },
        { label: 'Install App', translations: { ru: 'Установка' }, link: '/install/' },
        { label: 'About App', translations: { ru: 'О приложении' }, link: '/about/' },
        { label: 'Prizrak-Core', translations: { ru: 'Ядро Prizrak-Core' }, link: '/prizrak-core/' },
        { label: 'Links', translations: { ru: 'Ссылки' }, link: '/links/' },
        { label: 'Privacy Policy', translations: { ru: 'Политика конфиденциальности' }, link: '/important-note/' },
        { label: 'Terms of Use', translations: { ru: 'Условия использования' }, link: '/terms/' },
        { label: 'FAQ', translations: { ru: 'FAQ (Частые вопросы)' }, link: '/faq/' },
        {
          label: 'Integration',
          translations: { ru: 'Интеграция' },
          collapsed: true,
          items: [
            {
              label: 'Integration Overview',
              translations: { ru: 'Обзор интеграции' },
              link: '/for-devs/',
            },
            {
              label: 'Deep Linking',
              translations: { ru: 'Диплинки' },
              link: '/for-devs/deep-linking/',
            },
            {
              label: 'Android TV',
              link: '/for-devs/android-tv/',
            },
            {
              label: 'Supported Headers',
              translations: { ru: 'Поддерживаемые заголовки' },
              link: '/for-devs/supported-headers/',
            },
            {
              label: 'AGE Encryption',
              translations: { ru: 'Шифрование AGE' },
              link: '/for-devs/age-encryption/',
            },
            {
              label: 'Remnawave Integration',
              translations: { ru: 'Интеграция с Remnawave' },
              link: '/for-devs/remnawave-integration/',
            },
            {
              label: 'Smart Proxy Group',
              translations: { ru: 'Прокси группа Smart' },
              link: '/for-devs/smart-proxy-group/',
            },
            {
              label: 'Proxy and Group Descriptions',
              translations: { ru: 'Описания прокси и групп' },
              link: '/for-devs/proxy-descriptions/',
            },
          ],
        },
        {
          label: 'PC FAQ Video by CrazyOpS',
          translations: { ru: 'ПК FAQ Video by CrazyOpS' },
          link: 'https://t.me/crazy_day_admin/168',
          attrs: { target: '_blank', rel: 'noopener noreferrer' },
        },
      ],
      editLink: {
        baseUrl: 'https://github.com/legiz-ru/Prizrak-Box/edit/main/docs/',
      },
      lastUpdated: true,
      components: {
        PageFrame: './src/components/PageFrame.astro',
        Header: './src/components/Header.astro',
        SiteTitle: './src/components/SiteTitle.astro',
        Sidebar: './src/components/CustomSidebar.astro',
        TwoColumnContent: './src/components/TwoColumnContent.astro',
        ThemeSelect: './src/components/ThemeSelect.astro',
        LanguageSelect: './src/components/LanguageSelect.astro',
      },
    }),
  ],
});
