import { useVersionInfo } from '@/lib/api'

const isRealCommit = (commitId: string | undefined) =>
  !!commitId && commitId !== 'unknown' && /^[0-9a-f]{7,40}$/i.test(commitId)

export function VersionInfo() {
  const { data: versionInfo } = useVersionInfo()

  if (!versionInfo) return null

  const commitShort = isRealCommit(versionInfo.commitId)
    ? versionInfo.commitId.slice(0, 7)
    : versionInfo.commitId
  // Link the commit only when the backend was built with a repo URL and a
  // real commit id; dev builds would otherwise link to a dead URL.
  const commitUrl =
    isRealCommit(versionInfo.commitId) && versionInfo.repoUrl
      ? `${versionInfo.repoUrl}/commit/${versionInfo.commitId}`
      : undefined

  return (
    <div className="text-[10px] text-muted-foreground/60 font-mono leading-none">
      v{versionInfo.version.replace(/^v/, '')} •{' '}
      {commitUrl ? (
        <button
          onClick={() => window.open(commitUrl, '_blank')}
          className="hover:text-primary/80 hover:underline transition-colors cursor-pointer"
          title={`View commit ${versionInfo.commitId}`}
        >
          {commitShort}
        </button>
      ) : (
        <span>{commitShort}</span>
      )}
    </div>
  )
}
