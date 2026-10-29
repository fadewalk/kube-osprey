import { defineConfig } from "vitepress";

// https://vitepress.dev/reference/site-config
// BASE_PATH is set by the docs deploy workflow; GitHub Pages serves the
// project site from /kube-osprey/.
// 中文 is the default locale at the root; English lives under /en/.
export default defineConfig({
  base: process.env.BASE_PATH || "/",
  title: "KubeOsprey",
  description: "一个现代 Kubernetes 仪表盘",

  sitemap: {
    hostname: "https://fadewalk.github.io/kube-osprey/",
    lastmodDateOnly: false,
  },

  markdown: {
    image: {
      lazyLoading: true,
    },
  },

  lastUpdated: true,

  // Preload the home page fonts with the configured base; frontmatter head
  // values are static and cannot express the base path.
  transformHead: ({ pageData, siteConfig }) => {
    if (!["index.md", "en/index.md"].includes(pageData.relativePath)) return [];
    return ["cormorant-garamond-latin-400-normal", "dm-mono-latin-400-normal", "dm-mono-latin-500-normal"].map((font) => [
      "link",
      {
        rel: "preload",
        href: `${siteConfig.base}assets/fonts/${font}.woff2`,
        as: "font",
        type: "font/woff2",
        crossorigin: "",
      },
    ]);
  },

  locales: {
    root: {
      label: "中文",
      lang: "zh-CN",
    },
    en: {
      label: "English",
      lang: "en",
      link: "/en/",
      title: "KubeOsprey",
      description: "A modern Kubernetes dashboard",
      themeConfig: {
        outline: { label: "On this page" },
        sidebarMenuLabel: "Menu",
        returnToTopLabel: "Back to top",
        darkModeSwitchLabel: "Appearance",
        darkModeSwitchTitle: "Switch to dark theme",
        lightModeSwitchTitle: "Switch to light theme",
        nav: [
          { text: "Home", link: "/en/" },
          { text: "Guide", link: "/en/guide/", activeMatch: "^/en/guide/" },
          {
            text: "Configuration",
            link: "/en/config/",
            activeMatch: "^/en/config/",
          },
          {
            text: "Plugins",
            link: "/en/plugins/",
            activeMatch: "^/en/plugins/",
          },
          {
            text: "API",
            link: "/en/api/authentication",
            activeMatch: "^/en/api/",
          },
          { text: "FAQ", link: "/en/faq" },
        ],
        editLink: {
          pattern: "https://github.com/fadewalk/kube-osprey/tree/main/docs/:path",
          text: "Edit this page on GitHub",
        },
      },
    },
  },

  head: [
    [
      "link",
      { rel: "icon", href: `${process.env.BASE_PATH || ""}/logo.svg` },
    ],
  ],

  themeConfig: {
    // https://vitepress.dev/reference/default-theme-config
    logo: "/logo.svg",
    search: {
      provider: "local",
      options: {
        locales: {
          root: {
            translations: {
              button: { buttonText: "搜索文档", buttonAriaLabel: "搜索文档" },
              modal: {
                displayDetails: "显示详细列表",
                resetButtonTitle: "清除搜索",
                backButtonTitle: "关闭搜索",
                noResultsText: "无法找到相关结果",
                footer: {
                  selectText: "选择",
                  selectKeyAriaLabel: "回车键",
                  navigateText: "切换",
                  navigateUpKeyAriaLabel: "向上箭头",
                  navigateDownKeyAriaLabel: "向下箭头",
                  closeText: "关闭",
                  closeKeyAriaLabel: "Esc 键",
                },
              },
            },
          },
          en: {
            translations: {
              button: {
                buttonText: "Search docs",
                buttonAriaLabel: "Search docs",
              },
            },
          },
        },
      },
    },
    langMenuLabel: "语言",
    editLink: {
      pattern: "https://github.com/fadewalk/kube-osprey/tree/main/docs/:path",
      text: "在 GitHub 上编辑此页面",
    },

    nav: [
      { text: "首页", link: "/" },
      { text: "指南", link: "/guide/", activeMatch: "^/guide/" },
      { text: "配置", link: "/config/", activeMatch: "^/config/" },
      { text: "插件", link: "/plugins/", activeMatch: "^/plugins/" },
      { text: "API", link: "/api/authentication", activeMatch: "^/api/" },
      { text: "常见问题", link: "/faq" },
    ],

    sidebar: {
      "/": [
        {
          text: "介绍",
          items: [
            { text: "什么是 KubeOsprey?", link: "/guide/" },
            { text: "开始", link: "/guide/installation" },
            { text: "架构", link: "/guide/architecture" },
          ],
        },
        {
          text: "配置",
          items: [
            { text: "用户管理", link: "/config/user-management" },
            { text: "OAuth 设置", link: "/config/oauth-setup" },
            { text: "RBAC 配置", link: "/config/rbac-config" },
            { text: "Prometheus 设置", link: "/config/prometheus-setup" },
            { text: "通知渠道", link: "/config/notification-channels" },
            { text: "托管 K8s 认证", link: "/config/managed-k8s-auth" },
            { text: "环境变量", link: "/config/env" },
            { text: "配置文件", link: "/config/config-file" },
            { text: "Chart Values", link: "/config/chart-values" },
            { text: "安全加固", link: "/config/security-hardening" },
          ],
        },
        {
          text: "使用指南",
          items: [
            { text: "全局搜索", link: "/guide/global-search" },
            { text: "资源同步", link: "/guide/resource-sync" },
            { text: "相关资源", link: "/guide/related-resources" },
            { text: "日志", link: "/guide/logs" },
            { text: "日志中心", link: "/guide/log-center" },
            { text: "监控", link: "/guide/monitoring" },
            { text: "告警中心", link: "/guide/alert-center" },
            { text: "健康诊断", link: "/guide/health-diagnostics" },
            { text: "批量运维", link: "/guide/batch-operations" },
            { text: "Helm 管理", link: "/guide/helm-management" },
            { text: "AI 助手", link: "/guide/ai-assistant" },
            { text: "MCP 服务器", link: "/guide/mcp-server" },
            { text: "变更计划", link: "/guide/change-plans" },
            { text: "Web 终端", link: "/guide/web-terminal" },
            {
              text: "KubeOsprey Cluster Agent",
              link: "/guide/cluster-agent",
            },
            { text: "资源历史", link: "/guide/resource-history" },
            { text: "自定义侧边栏", link: "/guide/custom-sidebar" },
            { text: "Kube Proxy", link: "/guide/kube-proxy" },
          ],
        },
        {
          text: "常见问题",
          link: "/faq",
        },
      ],
      "/api/": [
        {
          text: "认证",
          link: "/api/authentication",
        },
        {
          text: "资源操作",
          link: "/api/resources",
        },
        {
          text: "集群管理",
          link: "/api/cluster-management",
        },
        {
          text: "RBAC 管理",
          link: "/api/rbac-management",
        },
        {
          text: "用户管理",
          link: "/api/user-management",
        },
      ],
      "/plugins/": [
        {
          text: "开始使用",
          items: [
            { text: "插件简介", link: "/plugins/" },
            { text: "快速开始", link: "/plugins/quick-start" },
          ],
        },
        {
          text: "API 参考",
          items: [
            { text: "插件身份", link: "/plugins/api/plugin-identity" },
            { text: "插件配置", link: "/plugins/api/plugin-config" },
            { text: "资源查询与操作", link: "/plugins/api/resources" },
            { text: "UI 组件", link: "/plugins/api/ui" },
            { text: "页面跳转", link: "/plugins/api/navigation" },
            { text: "宿主状态与交互", link: "/plugins/api/hooks" },
            { text: "指标与日志", link: "/plugins/api/observability" },
          ],
        },
        {
          text: "指南",
          items: [
            { text: "国际化", link: "/plugins/i18n" },
            { text: "调试", link: "/plugins/debugging" },
            { text: "发布插件", link: "/plugins/publishing" },
          ],
        },
      ],
      "/en/": [
        {
          text: "Introduction",
          items: [
            { text: "What is KubeOsprey?", link: "/en/guide/" },
            { text: "Getting Started", link: "/en/guide/installation" },
            { text: "Architecture", link: "/en/guide/architecture" },
          ],
        },
        {
          text: "Configuration",
          items: [
            { text: "User Management", link: "/en/config/user-management" },
            { text: "OAuth Setup", link: "/en/config/oauth-setup" },
            { text: "RBAC Configuration", link: "/en/config/rbac-config" },
            { text: "Prometheus Setup", link: "/en/config/prometheus-setup" },
            {
              text: "Notification Channels",
              link: "/en/config/notification-channels",
            },
            { text: "Managed K8s Auth", link: "/en/config/managed-k8s-auth" },
            { text: "Environment Variables", link: "/en/config/env" },
            { text: "Configuration File", link: "/en/config/config-file" },
            { text: "Chart Values", link: "/en/config/chart-values" },
            { text: "Security Hardening", link: "/en/config/security-hardening" },
          ],
        },
        {
          text: "Usage",
          items: [
            { text: "Global Search", link: "/en/guide/global-search" },
            { text: "Resource Sync", link: "/en/guide/resource-sync" },
            { text: "Related Resources", link: "/en/guide/related-resources" },
            { text: "Logs", link: "/en/guide/logs" },
            { text: "Log Center", link: "/en/guide/log-center" },
            { text: "Monitor", link: "/en/guide/monitoring" },
            { text: "Alert Center", link: "/en/guide/alert-center" },
            { text: "Health Diagnostics", link: "/en/guide/health-diagnostics" },
            { text: "Batch Operations", link: "/en/guide/batch-operations" },
            { text: "Helm Management", link: "/en/guide/helm-management" },
            { text: "AI Assistant", link: "/en/guide/ai-assistant" },
            { text: "MCP Server", link: "/en/guide/mcp-server" },
            { text: "Change Plans", link: "/en/guide/change-plans" },
            { text: "Web Terminal", link: "/en/guide/web-terminal" },
            {
              text: "KubeOsprey Cluster Agent",
              link: "/en/guide/cluster-agent",
            },
            { text: "Resource History", link: "/en/guide/resource-history" },
            { text: "Custom Sidebar", link: "/en/guide/custom-sidebar" },
            { text: "Kube Proxy", link: "/en/guide/kube-proxy" },
          ],
        },
        {
          text: "FAQ",
          link: "/en/faq",
        },
      ],
      "/en/api/": [
        {
          text: "Authentication",
          link: "/en/api/authentication",
        },
        {
          text: "Resources",
          link: "/en/api/resources",
        },
        {
          text: "Cluster Management",
          link: "/en/api/cluster-management",
        },
        {
          text: "RBAC Management",
          link: "/en/api/rbac-management",
        },
        {
          text: "User Management",
          link: "/en/api/user-management",
        },
      ],
      "/en/plugins/": [
        {
          text: "Getting Started",
          items: [
            { text: "Introduction to Plugins", link: "/en/plugins/" },
            { text: "Quick Start", link: "/en/plugins/quick-start" },
          ],
        },
        {
          text: "API Reference",
          items: [
            { text: "Plugin Identity", link: "/en/plugins/api/plugin-identity" },
            { text: "Plugin Configuration", link: "/en/plugins/api/plugin-config" },
            {
              text: "Resource Queries and Operations",
              link: "/en/plugins/api/resources",
            },
            { text: "UI Components", link: "/en/plugins/api/ui" },
            { text: "Page Navigation", link: "/en/plugins/api/navigation" },
            { text: "Host State and Interaction", link: "/en/plugins/api/hooks" },
            { text: "Metrics and Logs", link: "/en/plugins/api/observability" },
          ],
        },
        {
          text: "Guides",
          items: [
            { text: "Internationalization", link: "/en/plugins/i18n" },
            { text: "Debugging", link: "/en/plugins/debugging" },
            { text: "Publishing Plugins", link: "/en/plugins/publishing" },
          ],
        },
      ],
    },

    socialLinks: [
      { icon: "github", link: "https://github.com/fadewalk/kube-osprey" },
    ],

    footer: {
      message: "Released under the Apache License.",
      copyright: "Copyright © 2025-present KubeOsprey Contributors",
    },
  },
});
