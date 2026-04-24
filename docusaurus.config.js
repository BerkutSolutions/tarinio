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
        {to: '/', label: 'Home', position: 'left'},
        {to: '/docs-overview', label: 'Navigator', position: 'left'},
        {to: '/ru/', label: 'Russian Wiki', position: 'left'},
        {to: '/en/', label: 'English Wiki', position: 'left'},
        {
          label: 'Language',
          position: 'right',
          items: [
            {label: 'Russian', to: '/ru/'},
            {label: 'English', to: '/en/'},
          ],
        },
        {
          label: 'Version 3.0.0',
          position: 'right',
          items: [
            {label: 'Current Release 3.0.0', to: '/ru/core-docs/release-policy/'},
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
          title: 'Documentation',
          items: [
            {label: 'Russian Wiki', to: '/ru/'},
            {label: 'English Wiki', to: '/en/'},
            {label: 'Navigator', to: '/docs-overview'},
          ],
        },
        {
          title: 'Enterprise',
          items: [
            {label: 'High Availability / Multi-Node', to: '/en/high-availability-docs/high-availability/'},
            {label: 'Observability', to: '/en/core-docs/observability/'},
            {label: 'Benchmarks', to: '/en/core-docs/benchmarks/'},
          ],
        },
        {
          title: 'Source',
          items: [{label: 'GitHub', href: 'https://github.com/BerkutSolutions/tarinio'}],
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
