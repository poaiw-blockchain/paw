import { defineConfig } from 'vitepress'

export default defineConfig({
  ignoreDeadLinks: true,
  title: 'PAW Blockchain',
  description: 'Comprehensive documentation for PAW Blockchain - A lean Layer-1 blockchain with built-in DEX, secure compute aggregation, and mobile-ready wallets',

  // Base configuration
  lang: 'en-US',
  base: '/docs/',

  // Theme configuration
  themeConfig: {
    // Logo and site title
    logo: '/logo.svg',
    siteTitle: 'PAW Blockchain',

    // Navigation
    nav: [
      { text: 'Home', link: '/' },
      { text: 'Guide', link: '/guide/getting-started' },
      { text: 'Developer', link: '/developer/quick-start' },
      { text: 'Validator', link: '/validator/setup' },
      {
        text: 'Resources',
        items: [
          { text: 'FAQ', link: '/faq' },
          { text: 'Glossary', link: '/glossary' },
          { text: 'Architecture', link: '/reference/architecture' },
          { text: 'Tokenomics', link: '/reference/tokenomics' }
        ]
      },
      {
        text: 'v1.0',
        items: [
          { text: 'v1.0 (Current)', link: '/' },
          { text: 'Changelog', link: '/changelog' }
        ]
      }
    ],

    // Sidebar
    sidebar: {
      '/guide/': [
        {
          text: 'User Guide',
          items: [
            { text: 'Getting Started', link: '/guide/getting-started' },
            { text: 'Creating a Wallet', link: '/guide/wallets' },
            { text: 'Using the DEX', link: '/guide/dex' },
            { text: 'Staking Guide', link: '/guide/staking' },
            { text: 'Governance', link: '/guide/governance' }
          ]
        }
      ],
      '/developer/': [
        {
          text: 'Developer Guide',
          items: [
            { text: 'Quick Start', link: '/developer/quick-start' },
            { text: 'JavaScript SDK', link: '/developer/javascript-sdk' },
            { text: 'Python SDK', link: '/developer/python-sdk' },
            { text: 'Go Development', link: '/developer/go-development' },
            { text: 'Smart Contracts', link: '/developer/smart-contracts' },
            { text: 'Module Development', link: '/developer/module-development' },
            { text: 'API Reference', link: '/developer/api' }
          ]
        }
      ],
      '/validator/': [
        {
          text: 'Validator Guide',
          items: [
            { text: 'Setup Guide', link: '/validator/setup' },
            { text: 'Operations', link: '/validator/operations' },
            { text: 'Security', link: '/validator/security' },
            { text: 'Monitoring', link: '/validator/monitoring' },
            { text: 'Troubleshooting', link: '/validator/troubleshooting' }
          ]
        }
      ],
      '/reference/': [
        {
          text: 'Reference',
          items: [
            { text: 'Architecture', link: '/reference/architecture' },
            { text: 'Tokenomics', link: '/reference/tokenomics' },
            { text: 'Network Specs', link: '/reference/network-specs' }
          ]
        }
      ]
    },

    // Social links
    socialLinks: [
      { icon: 'github', link: 'https://github.com/decristofaroj/paw' },
      { icon: 'twitter', link: 'https://twitter.com/pawblockchain' },
      { icon: 'discord', link: 'https://discord.gg/pawblockchain' }
    ],

    // Edit link
    editLink: {
      pattern: 'https://github.com/decristofaroj/paw/edit/master/docs/portal/:path',
      text: 'Edit this page on GitHub'
    },

    // Footer
    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2025 PAW Blockchain'
    },

    // Search
    search: {
      provider: 'local',
      options: {
        detailedView: true,
        miniSearch: {
          searchOptions: {
            fuzzy: 0.2,
            prefix: true,
            boost: {
              title: 4,
              text: 2,
              titles: 1
            }
          }
        }
      }
    },

    // Last updated
    lastUpdated: {
      text: 'Last updated',
      formatOptions: {
        dateStyle: 'medium',
        timeStyle: 'short'
      }
    }
  },

  // Markdown configuration
  markdown: {
    lineNumbers: true,
    theme: {
      light: 'github-light',
      dark: 'github-dark'
    }
  },

  // Head configuration
  head: [
    ['link', { rel: 'icon', href: '/favicon.ico' }],
    ['meta', { name: 'theme-color', content: '#3eaf7c' }],
    ['meta', { name: 'og:type', content: 'website' }],
    ['meta', { name: 'og:locale', content: 'en' }],
    ['meta', { name: 'og:site_name', content: 'PAW Blockchain Documentation' }],
    ['meta', { name: 'og:image', content: '/og-image.png' }]
  ],

  // Multi-language support ready
  locales: {
    root: {
      label: 'English',
      lang: 'en'
    }
  }
})
