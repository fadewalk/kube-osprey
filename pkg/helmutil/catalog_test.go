package helmutil

import (
	"testing"

	"github.com/fadewalk/kube-osprey/pkg/model"
	"github.com/stretchr/testify/require"
)

func TestLoadRepositoryCatalogFromOCIRepository(t *testing.T) {
	repository := model.HelmRepository{
		Name: "kite",
		// TODO(swatch): switch to oci://ghcr.io/fadewalk/charts/kube-osprey once
		// the first renamed chart release exists.
		URL: "oci://ghcr.io/fadewalk/charts/kite",
	}

	indexFile, err := LoadRepositoryCatalog(repository)
	require.NoError(t, err)

	entries := indexFile.Entries["kite"]
	require.NotEmpty(t, entries)
	require.Equal(t, "kite", entries[0].Name)
	require.NotEmpty(t, entries[0].Version)
	require.NotEmpty(t, entries[0].URLs)
	require.Equal(t, repository.URL+":"+entries[0].Version, entries[0].URLs[0])
}
