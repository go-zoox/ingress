import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'Ingress',
  description: 'An Easy, Powerful, Flexible Reverse Proxy',
  base: '/ingress/',
  
  head: [
    ['link', { rel: 'icon', href: '/favicon.ico' }]
  ],

  locales: {
    root: {
      label: 'English',
      lang: 'en',
      themeConfig: {
        nav: [
          { text: 'Home', link: '/' },
          { text: 'Guide', link: '/guide/getting-started' },
          { text: 'Examples', link: '/examples/basic' },
          { text: 'GitHub', link: 'https://github.com/go-zoox/ingress' }
        ],
        sidebar: {
          '/guide/': [
            {
              text: 'Getting Started',
              items: [
                { text: 'Introduction', link: '/guide/getting-started' }
              ]
            },
            {
              text: 'Configuration',
              items: [
                { text: 'Configuration Reference', link: '/guide/configuration' }
              ]
            },
            {
              text: 'Features',
              items: [
                { text: 'Routing', link: '/guide/routing' },
                { text: 'Admin Console', link: '/guide/admin' },
                { text: 'Scheduled Jobs', link: '/guide/jobs' },
                { text: 'Web Application Firewall', link: '/guide/waf' },
                { text: 'Maintenance Mode', link: '/guide/maintenance' },
                { text: 'Security Headers', link: '/guide/security-headers' },
                { text: 'Authentication', link: '/guide/authentication' },
                { text: 'SSL/TLS', link: '/guide/ssl-tls' },
                { text: 'Health Checks', link: '/guide/health-checks' },
                { text: 'Caching', link: '/guide/caching' },
                { text: 'Rewriting', link: '/guide/rewriting' }
              ]
            }
          ],
          '/examples/': [
            {
              text: 'Examples',
              items: [
                { text: 'Basic Setup', link: '/examples/basic' },
                { text: 'Path Routing', link: '/examples/path-routing' },
                { text: 'Authentication', link: '/examples/authentication' },
                { text: 'SSL/TLS', link: '/examples/ssl' },
                { text: 'Redirects', link: '/examples/redirect' },
                { text: 'Handler Backends', link: '/examples/handler' },
                { text: 'WAF', link: '/examples/waf' },
                { text: 'Maintenance', link: '/examples/maintenance' },
                { text: 'Security headers', link: '/examples/security-headers' },
                { text: 'Admin Console', link: '/examples/admin-console' },
                { text: 'Admin Auth & RBAC', link: '/examples/admin-auth' },
                { text: 'Scheduled Jobs', link: '/examples/jobs' },
                { text: 'Advanced', link: '/examples/advanced' }
              ]
            }
          ]
        }
      }
    },
    zh: {
      label: '简体中文',
      lang: 'zh-CN',
      link: '/zh/',
      themeConfig: {
        nav: [
          { text: '首页', link: '/zh/' },
          { text: '指南', link: '/zh/guide/getting-started' },
          { text: '示例', link: '/zh/examples/basic' },
          { text: 'GitHub', link: 'https://github.com/go-zoox/ingress' }
        ],
        sidebar: {
          '/zh/guide/': [
            {
              text: '快速开始',
              items: [
                { text: '介绍', link: '/zh/guide/getting-started' }
              ]
            },
            {
              text: '配置',
              items: [
                { text: '配置参考', link: '/zh/guide/configuration' }
              ]
            },
            {
              text: '功能',
              items: [
                { text: '路由', link: '/zh/guide/routing' },
                { text: 'Admin 控制台', link: '/zh/guide/admin' },
                { text: '定时任务', link: '/zh/guide/jobs' },
                { text: 'Web 应用防火墙（WAF）', link: '/zh/guide/waf' },
                { text: '维护模式', link: '/zh/guide/maintenance' },
                { text: '安全响应头', link: '/zh/guide/security-headers' },
                { text: '认证', link: '/zh/guide/authentication' },
                { text: 'SSL/TLS', link: '/zh/guide/ssl-tls' },
                { text: '健康检查', link: '/zh/guide/health-checks' },
                { text: '缓存', link: '/zh/guide/caching' },
                { text: '重写', link: '/zh/guide/rewriting' }
              ]
            }
          ],
          '/zh/examples/': [
            {
              text: '示例',
              items: [
                { text: '基础设置', link: '/zh/examples/basic' },
                { text: '路径路由', link: '/zh/examples/path-routing' },
                { text: '认证', link: '/zh/examples/authentication' },
                { text: 'SSL/TLS', link: '/zh/examples/ssl' },
                { text: '重定向', link: '/zh/examples/redirect' },
                { text: 'Handler 后端', link: '/zh/examples/handler' },
                { text: 'WAF', link: '/zh/examples/waf' },
                { text: '维护模式', link: '/zh/examples/maintenance' },
                { text: '安全响应头', link: '/zh/examples/security-headers' },
                { text: 'Admin 控制台', link: '/zh/examples/admin-console' },
                { text: 'Admin 认证与 RBAC', link: '/zh/examples/admin-auth' },
                { text: '定时任务', link: '/zh/examples/jobs' },
                { text: '高级用法', link: '/zh/examples/advanced' }
              ]
            }
          ]
        }
      }
    }
  },

  themeConfig: {
    search: {
      provider: 'local'
    },
    socialLinks: [
      { icon: 'github', link: 'https://github.com/go-zoox/ingress' }
    ],
    editLink: {
      pattern: 'https://github.com/go-zoox/ingress/edit/master/docs/:path',
      text: 'Edit this page on GitHub'
    },
    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2024 GoZoox'
    }
  }
})
