package version

import (
	"net/http"

	"github.com/fadewalk/kube-osprey/pkg/common"
	"github.com/gin-gonic/gin"
)

var (
	Version   = "dev"
	BuildDate = "unknown"
	CommitID  = "unknown"
	// RepoURL is the browsable base URL of the repository this binary was
	// built from (injected at link time); empty disables commit links.
	RepoURL = ""
)

type VersionInfo struct {
	Version   string `json:"version"`
	BuildDate string `json:"buildDate"`
	CommitID  string `json:"commitId"`
	RepoURL   string `json:"repoUrl"`
	HasNew    bool   `json:"hasNewVersion"`
	Release   string `json:"releaseUrl"`
}

func GetVersion(c *gin.Context) {
	versionInfo := VersionInfo{
		Version:   Version,
		BuildDate: BuildDate,
		CommitID:  CommitID,
		RepoURL:   RepoURL,
	}

	if common.EnableVersionCheck {
		r := checkForUpdate(c.Request.Context(), Version)
		versionInfo.HasNew = r.hasNew
		if versionInfo.HasNew {
			versionInfo.Release = r.releaseURL
		}
	}
	c.JSON(http.StatusOK, versionInfo)
}
