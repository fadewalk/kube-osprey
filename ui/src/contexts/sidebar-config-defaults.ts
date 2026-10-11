import type { ComponentType } from 'react'
import {
  IconBell,
  IconBox,
  IconChartLine,
  IconClipboardCheck,
  IconFileText,
  IconGauge,
  IconLoadBalancer,
  IconRoute,
  IconTopologyBus,
  type Icon,
  type IconProps,
} from '@tabler/icons-react'

import {
  DefaultMenus,
  MenuItemData,
  SidebarConfig,
  SidebarGroup,
  SidebarItem,
} from '@/types/sidebar'
import {
  getResourceIconComponent,
  resourceCatalog,
  resourceIconMap,
  sidebarGroupOrder,
} from '@/lib/resource-catalog'

const sidebarIconMap = resourceIconMap
type CatalogResource = (typeof resourceCatalog)[number]
type StoredSidebarItem = {
  id: string
  titleKey: string
  url?: string
  icon: string
  visible: boolean
  pinned: boolean
  order: number
  type?: SidebarItem['type']
  apiGroup?: string
  crdGroup?: string
  children?: StoredSidebarItem[]
}
type StoredSidebarConfig = Omit<SidebarConfig, 'groups'> & {
  groups: Array<Omit<SidebarGroup, 'items'> & { items: StoredSidebarItem[] }>
}
type SidebarResource = CatalogResource & {
  sidebar: {
    groupKey: (typeof sidebarGroupOrder)[number]
    order: number
    titleKey?: string
    defaultHidden?: boolean
  }
}

function hasSidebar(resource: CatalogResource): resource is SidebarResource {
  return 'sidebar' in resource
}

const defaultMenus: DefaultMenus = Object.fromEntries(
  sidebarGroupOrder.map((groupKey) => [groupKey, []])
) as DefaultMenus

resourceCatalog
  .filter(hasSidebar)
  .slice()
  .sort((a, b) => a.sidebar.order - b.sidebar.order)
  .forEach((resource) => {
    const sidebar = resource.sidebar
    defaultMenus[sidebar.groupKey].push({
      titleKey:
        sidebar.titleKey ||
        ('titleKey' in resource ? resource.titleKey : undefined) ||
        resource.pluralLabel,
      url: `/${resource.type}`,
      icon: getResourceIconComponent(resource.icon),
      defaultHidden: sidebar.defaultHidden,
    })
  })

defaultMenus['sidebar.groups.application'].push({
  titleKey: 'nav.prometheusExplorer',
  url: '/prometheus',
  icon: IconChartLine,
})
defaultMenus['sidebar.groups.application'].push({
  titleKey: 'nav.alertCenter',
  url: '/alerts',
  icon: IconBell,
})
defaultMenus['sidebar.groups.application'].push({
  titleKey: 'nav.logCenter',
  url: '/logs',
  icon: IconFileText,
})
defaultMenus['sidebar.groups.application'].push({
  titleKey: 'nav.diagnostics',
  url: '/diagnostics',
  icon: IconGauge,
})
defaultMenus['sidebar.groups.application'].push({
  titleKey: 'nav.changePlans',
  url: '/change-plans',
  icon: IconClipboardCheck,
})
defaultMenus['sidebar.groups.application'].push({
  titleKey: 'nav.topology',
  url: '/topology',
  icon: IconTopologyBus,
})
defaultMenus['sidebar.groups.application'].push({
  titleKey: 'nav.helmCharts',
  url: '/charts',
  icon: getResourceIconComponent('IconPackage'),
})

// Kuboard-style nested subgroups: entries listed here are pulled out of
// their flat group and rendered as collapsible parents with children.
const nestedSubgroups: Record<string, MenuItemData[]> = {}
{
  const traffic = defaultMenus['sidebar.groups.traffic']
  const take = (url: string): MenuItemData | undefined => {
    const index = traffic.findIndex((item) => item.url === url)
    return index >= 0 ? traffic.splice(index, 1)[0] : undefined
  }
  const subgroup = (
    titleKey: string,
    icon: ComponentType<{ className?: string }>,
    urls: string[]
  ): MenuItemData | undefined => {
    const children = urls
      .map(take)
      .filter((item): item is MenuItemData => item !== undefined)
    if (children.length === 0) return undefined
    return { titleKey, icon, url: '', children }
  }

  const subgroups = [
    subgroup('sidebar.subgroups.servicesRouting', IconRoute, [
      '/services',
      '/ingresses',
      '/networkpolicies',
    ]),
    subgroup('sidebar.subgroups.gatewayApi', IconLoadBalancer, [
      '/gateways',
      '/httproutes',
    ]),
  ].filter((item): item is MenuItemData => item !== undefined)

  if (subgroups.length > 0) {
    nestedSubgroups['sidebar.groups.traffic'] = subgroups
  }
  // Subgroups render after the remaining flat entries of their group.
  for (const [groupKey, parents] of Object.entries(nestedSubgroups)) {
    defaultMenus[groupKey].push(...parents)
  }
}

export const SIDEBAR_CONFIG_VERSION = 9

function getIconName(iconComponent: ComponentType<{ className?: string }>) {
  const entry = Object.entries(sidebarIconMap).find(
    ([, component]) => component === iconComponent
  )
  return entry ? entry[0] : 'IconBox'
}

