import starlight from '@astrojs/starlight';
import { defineConfig } from 'astro/config';
import starlightThemeNord from 'starlight-theme-nord';

export default defineConfig({
  integrations: [
    starlight({
      plugins: [starlightThemeNord()],
      title: 'Prizrak Box',
      logo: {
        src: './src/assets/logo.svg',
        replacesTitle: false,
      },
      social: [
        { icon: 'github', label: 'GitHub', href: 'https://github.com/legiz-ru/Prizrak-Box' },
        { icon: 'telegram', label: 'Telegram', href: 'https://t.me/prizrak_box' },
      ],
      sidebar: [
        { label: 'Welcome', link: '/' },
        { label: 'Install App', link: '/install/' },
        { label: 'About App', link: '/about/' },
        { label: 'Deep Linking', link: '/deep-linking/' },
        { label: 'Links', link: '/links/' },
        { label: 'Important Note', link: '/important-note/' },
        {
          label: 'For Devs',
          items: [
            { label: 'Supported Headers', link: '/for-devs/supported-headers/' },
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
