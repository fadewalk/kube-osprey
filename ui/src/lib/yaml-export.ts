import * as yaml from 'js-yaml'

interface YamlExportable {
  kind?: string
  metadata?: {
    name?: string
    namespace?: string
  }
}

function downloadTextFile(content: string, filename: string, mime: string) {
  const blob = new Blob([content], { type: mime })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

function asExportable(value: unknown): YamlExportable {
  if (value && typeof value === 'object') {
    return value as YamlExportable
  }
  return {}
}

export function downloadResourceYaml(value: unknown) {
  const obj = asExportable(value)
  const kind = obj.kind || 'resource'
  const name = obj.metadata?.name || 'unnamed'
  const namespace = obj.metadata?.namespace
  const filename = `${kind.toLowerCase()}-${namespace ? `${namespace}-` : ''}${name}.yaml`
  downloadTextFile(
    yaml.dump(value, { indent: 2 }),
    filename,
    'application/x-yaml'
  )
}

export function exportResourcesAsYaml(items: unknown[], resourceName: string) {
  if (items.length === 0) return
  const docs = items
    .map((item) => yaml.dump(item, { indent: 2 }).trimEnd())
    .join('\n---\n')
  const stamp = new Date().toISOString().replace(/[-:T]/g, '').slice(0, 14)
  const slug =
    resourceName
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/^-|-$/g, '') || 'resources'
  downloadTextFile(docs, `${slug}-export-${stamp}.yaml`, 'application/x-yaml')
}
