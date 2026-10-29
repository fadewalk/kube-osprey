package helmutil

import (
	"testing"

	"github.com/fadewalk/kube-osprey/pkg/model"
	"github.com/stretchr/testify/require"
)

func TestLoadRepositoryCatalogFromOCIRepository(t *testing.T) {
	repository := model.HelmRepository{
		Name: "kube-osprey",
		URL:  "oci://ghcr.io/fadewalk/charts/kube-osprey",
	}

	indexFile, err := LoadRepositoryCatalog(repository)
	require.NoError(t, err)

	entries := indexFile.Entries["kube-osprey"]
	require.NotEmpty(t, entries)
	require.Equal(t, "kube-osprey", entries[0].Name)
	require.NotEmpty(t, entries[0].Version)
	require.NotEmpty(t, entries[0].URLs)
	require.Equal(t, repository.URL+":"+entries[0].Version, entries[0].URLs[0])
}
