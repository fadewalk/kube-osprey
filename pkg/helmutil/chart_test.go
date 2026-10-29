package helmutil

import (
	"testing"

	"github.com/fadewalk/kube-osprey/pkg/model"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v4/pkg/getter"
	repo "helm.sh/helm/v4/pkg/repo/v1"
)

func TestLoadRepositoryArchiveFromRepository(t *testing.T) {
	repository := model.HelmRepository{
		Name: "kite",
		URL:  "https://charts.kitehq.dev",
	}
	chartRepository, err := repo.NewChartRepository(&repo.Entry{
		Name: repository.Name,
		URL:  repository.URL,
	}, getter.Getters())
	require.NoError(t, err)
	chartRepository.CachePath = t.TempDir()

	indexPath, err := chartRepository.DownloadIndexFile()
	require.NoError(t, err)
	indexFile, err := repo.LoadIndexFile(indexPath)
	require.NoError(t, err)

	entries := indexFile.Entries["kite"]
	require.NotEmpty(t, entries)

	loadedChart, err := LoadRepositoryArchive(repository, entries[0])
	require.NoError(t, err)
	require.Equal(t, "kite", loadedChart.Metadata.Name)
	require.Equal(t, entries[0].Version, loadedChart.Metadata.Version)
}

func TestLoadArchiveFromOCIRepository(t *testing.T) {
	loadedChart, err := LoadArchive("oci://ghcr.io/fadewalk/charts/kube-osprey", nil)
	require.NoError(t, err)
	require.Equal(t, "kube-osprey", loadedChart.Metadata.Name)
	require.NotEmpty(t, loadedChart.Metadata.Version)
}
