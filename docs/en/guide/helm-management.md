# Helm Management

KubeOsprey provides basic Helm management in the dashboard, covering chart discovery, release installation, upgrade, rollback, and uninstall.

## App Catalog

Open **App Catalog** from the sidebar to browse Helm charts.

KubeOsprey supports two chart sources:

- **Artifact Hub**: search public Helm charts.
- **Repositories**: browse Helm repositories managed in KubeOsprey.

::: tip
When using the Artifact Hub source, KubeOsprey may request Artifact Hub to fetch chart lists and chart details.
:::

::: warning
KubeOsprey only displays chart information and is not responsible for the chart content. Review chart details, templates, and values carefully before installing or upgrading.
:::

Users with the **admin** role can add or remove Helm repositories. Both classic (`https://`) and OCI (`oci://`) repositories are supported; an OCI URL must point at a single chart path without a tag or digest (for example `oci://registry.example.com/charts/app`), and its semver registry tags become the chart's versions. Removing a repository only removes it from KubeOsprey and does not uninstall existing releases.

OCI registries use HTTPS by default. For an HTTP-only registry, enable **Use HTTP (unencrypted)** when adding the repository; keep the URL in `oci://` form. KubeOsprey saves this setting for browsing, installation, upgrades, and automatic upgrades. When enabled, credentials and chart data are sent without encryption.

Open a chart to view its README, values, templates, and versions. If the chart package is available, you can install it directly from KubeOsprey.

## Helm Releases

Open **Helm Release** from the sidebar to view installed releases.

The release detail page shows release status, chart version, values, resources, history, logs, and rendered manifests.

KubeOsprey supports dry-run previews before install and upgrade. You can upgrade a release from the detail page, roll back from the history tab, or delete a release to uninstall it from the cluster.

## Permissions

Repository management requires the **admin** role. Release operations are controlled by KubeOsprey RBAC through the `helmrelease` resource (`get`, `create`, `update`, `delete`).

::: warning
Grant `helmrelease` permissions carefully. Helm actions are executed by KubeOsprey with the cluster credentials configured in KubeOsprey, so users with `helmrelease` `create`, `update`, or `delete` permissions may create, update, or delete chart-rendered resources even if their own Kubernetes RBAC permissions would not allow those direct operations.

The cluster credentials configured in KubeOsprey also need enough Kubernetes permissions for the resources rendered by the chart.
:::
