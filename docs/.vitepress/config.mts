import { defineConfig } from "vitepress";

// https://vitepress.dev/reference/site-config
// BASE_PATH is set by the docs deploy workflow; GitHub Pages serves the
// project site from /kube-osprey/.
export default defineConfig({
  base: process.env.BASE_PATH || "/",
  title: "KubeOsprey",
  description: "A modern Kubernetes dashboard",

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
  locales: {
    root: {
      label: "English",
      lang: "en",
    },
    zh: {
      label: "中文",
      lang: "zh-CN",
      link: "/zh/",
      title: "KubeOsprey",
      description: "一个现代 Kubernetes 仪表盘",
      themeConfig: {
        outline: { label: "本页目录" },
        sidebarMenuLabel: "目录",
        returnToTopLabel: "返回顶部",
        darkModeSwitchLabel: "外观",
        darkModeSwitchTitle: "切换至深色模式",
        lightModeSwitchTitle: "切换至浅色模式",
        nav: [
          { text: "首页", link: "/zh/" },
          { text: "指南", link: "/zh/guide/", activeMatch: "^/zh/guide/" },
          { text: "配置", link: "/zh/config/", activeMatch: "^/zh/config/" },
          { text: "插件", link: "/zh/plugins/", activeMatch: "^/zh/plugins/" },
          {
            text: "API",
            link: "/zh/api/authentication",
            activeMatch: "^/zh/api/",
          },
          { text: "常见问题", link: "/zh/faq" },
        ],
        editLink: {
          pattern: "https://github.com/fadewalk/kube-osprey/tree/main/docs/:path",
          text: "在 GitHub 上编辑此页面",
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
              button: {
                buttonText: "Search docs",
                buttonAriaLabel: "Search docs",
              },
            },
          },
          zh: {
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
        },
      },
    },
    langMenuLabel: "Language",
    editLink: {
      pattern: "https://github.com/fadewalk/kube-osprey/tree/main/docs/:path",
      text: "Edit this page on GitHub",
    },

    nav: [
      { text: "Home", link: "/" },
      { text: "Guide", link: "/guide/", activeMatch: "^/guide/" },
      { text: "Configuration", link: "/config/", activeMatch: "^/config/" },
      { text: "Plugins", link: "/plugins/", activeMatch: "^/plugins/" },
      { text: "API", link: "/api/authentication", activeMatch: "^/api/" },
      { text: "FAQ", link: "/faq" },
    ],

    sidebar: {
      "/": [
        {
          text: "Introduction",
          items: [
            { text: "What is KubeOsprey?", link: "/guide/" },
            { text: "Getting Started", link: "/guide/installation" },
          ],
        },
        {
          text: "Configuration",
          items: [
            { text: "User Management", link: "/config/user-management" },
            { text: "OAuth Setup", link: "/config/oauth-setup" },
            { text: "RBAC Configuration", link: "/config/rbac-config" },
            { text: "Prometheus Setup", link: "/config/prometheus-setup" },
            { text: "Notification Channels", link: "/config/notification-channels" },
            { text: "Managed K8s Auth", link: "/config/managed-k8s-auth" },
            { text: "Environment Variables", link: "/config/env" },
            { text: "Configuration File", link: "/config/config-file" },
            { text: "Chart Values", link: "/config/chart-values" },
            { text: "Security Hardening", link: "/config/security-hardening" },
          ],
        },
        {
          text: "Usage",
          items: [
            { text: "Global Search", link: "/guide/global-search" },
            { text: "Related Resources", link: "/guide/related-resources" },
            { text: "Logs", link: "/guide/logs" },
            { text: "Log Center", link: "/guide/log-center" },
            { text: "Monitor", link: "/guide/monitoring" },
            { text: "Alert Center", link: "/guide/alert-center" },
            { text: "Health Diagnostics", link: "/guide/health-diagnostics" },
            { text: "Batch Operations", link: "/guide/batch-operations" },
            { text: "Helm Management", link: "/guide/helm-management" },
            { text: "AI Assistant", link: "/guide/ai-assistant" },
            { text: "Change Plans", link: "/guide/change-plans" },
            { text: "Web Terminal", link: "/guide/web-terminal" },
            { text: "KubeOsprey Cluster Agent", link: "/guide/kite-cluster-agent" },
            { text: "Resource History", link: "/guide/resource-history" },
            { text: "Custom Sidebar", link: "/guide/custom-sidebar" },
            { text: "Kube Proxy", link: "/guide/kube-proxy" },
          ],
        },
        {
          text: "FAQ",
          link: "/faq",
        },
      ],
      "/api/": [
        {
          text: "Authentication",
          link: "/api/authentication",
        },
        {
          text: "Resources",
          link: "/api/resources",
        },
        {
          text: "Cluster Management",
          link: "/api/cluster-management",
        },
        {
          text: "RBAC Management",
          link: "/api/rbac-management",
        },
        {
          text: "User Management",
          link: "/api/user-management",
        },
      ],
      "/zh/": [
        {
          text: "介绍",
          items: [
            { text: "什么是 KubeOsprey?", link: "/zh/guide/" },
            { text: "开始", link: "/zh/guide/installation" },
          ],
        },
        {
          text: "配置",
          items: [
            { text: "用户管理", link: "/zh/config/user-management" },
            { text: "OAuth 设置", link: "/zh/config/oauth-setup" },
            { text: "RBAC 配置", link: "/zh/config/rbac-config" },
            { text: "Prometheus 设置", link: "/zh/config/prometheus-setup" },
            { text: "通知渠道", link: "/zh/config/notification-channels" },
            { text: "托管 K8s 认证", link: "/zh/config/managed-k8s-auth" },
            { text: "环境变量", link: "/zh/config/env" },
            { text: "配置文件", link: "/zh/config/config-file" },
            { text: "Chart Values", link: "/zh/config/chart-values" },
            { text: "安全加固", link: "/zh/config/security-hardening" },
          ],
        },
        {
          text: "使用指南",
          items: [
            { text: "全局搜索", link: "/zh/guide/global-search" },
            { text: "相关资源", link: "/zh/guide/related-resources" },
            { text: "日志", link: "/zh/guide/logs" },
            { text: "日志中心", link: "/zh/guide/log-center" },
            { text: "监控", link: "/zh/guide/monitoring" },
            { text: "告警中心", link: "/zh/guide/alert-center" },
            { text: "健康诊断", link: "/zh/guide/health-diagnostics" },
            { text: "批量运维", link: "/zh/guide/batch-operations" },
            { text: "Helm 管理", link: "/zh/guide/helm-management" },
            { text: "AI 助手", link: "/zh/guide/ai-assistant" },
            { text: "变更计划", link: "/zh/guide/change-plans" },
            { text: "Web 终端", link: "/zh/guide/web-terminal" },
            {
              text: "KubeOsprey Cluster Agent",
              link: "/zh/guide/kite-cluster-agent",
            },
            { text: "资源历史", link: "/zh/guide/resource-history" },
            { text: "自定义侧边栏", link: "/zh/guide/custom-sidebar" },
            { text: "Kube Proxy", link: "/zh/guide/kube-proxy" },
          ],
        },
        {
          text: "常见问题",
          link: "/zh/faq",
        },
      ],
      "/zh/api/": [
        {
          text: "认证",
          link: "/zh/api/authentication",
        },
        {
          text: "资源操作",
          link: "/zh/api/resources",
        },
        {
          text: "集群管理",
          link: "/zh/api/cluster-management",
        },
        {
          text: "RBAC 管理",
          link: "/zh/api/rbac-management",
        },
        {
          text: "用户管理",
          link: "/zh/api/user-management",
        },
      ],
      "/plugins/": [
        {
          text: "Getting Started",
          items: [
            { text: "Introduction to Plugins", link: "/plugins/" },
            { text: "Quick Start", link: "/plugins/quick-start" },
          ],
        },
        {
          text: "API Reference",
          items: [
            { text: "Plugin Identity", link: "/plugins/api/plugin-identity" },
            { text: "Plugin Configuration", link: "/plugins/api/plugin-config" },
            {
              text: "Resource Queries and Operations",
              link: "/plugins/api/resources",
            },
            { text: "UI Components", link: "/plugins/api/ui" },
            { text: "Page Navigation", link: "/plugins/api/navigation" },
            { text: "Host State and Interaction", link: "/plugins/api/hooks" },
            { text: "Metrics and Logs", link: "/plugins/api/observability" },
          ],
        },
        {
          text: "Guides",
          items: [
            { text: "Internationalization", link: "/plugins/i18n" },
            { text: "Debugging", link: "/plugins/debugging" },
            { text: "Publishing Plugins", link: "/plugins/publishing" },
          ],
        },
      ],
      "/zh/plugins/": [
        {
          text: "开始使用",
          items: [
            { text: "插件简介", link: "/zh/plugins/" },
            { text: "快速开始", link: "/zh/plugins/quick-start" },
          ],
        },
        {
          text: "API 参考",
          items: [
            { text: "插件身份", link: "/zh/plugins/api/plugin-identity" },
            { text: "插件配置", link: "/zh/plugins/api/plugin-config" },
            { text: "资源查询与操作", link: "/zh/plugins/api/resources" },
            { text: "UI 组件", link: "/zh/plugins/api/ui" },
            { text: "页面跳转", link: "/zh/plugins/api/navigation" },
            { text: "宿主状态与交互", link: "/zh/plugins/api/hooks" },
            { text: "指标与日志", link: "/zh/plugins/api/observability" },
          ],
        },
        {
          text: "指南",
          items: [
            { text: "国际化", link: "/zh/plugins/i18n" },
            { text: "调试", link: "/zh/plugins/debugging" },
            { text: "发布插件", link: "/zh/plugins/publishing" },
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
