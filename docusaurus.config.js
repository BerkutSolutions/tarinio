const config = {
  title: 'TARINIO Docs',
  tagline: 'Enterprise-grade self-hosted WAF platform documentation',
  favicon: 'img/favicon.png',
  url: process.env.DOCS_SITE_URL || 'http://127.0.0.1:3000',
  baseUrl: process.env.DOCS_BASE_URL || '/',
  organizationName: 'BerkutSolutions',
  projectName: 'tarinio',
  deploymentBranch: 'gh-pages',
  onBrokenLinks: 'throw',
  trailingSlash: true,
  future: {
    faster: {
      rspackBundler: true,
    },
  },
  markdown: {
    hooks: {
      onBrokenMarkdownLinks: 'warn',
    },
  },

  presets: [
    [
      'classic',
      {
        docs: false,
        blog: false,
        pages: {},
        theme: {
          customCss: require.resolve('./src/css/custom.css'),
        },
      },
    ],
  ],

  plugins: [
    [
      '@easyops-cn/docusaurus-search-local',
      {
        indexDocs: true,
        indexPages: true,
        docsRouteBasePath: ['ru', 'en'],
        docsDir: ['docs/ru', 'docs/eng'],
        docsPluginIdForPreferredVersion: 'ru',
        language: ['en', 'ru'],
        explicitSearchResultPath: true,
        searchBarShortcut: true,
        searchBarPosition: 'right',
        hashed: 'filename',
        indexBlog: false,
      },
    ],
    [
      '@docusaurus/plugin-content-docs',
      {
        id: 'ru',
        path: 'docs/ru',
        routeBasePath: 'ru',
        exclude: ['README.md'],
        sidebarPath: require.resolve('./sidebars.ru.js'),
        editUrl: 'https://github.com/BerkutSolutions/tarinio/edit/main/docs/ru/',
      },
    ],
    [
      '@docusaurus/plugin-content-docs',
      {
        id: 'en',
        path: 'docs/eng',
        routeBasePath: 'en',
        exclude: ['README.md'],
        sidebarPath: require.resolve('./sidebars.en.js'),
        editUrl: 'https://github.com/BerkutSolutions/tarinio/edit/main/docs/eng/',
      },
    ],
  ],

  themeConfig: {
    image: 'img/logo.png',
    navbar: {
      title: 'TARINIO',
      logo: {
        alt: 'TARINIO logo',
        src: 'img/logo.png',
      },
      items: [
        {to: '/', label: 'Главная', position: 'left'},
        {to: '/docs-overview', label: 'Навигатор', position: 'left'},
        {to: '/ru/', label: 'Русская wiki', position: 'left'},
        {to: '/en/', label: 'English wiki', position: 'left'},
        {
          label: 'Язык',
          position: 'right',
          items: [
            {label: 'Русский', to: '/ru/'},
            {label: 'English', to: '/en/'},
          ],
        },
        {
          label: 'Версия 2.0.6',
          position: 'right',
          items: [
            {label: 'Текущий релиз 2.0.6', to: '/ru/release-policy/'},
            {label: 'CHANGELOG', href: 'https://github.com/BerkutSolutions/tarinio/blob/main/CHANGELOG.md'},
          ],
        },
        {type: 'search', position: 'right'},
        {
          href: 'https://github.com/BerkutSolutions/tarinio',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Документация',
          items: [
            {label: 'Русская wiki', to: '/ru/'},
            {label: 'English wiki', to: '/en/'},
            {label: 'Навигатор', to: '/docs-overview'},
          ],
        },
        {
          title: 'Enterprise',
          items: [
            {label: 'HA / Multi-Node', to: '/en/ha/'},
            {label: 'Observability', to: '/en/observability/'},
            {label: 'Benchmarks', to: '/en/benchmarks/'},
          ],
        },
        {
          title: 'Source',
          items: [
            {label: 'GitHub', href: 'https://github.com/BerkutSolutions/tarinio'},
          ],
        },
      ],
      copyright: `Copyright ${new Date().getFullYear()} Berkut Solutions`,
    },
    prism: {
      additionalLanguages: ['bash', 'powershell', 'json', 'yaml', 'go'],
    },
    docs: {
      sidebar: {
        hideable: true,
        autoCollapseCategories: false,
      },
    },
  },
};

module.exports = config;