export function getSidebarIconComponent(
  iconName: string
):
  | React.ForwardRefExoticComponent<IconProps & React.RefAttributes<Icon>>
  | React.ElementType {
  return sidebarIconMap[iconName as keyof typeof sidebarIconMap] || IconBox
}

export function buildDefaultSidebarConfig(): SidebarConfig {
  const groups: SidebarGroup[] = []
  const hiddenItems: string[] = []
  let groupOrder = 0

  Object.entries(defaultMenus).forEach(([groupKey, items]) => {
    const groupId = groupKey
      .toLowerCase()
      .replace(/\./g, '-')
      .replace(/\s+/g, '-')
    const sidebarItems: SidebarItem[] = items.map((item, index) => {
      // Subgroup parents have no url of their own; slug the titleKey so two
      // parents in one group can never collide (shared ids made their
      // collapse state and React keys clash).
      const id = `${groupId}-${
        item.url
          ? item.url.replace(/[^a-zA-Z0-9]/g, '-')
          : `${item.titleKey.replace(/[^a-zA-Z0-9]/g, '-')}-sub`
      }`
      if (item.defaultHidden) {
        hiddenItems.push(id)
      }
      const base = {
        id,
        titleKey: item.titleKey,
        icon: getIconName(item.icon),
        visible: true,
        pinned: false,
        order: index,
      }
      if (item.children && item.children.length > 0) {
        return {
          ...base,
          type: 'link' as const,
          url: '',
          children: item.children.map((child, childIndex) => ({
            id: `${id}-${child.url.replace(/[^a-zA-Z0-9]/g, '-')}`,
            type: 'link' as const,
            titleKey: child.titleKey,
            url: child.url,
            icon: getIconName(child.icon),
            visible: true,
            pinned: false,
            order: childIndex,
          })),
        }
      }
      return {
        ...base,
        type: 'link' as const,
        url: item.url,
      }
    })

    groups.push({
      id: groupId,
      nameKey: groupKey,
      items: sidebarItems,
      visible: true,
      collapsed: false,
      order: groupOrder++,
    })
  })

  return {
    version: SIDEBAR_CONFIG_VERSION,
    groups,
    hiddenItems,
    pinnedItems: [],
    groupOrder: groups.map((g) => g.id),
    lastUpdated: Date.now(),
  }
}

export function migrateSidebarConfig(
  storedConfig: StoredSidebarConfig
): SidebarConfig {
  const removedItemIds = new Set(
    [
      ...storedConfig.groups,
      ...Object.values(storedConfig.pluginPreferences?.groups ?? {}),
    ].flatMap((group) =>
      (group.items ?? [])
        .filter(
          (item) =>
            'url' in item &&
            (item.url === '/gateways' || item.url === '/httproutes')
        )
        .map((item) => item.id)
    )
  )
  return {
    ...storedConfig,
    version: SIDEBAR_CONFIG_VERSION,
    hiddenItems: storedConfig.hiddenItems.filter(
      (id) => !removedItemIds.has(id)
    ),
    pinnedItems: storedConfig.pinnedItems.filter(
      (id) => !removedItemIds.has(id)
    ),
    pluginPreferences: storedConfig.pluginPreferences
      ? {
          ...storedConfig.pluginPreferences,
          groups: Object.fromEntries(
            Object.entries(storedConfig.pluginPreferences.groups).map(
              ([id, group]) => [
                id,
                {
                  ...group,
                  items: group.items?.filter(
                    (item) => !removedItemIds.has(item.id)
                  ),
                },
              ]
            )
          ),
        }
      : undefined,
    groups: storedConfig.groups.map((group) => {
      // Older configs could carry duplicate ids (e.g. two subgroup parents
      // built from an empty url); suffix duplicates so React keys, collapse
      // state and the customizer stay per-item.
      const seen = new Set<string>()
      return {
        ...group,
        items: group.items
          .filter((item) => !removedItemIds.has(item.id))
          .map((item): SidebarItem => {
            let migrated = migrateSidebarItem(item)
            if (seen.has(migrated.id)) {
              let suffix = 2
              while (seen.has(`${migrated.id}-${suffix}`)) suffix++
              migrated = { ...migrated, id: `${migrated.id}-${suffix}` }
            }
            seen.add(migrated.id)
            return migrated
          }),
      }
    }),
  }
}

function migrateSidebarItem(item: StoredSidebarItem): SidebarItem {
  const apiGroup = item.apiGroup || item.crdGroup
  const baseItem = {
    id: item.id,
    titleKey: item.titleKey,
    icon: item.icon,
    visible: item.visible,
    pinned: item.pinned,
    order: item.order,
  }

  const children = item.children?.map((child): SidebarItem =>
    migrateSidebarItem(child)
  )
  if (children && children.length > 0) {
    return {
      ...baseItem,
      type: 'link',
      url: item.url ?? '',
      // Children of a subgroup are always link items.
      children: children as Extract<SidebarItem, { type: 'link' }>[],
    }
  }

  if (apiGroup) {
    return {
      ...baseItem,
      type: 'apiGroup',
      apiGroup,
    }
  }

  return {
    ...baseItem,
    type:
      item.type === 'customResource' ||
      (!item.type && item.url?.startsWith('/crds/'))
        ? 'customResource'
        : 'link',
    url: item.url!,
  }
}
